package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"

	kubevirtcorev1 "kubevirt.io/client-go/apis/core/v1"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kvtutil "kubevirt.io/kubevirt/tests/util"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const nodePlacementSamplingSize = 20

var _ = Describe("[rfe_id:4356][crit:medium][vendor:cnv-qe@redhat.com][level:system]Node Placement", func() {

	var workloadsNode *v1.Node

	tests.FlagParse()
	client, err := kubecli.GetKubevirtClient()
	kvtutil.PanicOnError(err)

	workloadsNodes, err := client.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{
		LabelSelector: "node.kubernetes.io/hco-test-node-type==workloads",
	})
	kvtutil.PanicOnError(err)

	if workloadsNodes != nil && len(workloadsNodes.Items) == 1 {
		workloadsNode = &workloadsNodes.Items[0]
		fmt.Fprintf(GinkgoWriter, "Found Workloads Node. Node name: %s; node labels:\n", workloadsNode.Name)
		w := json.NewEncoder(GinkgoWriter)
		w.SetIndent("", "  ")
		_ = w.Encode(workloadsNode.Labels)
	}

	BeforeEach(func() {
		if workloadsNode == nil {
			Skip("Skipping Node Placement tests")
		}
		tests.BeforeEach()
	})

	Context("validate node placement in workloads nodes", func() {
		It("[test_id:5677] all expected 'workloads' pod must be on infra node", func() {
			expectedWorkloadsPods := map[string]bool{
				"bridge-marker": false,
				"cni-plugins":   false,
				//"kube-multus":     false,
				"nmstate-handler": false,
				"ovs-cni-marker":  false,
				"virt-handler":    false,
			}

			By("Getting Network Addons Configs")
			cnaoCR := getNetworkAddonsConfigs(client)
			if cnaoCR.Spec.Ovs == nil {
				delete(expectedWorkloadsPods, "ovs-cni-marker")
			}

			By("Listing pods in infra node")
			pods := listPodsInNode(client, workloadsNode.Name)

			By("Collecting nodes of pods")
			updatePodAssignments(pods, expectedWorkloadsPods, "workload", workloadsNode.Name)

			By("Verifying that all expected workload pods exist in workload nodes")
			Expect(expectedWorkloadsPods).ToNot(ContainElement(false))
		})
	})

	Context("validate node placement on infra nodes", func() {
		It("[test_id:5678] all expected 'infra' pod must be on infra node", func() {
			expectedInfraPods := map[string]bool{
				"cdi-apiserver":   false,
				"cdi-controller":  false,
				"cdi-uploadproxy": false,
				"manager":         false,
				"nmstate-webhook": false,
				"virt-api":        false,
				"virt-controller": false,
			}

			By("Listing infra nodes")
			infraNodes := listInfraNodes(client)

			for _, node := range infraNodes.Items {
				By("Listing pods in " + node.Name)
				pods := listPodsInNode(client, node.Name)

				By("Collecting nodes of pods")
				updatePodAssignments(pods, expectedInfraPods, "infra", node.Name)
			}

			By("Verifying that all expected infra pods exist in infra nodes")
			Expect(expectedInfraPods).ToNot(ContainElement(false))
		})
	})

	Context("validate node placement for vmi", func() {
		It("[test_id:5679] should create, verify and delete VMIs with correct node placements", func() {
			// we are iterating many times to ensure that the vmi is not scheduled to the expected node by chance
			// this test may give false positive result. more iteration can give more accuracy.
			for i := 0; i < nodePlacementSamplingSize; i++ {
				fmt.Fprintf(GinkgoWriter, "Run %d/%d\n", i+1, nodePlacementSamplingSize)
				vmiName := verifyVMICreation(client)
				vmi := verifyVMIRunning(client, vmiName)
				verifyVMINodePlacement(vmi, workloadsNode.Name)
				verifyVMIDeletion(client, vmiName)
			}
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

func listPodsInNode(client kubecli.KubevirtClient, nodeName string) *v1.PodList {
	pods, err := client.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.TODO(), k8smetav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

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
	_ = apis.AddToScheme(s)
	s.AddKnownTypes(networkaddonsv1.SchemeGroupVersion)

	err := client.RestClient().Get().
		Resource("networkaddonsconfigs").
		Name("cluster").
		AbsPath("/apis", networkaddonsv1.SchemeGroupVersion.Group, networkaddonsv1.SchemeGroupVersion.Version).
		Timeout(10 * time.Second).
		Do(context.TODO()).Into(&cnaoCR)

	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return &cnaoCR
}

func verifyVMINodePlacement(vmi *kubevirtcorev1.VirtualMachineInstance, workloadNodeName string) {
	By("Verifying node placement of VMI")
	ExpectWithOffset(1, vmi.Labels["kubevirt.io/nodeName"]).Should(Equal(workloadNodeName))
	fmt.Fprintf(GinkgoWriter, "The VMI is running on the right node: %s\n", workloadNodeName)
}
