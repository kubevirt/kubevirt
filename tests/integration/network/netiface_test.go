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

// These specs exercise the +kubebuilder:validation:MaxLength and
// +kubebuilder:validation:XValidation (CEL) constraints on Interface.MacAddress
// (staging/src/kubevirt.io/api/core/v1/schema.go) restricting the MAC address
// to a colon-, hyphen-, or dot-separated 6-byte hex address. No KubeVirt
// webhook runs in this environment, so a rejection here can only come from
// the CRD-embedded constraints.
var _ = Describe("Interface MAC address CRD validation", func() {
	It("should accept an interface with no MAC address set", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding())),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should accept a well-formed MAC address", func(ctx SpecContext, macAddress string) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding(), libvmi.WithMac(macAddress))),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("lowercase, colon separated", "de:ad:00:00:be:af"),
		Entry("uppercase, colon separated", "DE:AD:00:00:BE:AF"),
		Entry("lowercase, dash separated", "de-ad-00-00-be-af"),
		Entry("uppercase, dash separated", "DE-AD-00-00-BE-AF"),
		Entry("lowercase, dot separated", "dead.0000.beaf"),
		Entry("uppercase, dot separated", "DEAD.0000.BEAF"),
	)

	// A MacAddress longer than 64 characters is rejected by the MaxLength
	// constraint before the XValidation (CEL) rule is ever evaluated, reported
	// as a distinct top-level failure from the CEL rule's own message. The
	// bound is set well above any format this field currently accepts (see
	// the "fits within the max length" table below for malformed-but-short
	// values), to leave room for a future longer hardware-address format
	// (e.g. InfiniBand's 59-character colon-separated form, RFC 4391)
	// without needing to widen this published schema keyword again later.
	DescribeTable("should reject a MAC address exceeding the max length", func(ctx SpecContext, macAddress string) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding(), libvmi.WithMac(macAddress))),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].macAddress"))
		Expect(err.Error()).To(ContainSubstring("Too long: may not be more than 64 bytes"))
	},
		Entry("too many octets (71 chars)", "de:ad:00:00:be:af:de:ad:00:00:be:af:de:ad:00:00:be:af:de:ad:00:00:be:af"),
		Entry("too many groups, dot form (74 chars)", "dead.0000.beaf.dead.0000.beaf.dead.0000.beaf.dead.0000.beaf.dead.0000.beaf"),
		Entry("not a MAC address at all (67 chars)", "not-a-mac-address-value-that-is-definitely-too-long-to-fit-anywhere"),
	)

	// These values fit within the 64-character MaxLength, so rejection comes
	// from the XValidation (CEL) regex rule instead.
	DescribeTable("should reject a malformed MAC address that fits within the max length", func(ctx SpecContext, macAddress string) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding(), libvmi.WithMac(macAddress))),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].macAddress"))
		Expect(err.Error()).To(ContainSubstring("interface has a malformed MAC address"))
	},
		Entry("too short, colon separated (14 chars)", "de:ad:00:00:be"),
		Entry("too short, dash separated (14 chars)", "de-ad-00-00-be"),
		Entry("single-digit octet (16 chars)", "de:ad:0:0:be:af"),
		Entry("single-digit group, dot form (12 chars)", "dead.00.beaf"),
		Entry("not a MAC address at all (9 chars)", "not-a-mac"),
	)
})
