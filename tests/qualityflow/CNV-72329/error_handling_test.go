/*
 * Error Handling Tests
 *
 * STP Reference: outputs/stp/CNV-72329/CNV-72329_test_plan.md
 * Jira: CNV-72329 - Live Update NAD Reference on Running VM
 *
 * Scenarios: TS-CNV72329-006
 */

package network

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Live Update NAD Reference - Error Handling"), decorators.SigNetwork, Serial, func() {

	Context("when target NAD does not exist", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vm *v1.VirtualMachine

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating bridge-based NAD (nad1 only)")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Fedora VM with secondary bridge interface on nad1")
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryIfName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryIfName, nad1Name)),
			)
			vm = libvmi.NewVirtualMachine(vmiSpec, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err = kubevirt.Client().VirtualMachine(namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm)).WithTimeout(6 * time.Minute).WithPolling(3 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineReady))

			vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			_ = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		})

		It("[test_id:TS-CNV72329-006] should report error condition", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Patching VM spec to change NAD reference to non-existent NAD")
			patchData := `[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "non-existent-nad"}]`
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying error condition is reported on VM status")
			Eventually(func() bool {
				updatedVM, err := kubevirt.Client().VirtualMachine(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				for _, cond := range updatedVM.Status.Conditions {
					if cond.Type == v1.VirtualMachineRestartRequired ||
						(cond.Message != "" && (cond.Reason == "NetworkNotFound" || cond.Reason == "NADNotFound")) {
						return true
					}
				}
				return false
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(BeTrue(),
				"Expected error condition for non-existent NAD")

			By("Verifying VM remains running")
			vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(vmi.Status.Phase).To(Equal(v1.Running), "VM should remain running despite invalid NAD reference")
		})
	})
})
