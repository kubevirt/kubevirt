package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/kubevirt/tests/flags"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kvtutil "kubevirt.io/kubevirt/tests/util"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
	"kubevirt.io/client-go/kubecli"
)

const (
	hcoLabel  = "node.kubernetes.io/hco-test-node-type"
	infra     = "infra"
	workloads = "workloads"
)

var _ = Describe("[rfe_id:4356][crit:medium][vendor:cnv-qe@redhat.com][level:system]Node Placement", Ordered, func() {
	var workloadsNode *v1.Node
	ctx := context.TODO()
	tests.FlagParse()
	client, err := kubecli.GetKubevirtClient()
	kvtutil.PanicOnError(err)
	hco := &hcov1beta1.HyperConverged{}
	var originalInfraSpec hcov1beta1.HyperConvergedConfig
	var originalWorkloadSpec hcov1beta1.HyperConvergedConfig

	BeforeAll(func() {
		nodes, err := client.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{LabelSelector: "node-role.kubernetes.io/control-plane!="})
		kvtutil.PanicOnError(err)
		if len(nodes.Items) < 2 {
			Skip("Skipping Node Placement tests due to insufficient cluster nodes")
		}
		// Label all but first node with "node.kubernetes.io/hco-test-node-type=infra"
		// We are doing this to remove dependency of this Describe block on a shell script that
		// labels the nodes this way
		Eventually(func(g Gomega) {
			for _, node := range nodes.Items[:len(nodes.Items)-1] {
				done, err := setHcoNodeTypeLabel(client, &node, infra)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(done).To(BeTrue())
			}
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
		// Label the last node with "node.kubernetes.io/hco-test-node-type=workloads"
		Eventually(func(g Gomega) {
			done, err := setHcoNodeTypeLabel(client, &nodes.Items[len(nodes.Items)-1], workloads)
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(done).To(BeTrue())
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

		// modify the HCO CR to use the labels we just applied to the nodes
		originalHco := tests.GetHCO(ctx, client)
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

		tests.UpdateHCORetry(ctx, client, hco)

		workloadsNodes, err := client.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{
			LabelSelector: "node.kubernetes.io/hco-test-node-type==workloads",
		})
		kvtutil.PanicOnError(err)
		Expect(workloadsNodes.Items).To(HaveLen(1))

		workloadsNode = &workloadsNodes.Items[0]
		GinkgoWriter.Printf("Found Workloads Node. Node name: %s; node labels:\n", workloadsNode.Name)
		w := json.NewEncoder(GinkgoWriter)
		w.SetIndent("", "  ")
		_ = w.Encode(workloadsNode.Labels)
	})

	AfterAll(func() {
		// undo the modification to HCO CR done in BeforeAll stage
		modifiedHco := tests.GetHCO(ctx, client)

		modifiedHco.DeepCopyInto(hco)
		hco.Spec.Infra = originalInfraSpec
		hco.Spec.Workloads = originalWorkloadSpec

		tests.UpdateHCORetry(ctx, client, hco)

		// unlabel the nodes
		nodes, err := client.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{LabelSelector: hcoLabel})
		kvtutil.PanicOnError(err)
		// wrap unlabelling in Eventually because for resourceVersion errors
		Eventually(func(g Gomega) {
			for _, node := range nodes.Items {
				n := &node
				labels := n.GetLabels()
				delete(labels, hcoLabel)
				n, err = client.CoreV1().Nodes().Get(context.TODO(), n.Name, k8smetav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				n.SetLabels(labels)
				_, err = client.CoreV1().Nodes().Update(context.TODO(), n, k8smetav1.UpdateOptions{})
				g.Expect(err).ToNot(HaveOccurred())
			}
		}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
	})

	BeforeEach(func() {
		tests.BeforeEach()
	})

	Context("validate node placement in workloads nodes", func() {
		It("[test_id:5677] all expected 'workloads' pod must be on infra node", func() {
			expectedWorkloadsPods := map[string]bool{
				"bridge-marker": false,
				"cni-plugins":   false,
				// "kube-multus":     false,
				"ovs-cni-marker": false,
				"virt-handler":   false,
				"secondary-dns":  false,
			}

			By("Getting Network Addons Configs")
			cnaoCR := getNetworkAddonsConfigs(client)
			if cnaoCR.Spec.Ovs == nil {
				delete(expectedWorkloadsPods, "ovs-cni-marker")
			}
			if cnaoCR.Spec.KubeSecondaryDNS == nil {
				delete(expectedWorkloadsPods, "secondary-dns")
			}

			Eventually(func(g Gomega) {
				By("Listing pods in infra node")
				pods := listPodsInNode(g, client, workloadsNode.Name)

				By("Collecting nodes of pods")
				updatePodAssignments(pods, expectedWorkloadsPods, "workload", workloadsNode.Name)

				By("Verifying that all expected workload pods exist in workload nodes")
				g.Expect(expectedWorkloadsPods).ToNot(ContainElement(false))
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
		})
	})

	Context("validate node placement on infra nodes", func() {
		It("[test_id:5678] all expected 'infra' pod must be on infra node", func() {
			expectedInfraPods := map[string]bool{
				"cdi-apiserver":   false,
				"cdi-controller":  false,
				"cdi-uploadproxy": false,
				"manager":         false,
				"virt-api":        false,
				"virt-controller": false,
			}

			Eventually(func(g Gomega) {
				By("Listing infra nodes")
				infraNodes := listInfraNodes(client)

				for _, node := range infraNodes.Items {
					By("Listing pods in " + node.Name)
					pods := listPodsInNode(g, client, node.Name)

					By("Collecting nodes of pods")
					updatePodAssignments(pods, expectedInfraPods, "infra", node.Name)
				}

				By("Verifying that all expected infra pods exist in infra nodes")
				g.Expect(expectedInfraPods).ToNot(ContainElement(false))
			}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
		})
	})
})

func updatePodAssignments(pods *v1.PodList, podMap map[string]bool, nodeType string, nodeName string) {
	for _, pod := range pods.Items {
		podName := pod.Spec.Containers[0].Name
		fmt.Fprintf(GinkgoWriter, "Found %s pod '%s' in the '%s' node %s\n", podName, pod.Name, nodeType, nodeName)
		if found, ok := podMap[podName]; ok {
			if !found {
				podMap[podName] = true
			}
		}
	}
}

func listPodsInNode(g Gomega, client kubecli.KubevirtClient, nodeName string) *v1.PodList {
	pods, err := client.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.TODO(), k8smetav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	g.ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return pods
}

func listInfraNodes(client kubecli.KubevirtClient) *v1.NodeList {
	infraNodes, err := client.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{
		LabelSelector: "node.kubernetes.io/hco-test-node-type==infra",
	})
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	return infraNodes
}

func getNetworkAddonsConfigs(client kubecli.KubevirtClient) *networkaddonsv1.NetworkAddonsConfig {
	var cnaoCR networkaddonsv1.NetworkAddonsConfig

	s := scheme.Scheme
	_ = networkaddonsv1.AddToScheme(s)
	s.AddKnownTypes(networkaddonsv1.GroupVersion)

	ExpectWithOffset(1, client.RestClient().Get().
		Resource("networkaddonsconfigs").
		Name("cluster").
		AbsPath("/apis", networkaddonsv1.GroupVersion.Group, networkaddonsv1.GroupVersion.Version).
		Timeout(10*time.Second).
		Do(context.TODO()).Into(&cnaoCR)).To(Succeed())

	return &cnaoCR
}

func setHcoNodeTypeLabel(client kubecli.KubevirtClient, node *v1.Node, value string) (bool, error) {
	labels := node.GetLabels()
	labels[hcoLabel] = value
	node, err := client.CoreV1().Nodes().Get(context.TODO(), node.Name, k8smetav1.GetOptions{})
	if err != nil {
		return false, err
	}
	node.SetLabels(labels)
	_, err = client.CoreV1().Nodes().Update(context.TODO(), node, k8smetav1.UpdateOptions{})
	if err != nil {
		return false, err
	}
	return true, nil
}
