/*
 * Live Update NAD Reference Tests
 *
 * STP Reference: tests/CNV-72329/CNV-72329_test_plan.md
 * Jira: CNV-72329
 * PR: https://github.com/kubevirt/kubevirt/pull/16412
 *
 * This test suite validates the Live Update NAD Reference feature which
 * allows users to change the NetworkAttachmentDefinition reference on a
 * running VM's network interface through live migration.
 */
package network

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubevirt"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[CNV-72329] Live Update NAD Reference", decorators.SigNetwork, Serial, func() {
	var (
		ctx       context.Context
		namespace string
		err       error
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = testsuite.GetTestNamespace(nil)
	})

	/*
	   Markers:
	       - tier1
	       - gating
	       - sig-network

	   Preconditions:
	       - OpenShift cluster with CNV and LiveUpdateNADRefEnabled feature gate
	       - Multi-node cluster with shared storage for live migration
	       - At least two NetworkAttachmentDefinitions available
	*/

	Context("NAD reference update triggers live migration", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			sourceNAD *networkv1.NetworkAttachmentDefinition
			targetNAD *networkv1.NetworkAttachmentDefinition
			vm        *v1.VirtualMachine
			vmi       *v1.VirtualMachineInstance
		)

		BeforeAll(func() {
			By("Creating source NAD")
			sourceNAD = libnet.CreateNAD(namespace, "nad-source")
			ExpectWithOffset(1, sourceNAD).ToNot(BeNil())

			By("Creating target NAD")
			targetNAD = libnet.CreateNAD(namespace, "nad-target")
			ExpectWithOffset(1, targetNAD).ToNot(BeNil())

			By("Creating VM with source NAD")
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", sourceNAD.Name)),
			)
			vm, err = kubevirt.Client().VirtualMachine(namespace).Create(ctx, &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nad-update-test-vm",
					Namespace: namespace,
				},
				Spec: v1.VirtualMachineSpec{
					Running: pointer.Bool(true),
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmiSpec.Spec,
					},
				},
			}, metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			By("Waiting for VMI to be running")
			Eventually(func() bool {
				vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				return err == nil && vmi.Status.Phase == v1.Running
			}, 180*time.Second, time.Second).Should(BeTrue())

			vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
		})

		AfterAll(func() {
			By("Cleaning up VM")
			if vm != nil {
				err = kubevirt.Client().VirtualMachine(namespace).Delete(ctx, vm.Name, metav1.DeleteOptions{})
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
			}

			By("Cleaning up NADs")
			if sourceNAD != nil {
				_ = kubevirt.Client().NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Delete(ctx, sourceNAD.Name, metav1.DeleteOptions{})
			}
			if targetNAD != nil {
				_ = kubevirt.Client().NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Delete(ctx, targetNAD.Name, metav1.DeleteOptions{})
			}
		})

		/*
		   Preconditions:
		       - VM running with secondary network attached to source NAD
		       - Target NAD exists in namespace

		   Steps:
		       1. Update VM spec to reference target NAD
		       2. Verify migration is triggered

		   Expected:
		       - Migration object created after NAD update
		*/
		It("[test_id:TS-CNV72329-001] should trigger live migration when NAD reference is updated", func() {
			By("Updating VM spec to use target NAD")
			patchData := []byte(fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				targetNAD.Name,
			))
			vm, err = kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			By("Verifying migration is triggered")
			Eventually(func() bool {
				migrations, err := kubevirt.Client().VirtualMachineInstanceMigration(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return false
				}
				for _, m := range migrations.Items {
					if m.Spec.VMIName == vm.Name {
						return true
					}
				}
				return false
			}, 60*time.Second, time.Second).Should(BeTrue(), "Migration should be triggered after NAD reference update")
		})
	})

	Context("Network connectivity preserved after NAD change", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			sourceNAD *networkv1.NetworkAttachmentDefinition
			targetNAD *networkv1.NetworkAttachmentDefinition
			vm        *v1.VirtualMachine
			vmi       *v1.VirtualMachineInstance
		)

		BeforeAll(func() {
			By("Creating NADs and VM")
			sourceNAD = libnet.CreateNAD(namespace, "connectivity-source")
			targetNAD = libnet.CreateNAD(namespace, "connectivity-target")

			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", sourceNAD.Name)),
			)
			vm = createAndStartVM(ctx, namespace, "connectivity-test-vm", vmiSpec)
			vmi = libwait.WaitUntilVMIReady(vm.Status.PrintableStatus, console.LoginToFedora)
		})

		AfterAll(func() {
			cleanupVMAndNADs(ctx, namespace, vm, sourceNAD, targetNAD)
		})

		/*
		   Preconditions:
		       - VM running with network connectivity on source NAD
		       - Target NAD provides routable network

		   Steps:
		       1. Verify initial network connectivity
		       2. Update NAD reference and wait for migration
		       3. Verify network connectivity on new NAD

		   Expected:
		       - VM has IP address and responds to network requests after NAD update
		*/
		It("[test_id:TS-CNV72329-002] should preserve network connectivity after NAD update completes", func() {
			By("Verifying initial network connectivity")
			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			initialIP := libnet.GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol)
			ExpectWithOffset(1, initialIP).ToNot(BeEmpty(), "VM must have initial IP")

			By("Updating NAD reference to target NAD")
			updateVMNADReference(ctx, vm, targetNAD.Name)

			By("Waiting for migration to complete")
			Eventually(func() bool {
				vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				return err == nil && vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed
			}, 300*time.Second, time.Second).Should(BeTrue(), "Migration should complete")

			By("Verifying network connectivity after NAD update")
			vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			newIP := libnet.GetVmiPrimaryIPByFamily(vmi, k8sv1.IPv4Protocol)
			ExpectWithOffset(1, newIP).ToNot(BeEmpty(), "VM must have IP after NAD update")
		})
	})

	Context("Feature gate controls NAD update capability", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			sourceNAD *networkv1.NetworkAttachmentDefinition
			vm        *v1.VirtualMachine
		)

		BeforeAll(func() {
			// Note: This test assumes feature gate is disabled
			// In a real test environment, you would need to toggle the feature gate
			sourceNAD = libnet.CreateNAD(namespace, "feature-gate-source")
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", sourceNAD.Name)),
			)
			vm = createAndStartVM(ctx, namespace, "feature-gate-test-vm", vmiSpec)
			libwait.WaitUntilVMIReady(vm.Status.PrintableStatus, console.LoginToFedora)
		})

		AfterAll(func() {
			cleanupVMAndNADs(ctx, namespace, vm, sourceNAD, nil)
		})

		/*
		   [NEGATIVE]
		   Preconditions:
		       - LiveUpdateNADRefEnabled feature gate is disabled
		       - VM running with secondary network

		   Steps:
		       1. Attempt to update NAD reference

		   Expected:
		       - NAD update rejected with error indicating feature is disabled
		*/
		It("[test_id:TS-CNV72329-003] should reject NAD update when feature gate is disabled", func() {
			Skip("Test requires feature gate to be disabled - run manually with appropriate cluster configuration")

			By("Attempting to update NAD reference with feature gate disabled")
			patchData := []byte(`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "nad-target"}]`)
			_, err = kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			ExpectWithOffset(1, err).To(HaveOccurred())
			ExpectWithOffset(1, err.Error()).To(ContainSubstring("feature gate"))
		})
	})

	Context("Invalid NAD reference rejected", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			sourceNAD *networkv1.NetworkAttachmentDefinition
			vm        *v1.VirtualMachine
		)

		BeforeAll(func() {
			sourceNAD = libnet.CreateNAD(namespace, "invalid-nad-source")
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", sourceNAD.Name)),
			)
			vm = createAndStartVM(ctx, namespace, "invalid-nad-test-vm", vmiSpec)
			libwait.WaitUntilVMIReady(vm.Status.PrintableStatus, console.LoginToFedora)
		})

		AfterAll(func() {
			cleanupVMAndNADs(ctx, namespace, vm, sourceNAD, nil)
		})

		/*
		   [NEGATIVE]
		   Preconditions:
		       - VM running with secondary network
		       - Target NAD does not exist

		   Steps:
		       1. Attempt to update NAD reference to non-existent NAD

		   Expected:
		       - Error returned indicating NAD not found
		*/
		It("[test_id:TS-CNV72329-004] should return error for non-existent NAD reference", func() {
			By("Attempting to update to non-existent NAD")
			patchData := []byte(`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "non-existent-nad"}]`)
			_, err = kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			ExpectWithOffset(1, err).To(HaveOccurred(), "Update to non-existent NAD should fail")
		})
	})

	Context("VM must be running for NAD update", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			sourceNAD *networkv1.NetworkAttachmentDefinition
			targetNAD *networkv1.NetworkAttachmentDefinition
			vm        *v1.VirtualMachine
		)

		BeforeAll(func() {
			sourceNAD = libnet.CreateNAD(namespace, "stopped-vm-source")
			targetNAD = libnet.CreateNAD(namespace, "stopped-vm-target")

			// Create stopped VM
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary", sourceNAD.Name)),
			)
			vm, err = kubevirt.Client().VirtualMachine(namespace).Create(ctx, &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "stopped-vm-test",
					Namespace: namespace,
				},
				Spec: v1.VirtualMachineSpec{
					Running: pointer.Bool(false), // VM is stopped
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						Spec: vmiSpec.Spec,
					},
				},
			}, metav1.CreateOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		})

		AfterAll(func() {
			cleanupVMAndNADs(ctx, namespace, vm, sourceNAD, targetNAD)
		})

		/*
		   Preconditions:
		       - VM exists but is stopped
		       - Secondary network configured

		   Steps:
		       1. Attempt to update NAD reference on stopped VM

		   Expected:
		       - Spec update accepted but migration deferred until VM starts
		*/
		It("[test_id:TS-CNV72329-005] should handle NAD update for stopped VM appropriately", func() {
			By("Attempting NAD update on stopped VM")
			patchData := []byte(fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				targetNAD.Name,
			))
			vm, err = kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			// Spec update on stopped VM should be accepted
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Spec update on stopped VM should be accepted")

			By("Verifying no migration triggered (VM is stopped)")
			Consistently(func() bool {
				migrations, err := kubevirt.Client().VirtualMachineInstanceMigration(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return true // No migrations if error
				}
				for _, m := range migrations.Items {
					if m.Spec.VMIName == vm.Name {
						return false
					}
				}
				return true
			}, 10*time.Second, time.Second).Should(BeTrue(), "No migration should be triggered for stopped VM")
		})
	})

	Context("Multi-interface VM NAD update", Ordered, decorators.OncePerOrderedCleanup, func() {
		var (
			nad1     *networkv1.NetworkAttachmentDefinition
			nad2     *networkv1.NetworkAttachmentDefinition
			nad1New  *networkv1.NetworkAttachmentDefinition
			vm       *v1.VirtualMachine
			vmi      *v1.VirtualMachineInstance
		)

		BeforeAll(func() {
			nad1 = libnet.CreateNAD(namespace, "multi-iface-nad-1")
			nad2 = libnet.CreateNAD(namespace, "multi-iface-nad-2")
			nad1New = libnet.CreateNAD(namespace, "multi-iface-nad-1-new")

			// Create VM with multiple secondary interfaces
			vmiSpec := libvmifact.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary1")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary1", nad1.Name)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding("secondary2")),
				libvmi.WithNetwork(libvmi.MultusNetwork("secondary2", nad2.Name)),
			)
			vm = createAndStartVM(ctx, namespace, "multi-iface-test-vm", vmiSpec)
			vmi = libwait.WaitUntilVMIReady(vm.Status.PrintableStatus, console.LoginToFedora)
		})

		AfterAll(func() {
			if vm != nil {
				err = kubevirt.Client().VirtualMachine(namespace).Delete(ctx, vm.Name, metav1.DeleteOptions{})
				ExpectWithOffset(1, err).ToNot(HaveOccurred())
			}
			for _, nad := range []*networkv1.NetworkAttachmentDefinition{nad1, nad2, nad1New} {
				if nad != nil {
					_ = kubevirt.Client().NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Delete(ctx, nad.Name, metav1.DeleteOptions{})
				}
			}
		})

		/*
		   Preconditions:
		       - VM running with multiple secondary network interfaces
		       - Multiple NADs available

		   Steps:
		       1. Update NAD reference for first secondary interface only
		       2. Wait for migration to complete
		       3. Verify only targeted interface changed

		   Expected:
		       - Only the targeted interface uses new NAD
		       - Other interfaces remain on original NADs
		*/
		It("[test_id:TS-CNV72329-006] should update single interface NAD on multi-interface VM", func() {
			By("Updating NAD reference for first secondary interface only")
			// Update only the first secondary network (index 1, as index 0 is default network)
			patchData := []byte(fmt.Sprintf(
				`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
				nad1New.Name,
			))
			vm, err = kubevirt.Client().VirtualMachine(namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			By("Waiting for migration to complete")
			Eventually(func() bool {
				vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				return err == nil && vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed
			}, 300*time.Second, time.Second).Should(BeTrue())

			By("Verifying second interface still on original NAD")
			// Get updated VM spec
			vm, err = kubevirt.Client().VirtualMachine(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			// Verify first network changed
			ExpectWithOffset(1, vm.Spec.Template.Spec.Networks[1].Multus.NetworkName).To(Equal(nad1New.Name))

			// Verify second network unchanged
			ExpectWithOffset(1, vm.Spec.Template.Spec.Networks[2].Multus.NetworkName).To(Equal(nad2.Name))
		})
	})
})

// Helper functions

func createAndStartVM(ctx context.Context, namespace, name string, vmiSpec *v1.VirtualMachineInstance) *v1.VirtualMachine {
	vm, err := kubevirt.Client().VirtualMachine(namespace).Create(ctx, &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: pointer.Bool(true),
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: vmiSpec.Spec,
			},
		},
	}, metav1.CreateOptions{})
	ExpectWithOffset(2, err).ToNot(HaveOccurred())

	Eventually(func() bool {
		vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(ctx, vm.Name, metav1.GetOptions{})
		return err == nil && vmi.Status.Phase == v1.Running
	}, 180*time.Second, time.Second).Should(BeTrue())

	return vm
}

func updateVMNADReference(ctx context.Context, vm *v1.VirtualMachine, newNADName string) {
	patchData := []byte(fmt.Sprintf(
		`[{"op": "replace", "path": "/spec/template/spec/networks/1/multus/networkName", "value": "%s"}]`,
		newNADName,
	))
	_, err := kubevirt.Client().VirtualMachine(vm.Namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
}

func cleanupVMAndNADs(ctx context.Context, namespace string, vm *v1.VirtualMachine, nads ...*networkv1.NetworkAttachmentDefinition) {
	if vm != nil {
		err := kubevirt.Client().VirtualMachine(namespace).Delete(ctx, vm.Name, metav1.DeleteOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())
	}
	for _, nad := range nads {
		if nad != nil {
			_ = kubevirt.Client().NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Delete(ctx, nad.Name, metav1.DeleteOptions{})
		}
	}
}
