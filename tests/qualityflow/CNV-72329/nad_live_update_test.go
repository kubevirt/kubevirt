/*
 * NAD Live Update Core Functionality Tests
 *
 * STP Reference: outputs/stp/CNV-72329/CNV-72329_test_plan.md
 * Jira: CNV-72329 - Live Update NAD Reference on Running VM
 *
 * Scenarios: TS-CNV72329-001, TS-CNV72329-005, TS-CNV72329-008, TS-CNV72329-012, TS-CNV72329-013
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

const (
	nad1Name         = "nad1"
	nad2Name         = "nad2"
	bridgeName1      = "br-test1"
	bridgeName2      = "br-test2"
	secondaryIfName  = "secondary"
	nadChangeTimeout = 2 * time.Minute
	nadChangePoll    = 5 * time.Second
)

var _ = Describe(SIG("Live Update NAD Reference"), decorators.SigNetwork, Serial, func() {

	Context("when changing NAD reference on a running VM", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			vm  *v1.VirtualMachine
			vmi *v1.VirtualMachineInstance
		)

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			nad2 := libnet.NewBridgeNetAttachDef(nad2Name, bridgeName2)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad2)
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

			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		})

		It("[test_id:TS-CNV72329-001] should connect to the new network", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Patching VM spec to change NAD reference from nad1 to nad2")
			patchData := fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				nad2Name,
			)
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for NAD change to take effect")
			Eventually(func() string {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				for _, net := range updatedVMI.Spec.Networks {
					if net.Name == secondaryIfName && net.Multus != nil {
						return net.Multus.NetworkName
					}
				}
				return ""
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(Equal(nad2Name))

			By("Verifying VM is reachable on new network")
			updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVMI.Status.Phase).To(Equal(v1.Running))
		})
	})

	Context("when NAD reference is changed", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			vm               *v1.VirtualMachine
			vmi              *v1.VirtualMachineInstance
			originalMAC      string
			originalIfceName string
		)

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			nad2 := libnet.NewBridgeNetAttachDef(nad2Name, bridgeName2)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad2)
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

			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

			By("Recording MAC address and interface name")
			for _, iface := range vmi.Status.Interfaces {
				if iface.Name == secondaryIfName {
					originalMAC = iface.MAC
					originalIfceName = iface.InterfaceName
					break
				}
			}
			Expect(originalMAC).NotTo(BeEmpty(), "Secondary interface MAC not found")
		})

		It("[test_id:TS-CNV72329-005] should preserve MAC address and interface name", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Patching VM spec to change NAD reference from nad1 to nad2")
			patchData := fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				nad2Name,
			)
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for NAD change to take effect")
			Eventually(func() string {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				for _, net := range updatedVMI.Spec.Networks {
					if net.Name == secondaryIfName && net.Multus != nil {
						return net.Multus.NetworkName
					}
				}
				return ""
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(Equal(nad2Name))

			By("Verifying MAC address is unchanged")
			updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			var currentMAC, currentIfceName string
			for _, iface := range updatedVMI.Status.Interfaces {
				if iface.Name == secondaryIfName {
					currentMAC = iface.MAC
					currentIfceName = iface.InterfaceName
					break
				}
			}
			Expect(currentMAC).To(Equal(originalMAC), "MAC address changed after NAD update")

			By("Verifying interface name is unchanged")
			Expect(currentIfceName).To(Equal(originalIfceName), "Interface name changed after NAD update")
		})
	})

	Context("when verifying no restart during NAD change", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			vm          *v1.VirtualMachine
			originalUID types.UID
		)

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			nad2 := libnet.NewBridgeNetAttachDef(nad2Name, bridgeName2)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad2)
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

			By("Waiting for VM to be ready and recording VMI UID")
			Eventually(matcher.ThisVM(vm)).WithTimeout(6 * time.Minute).WithPolling(3 * time.Second).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineReady))

			vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			_ = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
			originalUID = vmi.UID
		})

		It("[test_id:TS-CNV72329-008] should not restart the VM", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Patching VM spec to change NAD reference")
			patchData := fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				nad2Name,
			)
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for update to complete")
			Eventually(func() string {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				for _, net := range updatedVMI.Spec.Networks {
					if net.Name == secondaryIfName && net.Multus != nil {
						return net.Multus.NetworkName
					}
				}
				return ""
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(Equal(nad2Name))

			By("Verifying VMI UID is unchanged (no restart)")
			updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVMI.UID).To(Equal(originalUID), "VMI UID changed — VM was restarted")

			By("Verifying no RestartRequired condition")
			Consistently(matcher.ThisVM(vm)).WithTimeout(30 * time.Second).WithPolling(5 * time.Second).
				ShouldNot(matcher.HaveConditionTrue(v1.VirtualMachineRestartRequired))
		})
	})

	Context("when using namespace-qualified NAD names", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vm *v1.VirtualMachine

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			nad2 := libnet.NewBridgeNetAttachDef(nad2Name, bridgeName2)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad2)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Fedora VM with namespace-qualified NAD name")
			qualifiedNAD1 := fmt.Sprintf("%s/%s", namespace, nad1Name)
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(secondaryIfName)),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryIfName, qualifiedNAD1)),
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

		It("[test_id:TS-CNV72329-012] should change NAD reference using qualified names", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Patching VM spec with namespace-qualified NAD name")
			qualifiedNAD2 := fmt.Sprintf("%s/%s", namespace, nad2Name)
			patchData := fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				qualifiedNAD2,
			)
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying NAD change takes effect with qualified name")
			Eventually(func() string {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				for _, net := range updatedVMI.Spec.Networks {
					if net.Name == secondaryIfName && net.Multus != nil {
						return net.Multus.NetworkName
					}
				}
				return ""
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(ContainSubstring(nad2Name))
		})
	})

	Context("after NAD reference is changed", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vm *v1.VirtualMachine

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err := libnet.CreateNetAttachDef(ctx, namespace, nad1)
			Expect(err).NotTo(HaveOccurred())

			nad2 := libnet.NewBridgeNetAttachDef(nad2Name, bridgeName2)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad2)
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

		It("[test_id:TS-CNV72329-013] should reflect the change in VMI spec", func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Patching VM spec to change NAD reference to nad2")
			patchData := fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				nad2Name,
			)
			_, err := kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType,
				[]byte(patchData), metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for update to complete")
			Eventually(func() string {
				updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				for _, net := range updatedVMI.Spec.Networks {
					if net.Name == secondaryIfName && net.Multus != nil {
						return net.Multus.NetworkName
					}
				}
				return ""
			}).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).Should(Equal(nad2Name))

			By("Verifying VMI spec shows nad2")
			updatedVMI, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			found := false
			for _, net := range updatedVMI.Spec.Networks {
				if net.Name == secondaryIfName && net.Multus != nil {
					Expect(net.Multus.NetworkName).To(Equal(nad2Name))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Secondary network not found in VMI spec")
		})
	})
})
