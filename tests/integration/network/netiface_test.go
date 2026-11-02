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
