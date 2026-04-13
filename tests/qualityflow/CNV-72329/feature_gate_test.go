/*
 * Feature Gate Control Tests
 *
 * STP Reference: outputs/stp/CNV-72329/CNV-72329_test_plan.md
 * Jira: CNV-72329 - Live Update NAD Reference on Running VM
 *
 * Scenarios: TS-CNV72329-003, TS-CNV72329-004
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
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Live Update NAD Reference - Feature Gate"), decorators.SigNetwork, Serial, func() {

	Context("when LiveUpdateNADRef feature gate is disabled", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vm *v1.VirtualMachine

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Disabling LiveUpdateNADRef feature gate")
			kv := libkubevirt.GetCurrentKv(kubevirt.Client())
			if kv.Spec.Configuration.DeveloperConfiguration == nil {
				kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
			}
			// Remove LiveUpdateNADRef from feature gates if present
			var filteredGates []string
			for _, fg := range kv.Spec.Configuration.DeveloperConfiguration.FeatureGates {
				if fg != "LiveUpdateNADRef" {
					filteredGates = append(filteredGates, fg)
				}
			}
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = filteredGates
			_, err := kubevirt.Client().KubeVirt(kv.Namespace).Update(ctx, kv, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			currentKv := libkubevirt.GetCurrentKv(kubevirt.Client())
			config.WaitForConfigToBePropagatedToComponent(
				"kubevirt.io=virt-controller",
				currentKv.ResourceVersion,
				config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
				time.Minute)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad1)
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

		AfterAll(func() {
			By("Re-enabling LiveUpdateNADRef feature gate")
			kv := libkubevirt.GetCurrentKv(kubevirt.Client())
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(
				kv.Spec.Configuration.DeveloperConfiguration.FeatureGates, "LiveUpdateNADRef")
			_, err := kubevirt.Client().KubeVirt(kv.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("[test_id:TS-CNV72329-003] should require restart for NAD change", func() {
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

			By("Verifying RestartRequired condition is set")
			Eventually(matcher.ThisVM(vm)).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineRestartRequired))

			By("Verifying VM is NOT automatically migrated")
			Consistently(func() bool {
				vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				// Check that the NAD is still the old one (change not applied live)
				for _, net := range vmi.Spec.Networks {
					if net.Name == secondaryIfName && net.Multus != nil {
						return net.Multus.NetworkName == nad1Name
					}
				}
				return false
			}).WithTimeout(30 * time.Second).WithPolling(5 * time.Second).Should(BeTrue())
		})
	})

	Context("when feature gate is disabled and VM is restarted", Ordered, decorators.OncePerOrderedCleanup, func() {
		var vm *v1.VirtualMachine

		BeforeAll(func() {
			ctx := context.Background()
			namespace := testsuite.GetTestNamespace(nil)

			By("Disabling LiveUpdateNADRef feature gate")
			kv := libkubevirt.GetCurrentKv(kubevirt.Client())
			if kv.Spec.Configuration.DeveloperConfiguration == nil {
				kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
			}
			var filteredGates []string
			for _, fg := range kv.Spec.Configuration.DeveloperConfiguration.FeatureGates {
				if fg != "LiveUpdateNADRef" {
					filteredGates = append(filteredGates, fg)
				}
			}
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = filteredGates
			_, err := kubevirt.Client().KubeVirt(kv.Namespace).Update(ctx, kv, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			currentKv := libkubevirt.GetCurrentKv(kubevirt.Client())
			config.WaitForConfigToBePropagatedToComponent(
				"kubevirt.io=virt-controller",
				currentKv.ResourceVersion,
				config.ExpectResourceVersionToBeLessEqualThanConfigVersion,
				time.Minute)

			By("Creating two bridge-based NADs")
			nad1 := libnet.NewBridgeNetAttachDef(nad1Name, bridgeName1)
			_, err = libnet.CreateNetAttachDef(ctx, namespace, nad1)
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

		AfterAll(func() {
			By("Re-enabling LiveUpdateNADRef feature gate")
			kv := libkubevirt.GetCurrentKv(kubevirt.Client())
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(
				kv.Spec.Configuration.DeveloperConfiguration.FeatureGates, "LiveUpdateNADRef")
			_, err := kubevirt.Client().KubeVirt(kv.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("[test_id:TS-CNV72329-004] should connect to new network only after manual restart", func() {
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

			By("Verifying RestartRequired condition is set")
			Eventually(matcher.ThisVM(vm)).WithTimeout(nadChangeTimeout).WithPolling(nadChangePoll).
				Should(matcher.HaveConditionTrue(v1.VirtualMachineRestartRequired))

			By("Manually restarting VM")
			vm = libvmops.StopVirtualMachine(vm)
			vm = libvmops.StartVirtualMachine(vm)

			By("Waiting for VM to be ready after restart")
			vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

			By("Verifying VM connects to nad2 after restart")
			for _, net := range vmi.Spec.Networks {
				if net.Name == secondaryIfName && net.Multus != nil {
					Expect(net.Multus.NetworkName).To(Equal(nad2Name))
					return
				}
			}
			Fail("Secondary network with nad2 not found after restart")
		})
	})
})
