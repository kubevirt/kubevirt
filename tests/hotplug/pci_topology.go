/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package hotplug

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]PCI Topology", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("v3 annotation", func() {
		It("should be set on a new VMI", func() {
			vmi := libvmifact.NewCirros()
			vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Annotations).To(HaveKeyWithValue(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV3))
		})

		It("should be set on a new VM template", func() {
			vmi := libvmifact.NewCirros()
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyHalted))
			vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Template.ObjectMeta.Annotations).To(HaveKeyWithValue(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV3))
		})
	})

	Context("v3 PCI address stability", func() {
		It("should preserve PCI addresses across VM restart", func() {
			vmi := libvmifact.NewCirros()
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))
			vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to be ready")
			Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 360, 1).Should(matcher.Exist())
			runningVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			runningVMI = libwait.WaitUntilVMIReady(runningVMI, console.LoginToCirros)

			By("Recording PCI fingerprint before restart")
			fingerprintBefore := guestPCIFingerprint(runningVMI)

			By("Restarting the VM")
			vm = libvmops.StopVirtualMachine(vm)
			vm = libvmops.StartVirtualMachine(vm)

			By("Waiting for VMI to be ready after restart")
			runningVMI, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			runningVMI = libwait.WaitUntilVMIReady(runningVMI, console.LoginToCirros)

			By("Verifying PCI addresses are unchanged")
			fingerprintAfter := guestPCIFingerprint(runningVMI)
			Expect(fingerprintAfter).To(Equal(fingerprintBefore))
		})
	})

	Context("v2 frozen interface slot count", func() {
		It("should use the frozen slot count and preserve addresses across restart", func() {
			By("Creating a stopped VM with v2 annotations")
			vmi := libvmifact.NewCirros()
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyHalted))
			vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Patching VM template with v2 annotations and frozen slot count of 9 (8 placeholders + 1 interface)")
			patchPayload, err := patch.New(
				patch.WithReplace("/spec/template/metadata/annotations/"+patch.EscapeJSONPointer(v1.PciTopologyVersionAnnotation), v1.PciTopologyVersionV2),
				patch.WithAdd("/spec/template/metadata/annotations/"+patch.EscapeJSONPointer(v1.PciInterfaceSlotCountAnnotation), "9"),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Starting the VM")
			vm = libvmops.StartVirtualMachine(vm)

			By("Waiting for VMI to be ready")
			runningVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			runningVMI = libwait.WaitUntilVMIReady(runningVMI, console.LoginToCirros)

			By("Verifying VMI has v2 annotations")
			Expect(runningVMI.Annotations).To(HaveKeyWithValue(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV2))
			Expect(runningVMI.Annotations).To(HaveKeyWithValue(v1.PciInterfaceSlotCountAnnotation, "9"))

			By("Recording PCI fingerprint before restart")
			fingerprintBefore := guestPCIFingerprint(runningVMI)

			By("Restarting the VM")
			vm = libvmops.StopVirtualMachine(vm)
			vm = libvmops.StartVirtualMachine(vm)

			By("Waiting for VMI to be ready after restart")
			runningVMI, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			runningVMI = libwait.WaitUntilVMIReady(runningVMI, console.LoginToCirros)

			By("Verifying v2 annotations are preserved after restart")
			Expect(runningVMI.Annotations).To(HaveKeyWithValue(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV2))
			Expect(runningVMI.Annotations).To(HaveKeyWithValue(v1.PciInterfaceSlotCountAnnotation, "9"))

			By("Verifying PCI addresses are unchanged after restart")
			fingerprintAfter := guestPCIFingerprint(runningVMI)
			Expect(fingerprintAfter).To(Equal(fingerprintBefore))
		})

		It("should produce different PCI addresses than v3", func() {
			By("Creating a v3 VMI")
			v3VMI := libvmifact.NewCirros()
			v3VMI, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), v3VMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			v3VMI = libwait.WaitUntilVMIReady(v3VMI, console.LoginToCirros)
			v3Fingerprint := guestPCIFingerprint(v3VMI)

			By("Creating a v2 VMI with frozen slot count of 9 (8 placeholders + 1 interface)")
			v2VMI := libvmifact.NewCirros(
				libvmi.WithAnnotation(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV2),
				libvmi.WithAnnotation(v1.PciInterfaceSlotCountAnnotation, "9"),
			)
			v2VMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), v2VMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			v2VMI = libwait.WaitUntilVMIReady(v2VMI, console.LoginToCirros)
			v2Fingerprint := guestPCIFingerprint(v2VMI)

			By("Verifying v2 PCI addresses differ from v3")
			Expect(v2Fingerprint).NotTo(Equal(v3Fingerprint),
				"v2 with frozen slot count of 9 should produce different PCI addresses than v3")
		})
	})

	Context("annotation propagation", func() {
		It("should propagate PCI topology annotations from VMI to VM template", func() {
			By("Creating a stopped VM and removing the v3 annotation from the template")
			vmi := libvmifact.NewCirros()
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyHalted))
			vm, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Remove the v3 annotation that the webhook set, simulating a pre-upgrade VM
			patchPayload, err := patch.New(
				patch.WithRemove("/spec/template/metadata/annotations/" + patch.EscapeJSONPointer(v1.PciTopologyVersionAnnotation)),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Spec.Template.ObjectMeta.Annotations).NotTo(HaveKey(v1.PciTopologyVersionAnnotation))

			By("Starting the VM")
			vm = libvmops.StartVirtualMachine(vm)

			By("Verifying VMI gets v3 annotation from webhook")
			runningVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMI.Annotations).To(HaveKeyWithValue(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV3))

			By("Verifying virt-controller propagates annotation back to VM template")
			Eventually(func(g Gomega) {
				vm, err = virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(vm.Spec.Template.ObjectMeta.Annotations).To(
					HaveKeyWithValue(v1.PciTopologyVersionAnnotation, v1.PciTopologyVersionV3))
			}, 60*time.Second, 2*time.Second).Should(Succeed())
		})
	})
})

// guestPCIFingerprint returns an md5sum of the sorted PCI BDF addresses visible to the guest.
func guestPCIFingerprint(vmi *v1.VirtualMachineInstance) string {
	output, err := console.RunCommandAndStoreOutput(vmi, "lspci | awk '{print $1}' | sort | md5sum | awk '{print $1}'", 30*time.Second)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return output
}
