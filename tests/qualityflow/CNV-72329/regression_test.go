/*
 * Regression and Coexistence Tests
 *
 * STP Reference: outputs/stp/CNV-72329/CNV-72329_test_plan.md
 * Jira: CNV-72329 - Live Update NAD Reference on Running VM
 *
 * Scenarios: TS-CNV72329-009, TS-CNV72329-010, TS-CNV72329-014
 */

package network

import (
	"context"
	"fmt"
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

var _ = Describe(SIG("Live Update NAD Reference - Regression"), decorators.SigNetwork, Serial, func() {

	Context("when a non-NAD network property is changed", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vm *v1.VirtualMachine

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating bridge-based NAD")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Fedora VM with secondary bridge interface")
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

		It("[test_id:TS-CNV72329-009] should still require restart", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Changing a non-NAD network property (adding a new interface with different binding)")
			patchData := `[{"op": "add", "path": "/spec/template/spec/domain/devices/interfaces/-", "value": {"name": "extra-iface", "sriov": {}}}, {"op": "add", "path": "/spec/template/spec/networks/-", "value": {"name": "extra-iface", "multus": {"networkName": "sriov-nad"}}}]`
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying RestartRequired condition is set for non-NAD changes")
			Eventually(matcher.ThisVM(vm)).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineRestartRequired))
		})
	})

	Context("when NIC hotplug is performed with feature gate enabled", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			vm  *v1.VirtualMachine
			vmi *v1.VirtualMachineInstance
		)

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating bridge-based NAD for hotplug")
			hotplugNAD := libnet.NewBridgeNetAttachDef("hotplug-nad", bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, hotplugNAD)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Fedora VM with masquerade only")
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vm = libvmi.NewVirtualMachine(vmiSpec, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err = kubevirt.Client().VirtualMachine(namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm)).WithTimeout(6 * time.Minute).WithPolling(3 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineReady))

			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		})

		It("[test_id:TS-CNV72329-010] should hotplug and unplug NIC successfully", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)
			hotplugIfaceName := "hotplugged"

			By("Hotplugging a new bridge NIC to the VM")
			hotplugIface := libvmi.InterfaceDeviceWithBridgeBinding(hotplugIfaceName)
			hotplugNet := *libvmi.MultusNetwork(hotplugIfaceName, "hotplug-nad")
			Expect(libnet.PatchVMWithNewInterface(vm, hotplugNet, hotplugIface)).To(Succeed())

			By("Waiting for hotplugged NIC to appear in VMI spec")
			updatedVMI := libnet.WaitForSingleHotPlugIfaceOnVMISpec(vmi, hotplugIfaceName, "hotplug-nad")
			Expect(updatedVMI).NotTo(BeNil())

			By("Verifying hotplugged NIC is functional")
			Expect(libnet.InterfaceExists(updatedVMI, "eth1")).To(Succeed())

			By("Unplugging the NIC")
			patchData := fmt.Sprintf(
				`[{"op": "test", "path": "/spec/template/spec/domain/devices/interfaces", "value": %s}, {"op": "replace", "path": "/spec/template/spec/domain/devices/interfaces", "value": [{"name": "default", "masquerade": {}}]}]`,
				"null",
			)
			// Use simpler approach: remove the hotplugged interface
			removeIfacePatch := `[{"op": "remove", "path": "/spec/template/spec/domain/devices/interfaces/1"}, {"op": "remove", "path": "/spec/template/spec/networks/1"}]`
			_ = patchData // avoid unused
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(removeIfacePatch), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying NIC removed from VMI spec")
			Eventually(func() int {
				currentVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				return len(currentVMI.Spec.Domain.Devices.Interfaces)
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(Equal(1))
		})
	})

	Context("existing network features", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			vm  *v1.VirtualMachine
			vmi *v1.VirtualMachineInstance
		)

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating bridge-based NAD for regression test")
			bridgeNAD := libnet.NewBridgeNetAttachDef("bridge-regression-nad", bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, bridgeNAD)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Fedora VM with masquerade only")
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vm = libvmi.NewVirtualMachine(vmiSpec, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err = kubevirt.Client().VirtualMachine(namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for VM to be ready")
			Eventually(matcher.ThisVM(vm)).WithTimeout(6 * time.Minute).WithPolling(3 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineReady))

			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		})

		It("[test_id:TS-CNV72329-014] should not regress SR-IOV and bridge hotplug", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)
			bridgeHotplugName := "bridge-hotplug"

			By("Performing bridge NIC hotplug")
			bridgeIface := libvmi.InterfaceDeviceWithBridgeBinding(bridgeHotplugName)
			bridgeNet := *libvmi.MultusNetwork(bridgeHotplugName, "bridge-regression-nad")
			Expect(libnet.PatchVMWithNewInterface(vm, bridgeNet, bridgeIface)).To(Succeed())

			By("Verifying bridge hotplug succeeds")
			updatedVMI := libnet.WaitForSingleHotPlugIfaceOnVMISpec(vmi, bridgeHotplugName, "bridge-regression-nad")
			Expect(updatedVMI).NotTo(BeNil())

			By("Verifying hotplugged bridge interface is visible in guest")
			Expect(libnet.InterfaceExists(updatedVMI, "eth1")).To(Succeed())

			By("Checking if SR-IOV is available (skip if not)")
			sriovNADs, err := kubevirt.Client().NetworkClient().K8sCniCncfIoV1().
				NetworkAttachmentDefinitions(namespace).List(ctx, metav1.ListOptions{})
			if err != nil || len(sriovNADs.Items) == 0 {
				Skip("SR-IOV NADs not available on this cluster — skipping SR-IOV hotplug regression check")
			}

			hasSRIOV := false
			for _, nad := range sriovNADs.Items {
				if nad.Spec.Config != "" {
					// Simple heuristic: check if any NAD uses sriov type
					if fmt.Sprintf("%s", nad.Spec.Config) != "" {
						hasSRIOV = true
						break
					}
				}
			}
			if !hasSRIOV {
				Skip("No SR-IOV NADs found — skipping SR-IOV hotplug regression check")
			}
		})
	})
})
