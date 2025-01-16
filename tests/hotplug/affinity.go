package hotplug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]VM Affinity", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateRolloutStrategy, Serial, func() {
	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := libkubevirt.GetCurrentKv(virtClient)
		kv.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyLiveUpdate)
		testsuite.UpdateKubeVirtConfigValue(kv.Spec.Configuration)
	})

	Context("Updating VMs node affinity", func() {
		patchVMNodeSelector := func(newNodeSelectorMap map[string]string, op string, vmName string, vmNamespace string) {

			newNodeSelectorJson, err := json.Marshal(newNodeSelectorMap)
			Expect(err).ToNot(HaveOccurred())

			value := ""
			if op != patch.PatchRemoveOp {
				value = fmt.Sprintf(`, "value":%s`, newNodeSelectorJson)
			}
			patchData1Str := fmt.Sprintf(`[ {"op":"%s","path":"/spec/template/spec/nodeSelector"%s} ]`, op, value)
			patchData1 := []byte(patchData1Str)
			_, err = virtClient.VirtualMachine(vmNamespace).Patch(context.Background(), vmName, types.JSONPatchType, patchData1, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		validateNodeSelector := func(expectedMap, vmMap map[string]string) bool {
			for key, value := range expectedMap {
				if val, ok := vmMap[key]; !ok || val != value {
					return false
				}
			}
			return true
		}

		generateNodeAffinity := func(nodeName string) *k8sv1.Affinity {
			return &k8sv1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
						NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
							{
								MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{Key: k8sv1.LabelHostname, Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodeName}},
								},
							},
						},
					},
				},
			}
		}

		patchVMAffinity := func(vmAffinity *k8sv1.Affinity, op string, vmName string, vmNamespace string) {
			newAffinityJson, err := json.Marshal(vmAffinity)
			Expect(err).ToNot(HaveOccurred())

			value := ""
			if op != patch.PatchRemoveOp {
				value = fmt.Sprintf(`, "value":%s`, newAffinityJson)
			}
			patchData1Str := fmt.Sprintf(`[ {"op":"%s","path":"/spec/template/spec/affinity"%s} ]`, op, value)
			patchData1 := []byte(patchData1Str)
			_, err = virtClient.VirtualMachine(vmNamespace).Patch(context.Background(), vmName, types.JSONPatchType, patchData1, k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:11208]should successfully update node selector", func() {

			By("Creating a running VM")
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking(), libvmi.WithCPUCount(1, 2, 1))
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Adding node selector")
			vmNodeSelector := map[string]string{k8sv1.LabelOSStable: "not-existing-os"}
			patchVMNodeSelector(vmNodeSelector, patch.PatchAddOp, vm.Name, vm.Namespace)

			By("Ensuring the VMI has added node selector")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				if vmi.Spec.NodeSelector == nil {
					return false
				}
				return validateNodeSelector(vmNodeSelector, vmi.Spec.NodeSelector)
			}, 240*time.Second, time.Second).Should(BeTrue())

			By("Updating node selector")
			vmNodeSelector = map[string]string{k8sv1.LabelOSStable: "not-existing-os-updated"}
			patchVMNodeSelector(vmNodeSelector, patch.PatchReplaceOp, vm.Name, vm.Namespace)

			By("Ensuring the VMI has the updated node selector")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				if vmi.Spec.NodeSelector == nil {
					return false
				}
				return validateNodeSelector(vmNodeSelector, vmi.Spec.NodeSelector)
			}, 240*time.Second, time.Second).Should(BeTrue())

			By("Removing node selector")
			vmNodeSelector = map[string]string{}
			patchVMNodeSelector(vmNodeSelector, patch.PatchRemoveOp, vm.Name, vm.Namespace)

			By("Ensuring the VMI has removed the node selector")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				if vmi.Spec.NodeSelector == nil {
					return false
				}
				return validateNodeSelector(vmNodeSelector, vmi.Spec.NodeSelector)
			}, 240*time.Second, time.Second).Should(BeTrue())

		})
		It("[test_id:11209]should successfully update node affinity", func() {

			By("Creating a running VM")
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Adding Affinity")
			vmAffinity := generateNodeAffinity("fakeNode_1")
			patchVMAffinity(vmAffinity, patch.PatchAddOp, vm.Name, vm.Namespace)

			By("Ensuring the VMI has added affinity")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				if vmi.Spec.Affinity == nil {
					return false
				}
				return equality.Semantic.DeepEqual(vmAffinity, vmi.Spec.Affinity)
			}, 240*time.Second, time.Second).Should(BeTrue())

			By("Updating node affinity")
			vmAffinity = generateNodeAffinity("fakeNode_2")
			patchVMAffinity(vmAffinity, patch.PatchReplaceOp, vm.Name, vm.Namespace)

			By("Ensuring the VMI has the updated node affinity")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				if vmi.Spec.Affinity == nil {
					return false
				}
				return equality.Semantic.DeepEqual(vmAffinity, vmi.Spec.Affinity)
			}, 240*time.Second, time.Second).Should(BeTrue())

			By("Removing node affinity")
			emptyAffinity := k8sv1.Affinity{}
			patchVMAffinity(&emptyAffinity, patch.PatchRemoveOp, vm.Name, vm.Namespace)

			By("Ensuring the VMI has removed the node affinity")
			Eventually(func() bool {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return vmi.Spec.Affinity == nil
			}, 240*time.Second, time.Second).Should(BeTrue())

		})

	})

	Context("Machine Type Affinity", func() {

		const unsupportedMachineType = "pc-q35-test1.2.3"

		BeforeEach(func() {
			By("Verifying that no nodes have the unsupported machine type label")
			unsupportedLabel := v1.SupportedMachineTypeLabel + unsupportedMachineType
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			for _, node := range nodes.Items {
				Expect(node.Labels).ToNot(HaveKeyWithValue(unsupportedLabel, "true"))
				Expect(node.Labels).ToNot(HaveKeyWithValue(unsupportedLabel, "deprecated"))
			}
		})

		It("should not start a pod for a VMI with an old machine type", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vmi.Spec.Domain.Machine = &v1.Machine{Type: unsupportedMachineType}

			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, k8smetav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			var podList *k8sv1.PodList
			Eventually(func() ([]k8sv1.Pod, error) {
				podList, err = virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), k8smetav1.ListOptions{
					LabelSelector: fmt.Sprintf("%s=%s", v1.CreatedByLabel, string(vmi.UID)),
				})
				Expect(err).NotTo(HaveOccurred())
				return podList.Items, nil
			}, 10*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "Pod should eventually be created for the VMI")
			pod := podList.Items[0]

			By("Validating the pod's affinity contains the machine type label")
			expectPodNodeSelectorContainsMachineType(pod, unsupportedMachineType)

			By("Validating the pod is in Pending state with Unschedulable condition")
			Expect(pod.Status.Phase).To(Equal(k8sv1.PodPending), "Pod should remain in Pending state due to node affinity issues")
			expectPodHasUnschedulableConditionForNodeSelector(pod)
		})

		It("should fail migration if no node supports the VMI machine type", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
			vmi.Namespace = testsuite.GetTestNamespace(vmi)

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)
			machineType := vmi.Status.Machine.Type
			Expect(machineType).ToNot(BeEmpty(), "VMI should have a valid machine type in its status")

			By("Fetching all nodes in the cluster")
			nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), k8smetav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(nodeList.Items).ToNot(BeEmpty())

			By("Patching all nodes to remove the supported machine type label")
			for _, node := range nodeList.Items {
				nodeName := node.Name
				libinfra.ExpectStoppingNodeLabellerToSucceed(nodeName, virtClient)

				labelKey := strings.Replace(v1.SupportedMachineTypeLabel+machineType, "/", "~1", 1)
				patchPayload := fmt.Sprintf(`[{"op": "remove", "path": "/metadata/labels/%s"}]`, labelKey)
				_, err := virtClient.CoreV1().Nodes().Patch(context.Background(), nodeName, types.JSONPatchType, []byte(patchPayload), k8smetav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			DeferCleanup(func() {
				By("Restoring the machine type label for all nodes")
				for _, node := range nodeList.Items {
					libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, virtClient)
				}
			})

			By("Initiating a migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			createdMigration := libmigration.RunMigration(virtClient, migration)

			libmigration.WaitForMigrationPhase(virtClient, createdMigration.Namespace, createdMigration.Name, v1.MigrationScheduling, 30*time.Second)

			By("Fetching the target pod for the migration")
			targetPodList, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), k8smetav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", v1.MigrationJobLabel, string(createdMigration.UID))})
			Expect(err).ToNot(HaveOccurred())
			Expect(targetPodList.Items).ToNot(BeEmpty(), "Target pod for migration should be created")

			targetPod := targetPodList.Items[0]
			Expect(targetPod).ToNot(BeNil(), "Target pod should be found")

			By("Validating the pod's affinity")
			expectPodNodeSelectorContainsMachineType(targetPod, machineType)

			By("Validating the pod has an unschedulable condition")
			expectPodHasUnschedulableConditionForNodeSelector(targetPod)

			libmigration.EnsureMigrationRemainsInPhase(virtClient, createdMigration.Namespace, createdMigration.Name, v1.MigrationScheduling, 60*time.Second)
		})
	})
})

func expectPodNodeSelectorContainsMachineType(pod k8sv1.Pod, machineType string) {
	Expect(pod.Spec.NodeSelector).ToNot(BeNil(), "Pod NodeSelector should be defined")

	machineTypeKey := v1.SupportedMachineTypeLabel + machineType

	value, exists := pod.Spec.NodeSelector[machineTypeKey]
	Expect(exists).To(BeTrue(), fmt.Sprintf("Pod %s NodeSelector should include machine type label %s", pod.Name, machineTypeKey))
	Expect(value).To(BeEquivalentTo("true"), fmt.Sprintf("Pod NodeSelector should set machine type label %s to 'true'", machineTypeKey))
}

func expectPodHasUnschedulableConditionForNodeSelector(pod k8sv1.Pod) {
	foundUnschedulable := false
	for _, condition := range pod.Status.Conditions {
		if condition.Type == k8sv1.PodScheduled && condition.Status == k8sv1.ConditionFalse && condition.Reason == "Unschedulable" {
			foundUnschedulable = true
			break
		}
	}
	Expect(foundUnschedulable).To(BeTrue(), "Pod should have an Unschedulable condition due to node selector constraints")
}
