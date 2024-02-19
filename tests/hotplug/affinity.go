package hotplug

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/libnet"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"

	v1 "kubevirt.io/api/core/v1"

	util2 "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/libvmifact"

	"kubevirt.io/kubevirt/tests/testsuite"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute]VM Affinity", decorators.SigCompute, decorators.SigComputeMigrations, decorators.RequiresTwoSchedulableNodes, decorators.VMLiveUpdateFeaturesGate, Serial, func() {
	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		kv := util2.GetCurrentKv(virtClient)
		kv.Spec.Configuration.VMRolloutStrategy = pointer.P(v1.VMRolloutStrategyLiveUpdate)
		testsuite.UpdateKubeVirtConfigValue(kv.Spec.Configuration)
	})

	Context("Updating VMs node affinity", func() {
		patchValue := func(op, path string, obj interface{}) []byte {
			patches := patch.New()
			switch op {
			case patch.PatchReplaceOp:
				patches.Replace(path, obj)
			case patch.PatchAddOp:
				patches.Add(path, obj)
			case patch.PatchTestOp:
				patches.Test(path, obj)
			case patch.PatchRemoveOp:
				patches.Remove(path)
			default:
				ginkgo.Fail(fmt.Sprintf("Unrecognized patch operation: %s", op))
			}
			patch, err := patches.GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("Patch value: %s", string(patch)))

			return patch
		}
		patchVMNodeSelector := func(newNodeSelectorMap map[string]string, op string, vmName string, vmNamespace string) {
			patch := patchValue(op, "/spec/template/spec/nodeSelector", newNodeSelectorMap)
			_, err := virtClient.VirtualMachine(vmNamespace).Patch(context.Background(), vmName, types.JSONPatchType, patch, &k8smetav1.PatchOptions{})
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
									{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodeName}},
								},
							},
						},
					},
				},
			}
		}

		patchVMAffinity := func(vmAffinity *k8sv1.Affinity, op string, vmName string, vmNamespace string) {
			patch := patchValue(op, "/spec/template/spec/affinity", vmAffinity)
			_, err := virtClient.VirtualMachine(vmNamespace).Patch(context.Background(), vmName, types.JSONPatchType, patch, &k8smetav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		It("should successfully update node selector", func() {

			By("Creating a running VM")
			options := libnet.WithMasqueradeNetworking()
			options = append(options, libvmi.WithCPUCount(1, 2, 1))
			vmi := libvmifact.NewAlpineWithTestTooling(
				options...,
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

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
		It("should successfully update node affinity", func() {

			By("Creating a running VM")
			vmi := libvmifact.NewAlpineWithTestTooling(
				libnet.WithMasqueradeNetworking()...,
			)
			vmi.Namespace = testsuite.GetTestNamespace(vmi)
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunning())

			vm, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisVM(vm), 360*time.Second, 1*time.Second).Should(BeReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

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
})
