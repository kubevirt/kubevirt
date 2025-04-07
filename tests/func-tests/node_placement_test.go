package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	hcoLabel  = "node.kubernetes.io/hco-test-node-type"
	infra     = "infra"
	workloads = "workloads"
)

var _ = Describe("[rfe_id:4356][crit:medium][vendor:cnv-qe@redhat.com][level:system]Node Placement", Ordered, Serial, Label(tests.HighlyAvailableClusterLabel, "nodePlacement"), func() {
	tests.FlagParse()
	hco := &hcov1beta1.HyperConverged{}
	var (
		workloadsNode        *v1.Node
		originalInfraSpec    hcov1beta1.HyperConvergedConfig
		originalWorkloadSpec hcov1beta1.HyperConvergedConfig
		cli                  client.Client
		cliSet               *kubernetes.Clientset
		workerNodes          *v1.NodeList
	)

	BeforeAll(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()
		cliSet = tests.GetK8sClientSet()

		workerNodes = listNodesByLabels(ctx, cliSet, "node-role.kubernetes.io/worker")
		tests.FailIfSingleNodeCluster(len(workerNodes.Items) < 2)

		// Label all but the last node with "node.kubernetes.io/hco-test-node-type=infra"
		Eventually(func(g Gomega, ctx context.Context) {
			for _, node := range workerNodes.Items[:len(workerNodes.Items)-1] {
				done, err := setHcoNodeTypeLabel(ctx, cliSet, &node, infra)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(done).To(BeTrue())
			}
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
		// Label the last node with "node.kubernetes.io/hco-test-node-type=workloads"
		Eventually(func(g Gomega, ctx context.Context) {
			done, err := setHcoNodeTypeLabel(ctx, cliSet, &workerNodes.Items[len(workerNodes.Items)-1], workloads)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(done).To(BeTrue())
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())

		// modify the HCO CR to use the labels we just applied to the nodes
		originalHco := tests.GetHCO(ctx, cli)
		originalHco.DeepCopyInto(hco)
		originalInfraSpec = originalHco.Spec.Infra
		originalWorkloadSpec = originalHco.Spec.Workloads

		// modify the "infra" and "workloads" keys
		infraVal := hcov1beta1.HyperConvergedConfig{
			NodePlacement: &sdkapi.NodePlacement{
				NodeSelector: map[string]string{hcoLabel: infra},
			},
		}
		workloadsVal := hcov1beta1.HyperConvergedConfig{
			NodePlacement: &sdkapi.NodePlacement{
				NodeSelector: map[string]string{hcoLabel: workloads},
			},
		}

		hco.Spec.Infra = infraVal
		hco.Spec.Workloads = workloadsVal

		tests.UpdateHCORetry(ctx, cli, hco)

		const hcoSelector = hcoLabel + "==workloads"
		workloadsNodes := listNodesByLabels(ctx, cliSet, hcoSelector)
		Expect(workloadsNodes.Items).To(HaveLen(1))

		workloadsNode = &workloadsNodes.Items[0]
		GinkgoWriter.Printf("Found Workloads Node. Node name: %s; node labels:\n", workloadsNode.Name)
		w := json.NewEncoder(GinkgoWriter)
		w.SetIndent("", "  ")
		_ = w.Encode(workloadsNode.Labels)
	})

	AfterAll(func(ctx context.Context) {
		// undo the modification to HCO CR done in BeforeAll stage
		modifiedHco := tests.GetHCO(ctx, cli)

		modifiedHco.DeepCopyInto(hco)
		hco.Spec.Infra = originalInfraSpec
		hco.Spec.Workloads = originalWorkloadSpec

		tests.UpdateHCORetry(ctx, cli, hco)

		// unlabel the nodes
		nodes := listNodesByLabels(ctx, cliSet, hcoLabel)

		// wrap unlabelling in Eventually because for resourceVersion errors
		Eventually(func(g Gomega, ctx context.Context) {
			for _, node := range nodes.Items {
				n := &node
				labels := n.GetLabels()
				delete(labels, hcoLabel)
				n, err := cliSet.CoreV1().Nodes().Get(ctx, n.Name, k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				n.SetLabels(labels)
				_, err = cliSet.CoreV1().Nodes().Update(ctx, n, k8smetav1.UpdateOptions{})
				g.Expect(err).ToNot(HaveOccurred())
			}
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())

		By("make sure all the virt-handler pods are running again")
		Eventually(func(g Gomega, ctx context.Context) {
			labelSelector := "kubevirt.io=virt-handler"
			pods, err := cliSet.CoreV1().Pods(tests.InstallNamespace).List(ctx, k8smetav1.ListOptions{LabelSelector: labelSelector})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(len(pods.Items)).To(BeNumerically(">=", len(workerNodes.Items)))

			for _, pod := range pods.Items {
				podReady := false
				for _, cond := range pod.Status.Conditions {
					if cond.Type == v1.PodReady {
						g.Expect(cond.Status).To(Equal(v1.ConditionTrue))
						podReady = true
						break
					}
				}

				g.Expect(podReady).To(BeTrue())
			}
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
	})

	BeforeEach(func(ctx context.Context) {
		tests.BeforeEach(ctx)
	})

	Context("validate node placement in workloads nodes", func() {
		It("[test_id:5677] all expected 'workloads' pod must be on workloads node", Label("test_id:5677"), func(ctx context.Context) {
			expectedWorkloadsPods := map[string]bool{
				"bridge-marker":  false,
				"cni-plugins":    false,
				"ovs-cni-marker": false,
				"virt-handler":   false,
				"secondary-dns":  false,
			}

			By("Getting Network Addons Configs")
			cnaoCR := getNetworkAddonsConfigs(ctx, cliSet)
			if cnaoCR.Spec.Ovs == nil {
				delete(expectedWorkloadsPods, "ovs-cni-marker")
			}
			if cnaoCR.Spec.KubeSecondaryDNS == nil {
				delete(expectedWorkloadsPods, "secondary-dns")
			}

			Eventually(func(g Gomega, ctx context.Context) {
				By("Listing pods in infra node")
				pods := listPodsInNode(ctx, g, cliSet, workloadsNode.Name)

				By("Collecting nodes of pods")
				updatePodAssignments(pods, expectedWorkloadsPods, "workload", workloadsNode.Name)

				By("Verifying that all expected workload pods exist in workload nodes")
				g.Expect(expectedWorkloadsPods).ToNot(ContainElement(false))
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
		})
	})

	Context("validate node placement on infra nodes", func() {
		It("[test_id:5678] all expected 'infra' pod must be on infra node", Label("test_id:5678"), func(ctx context.Context) {
			expectedInfraPods := map[string]bool{
				"cdi-apiserver":       false,
				"cdi-deployment":      false,
				"cdi-uploadproxy":     false,
				"kubemacpool":         false,
				"virt-api":            false,
				"virt-controller":     false,
				"virt-exportproxy":    false,
				"ipam-virt-workloads": false,
			}

			Eventually(func(g Gomega, ctx context.Context) {
				By("Listing infra nodes")
				infraNodes := listInfraNodes(ctx, cliSet)

				for _, node := range infraNodes.Items {
					By("Listing pods in " + node.Name)
					pods := listPodsInNode(ctx, g, cliSet, node.Name)

					By("Collecting nodes of pods")
					updatePodAssignments(pods, expectedInfraPods, "infra", node.Name)
				}

				By("Verifying that all expected infra pods exist in infra nodes")
				g.Expect(expectedInfraPods).ToNot(ContainElement(false))
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).WithContext(ctx).Should(Succeed())
		})
	})
})

func updatePodAssignments(pods []v1.Pod, podMap map[string]bool, nodeType string, nodeName string) {
	for _, pod := range pods {
		podName := ""
		switch pod.Labels["app.kubernetes.io/managed-by"] {
		case "cdi-operator":
			podName = pod.Labels["cdi.kubevirt.io"]

		case "cnao-operator":
			podName = pod.Labels["app"]

		case "virt-operator":
			podName = pod.Labels["kubevirt.io"]

		default:
			continue
		}

		GinkgoWriter.Printf("Found %s pod %q in the %s node %s\n", podName, pod.Name, nodeType, nodeName)

		if found, ok := podMap[podName]; ok && !found {
			podMap[podName] = true
		}
	}
}

func listPodsInNode(ctx context.Context, g Gomega, cli *kubernetes.Clientset, nodeName string) []v1.Pod {
	pods, err := cli.CoreV1().Pods(tests.InstallNamespace).List(ctx, k8smetav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s,status.phase=Running", nodeName),
	})
	g.ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return pods.Items
}

func listInfraNodes(ctx context.Context, cli *kubernetes.Clientset) *v1.NodeList {
	infraNodes, err := cli.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{
		LabelSelector: "node.kubernetes.io/hco-test-node-type==infra",
	})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return infraNodes
}

func getNetworkAddonsConfigs(ctx context.Context, cli *kubernetes.Clientset) *networkaddonsv1.NetworkAddonsConfig {
	var cnaoCR networkaddonsv1.NetworkAddonsConfig

	s := scheme.Scheme
	_ = networkaddonsv1.AddToScheme(s)
	s.AddKnownTypes(networkaddonsv1.GroupVersion)

	ExpectWithOffset(1, cli.RESTClient().Get().
		Resource("networkaddonsconfigs").
		Name("cluster").
		AbsPath("/apis", networkaddonsv1.GroupVersion.Group, networkaddonsv1.GroupVersion.Version).
		Timeout(10*time.Second).
		Do(ctx).Into(&cnaoCR)).To(Succeed())

	return &cnaoCR
}

func setHcoNodeTypeLabel(ctx context.Context, cli *kubernetes.Clientset, node *v1.Node, value string) (bool, error) {
	labels := node.GetLabels()
	labels[hcoLabel] = value
	node, err := cli.CoreV1().Nodes().Get(ctx, node.Name, k8smetav1.GetOptions{})
	if err != nil {
		return false, err
	}
	node.SetLabels(labels)
	_, err = cli.CoreV1().Nodes().Update(ctx, node, k8smetav1.UpdateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func listNodesByLabels(ctx context.Context, cli *kubernetes.Clientset, labelSelector string) *v1.NodeList {
	var nodes *v1.NodeList
	Eventually(func(ctx context.Context) error {
		var err error
		nodes, err = cli.CoreV1().Nodes().List(ctx, k8smetav1.ListOptions{LabelSelector: labelSelector})
		return err
	}).WithTimeout(10 * time.Second).
		WithPolling(time.Second).
		WithOffset(1).
		WithContext(ctx).
		Should(Succeed())

	return nodes
}
