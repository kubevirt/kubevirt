package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +kubebuilder:validation:Enum constraint on
// Interface.Model (staging/src/kubevirt.io/api/core/v1/schema.go) restricting
// the interface model to a fixed allow-list. No KubeVirt webhook runs in this
// environment, so a rejection here can only come from the CRD-embedded enum
// constraint.
var _ = Describe("Interface model CRD validation", func() {
	DescribeTable("should accept a supported interface model", func(ctx SpecContext, model string) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding(), libvmi.WithModel(model))),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("e1000", "e1000"),
		Entry("e1000e", "e1000e"),
		Entry("igb", "igb"),
		Entry("ne2k_pci", "ne2k_pci"),
		Entry("pcnet", "pcnet"),
		Entry("rtl8139", "rtl8139"),
		Entry("virtio", "virtio"),
		// An empty string is also accepted by the enum (preserving the old
		// webhook behavior of tolerating an unset model), but Model has an
		// `omitempty` JSON tag, so WithModel("") and simply leaving the model
		// unset are indistinguishable on the wire: both omit the field. This
		// entry therefore also covers "no model set".
		Entry("empty (no model set)", ""),
	)

	It("should reject an unsupported interface model", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding(), libvmi.WithModel("not-a-model"))),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].model"))
	})
})

var _ = Describe("Interface PCI address CRD validation", func() {
	It("should accept an interface with no PCI address set", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding())),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should accept a well-formed PCI address", func(ctx SpecContext, pciAddress string) {
		ns := f.CreateNamespace(ctx)

		iface := libvmi.NewInterface("default", libvmi.WithBridgeBinding())
		iface.PciAddress = pciAddress
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(iface),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("0000:81:01.0", "0000:81:01.0"),
		Entry("0000:81:11.1", "0000:81:11.1"),
		Entry("0001:02:00.0", "0001:02:00.0"),
	)

	// A PciAddress longer than 12 characters is rejected by the MaxLength
	// constraint before the XValidation (CEL) rule is ever evaluated. The
	// apiserver reports this as a distinct top-level failure ("Too long: may
	// not be more than 12 bytes") rather than the CEL rule's "malformed PCI
	// address" message, so these cases are kept in their own table with their
	// own expected error text.
	DescribeTable("should reject a PCI address exceeding the max length", func(ctx SpecContext, pciAddress string) {
		ns := f.CreateNamespace(ctx)

		iface := libvmi.NewInterface("default", libvmi.WithBridgeBinding())
		iface.PciAddress = pciAddress
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(iface),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].pciAddress"))
		Expect(err.Error()).To(ContainSubstring("Too long: may not be more than 12 bytes"))
	},
		Entry("too many parts (14 chars)", "0000:80:80:1.0"),
		Entry("not a PCI address at all (18 chars)", "not-a-pci-address"),
	)

	// These values fit within the 12-character MaxLength, so rejection comes
	// from the XValidation (CEL) regex rule instead.
	DescribeTable("should reject a malformed PCI address that fits within the max length", func(ctx SpecContext, pciAddress string) {
		ns := f.CreateNamespace(ctx)

		iface := libvmi.NewInterface("default", libvmi.WithBridgeBinding())
		iface.PciAddress = pciAddress
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(iface),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].pciAddress"))
		Expect(err.Error()).To(ContainSubstring("interface has a malformed PCI address"))
	},
		Entry("too many dots (12 chars)", "0000:80.10.1"),
		Entry("function out of range (12 chars)", "0000:80:11.9"),
	)
})
