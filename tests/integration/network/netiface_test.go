package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +listType=map/+listMapKey=name structural-schema
// constraint on VirtualMachineInstanceSpec.Networks and
// Domain.Devices.Interfaces (staging/src/kubevirt.io/api/core/v1/types.go,
// schema.go) enforcing unique names. No KubeVirt webhook runs in this
// environment, so a rejection here can only come from the CRD's native
// list-of-maps uniqueness enforcement.
var _ = Describe("Network/interface name uniqueness CRD validation", func() {
	It("should accept networks and interfaces that are paired and uniquely named", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("net1", libvmi.WithBridgeBinding())),
			libvmi.WithInterface(libvmi.NewInterface("net2", libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(&v1.Network{Name: "net1", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}),
			libvmi.WithNetwork(libvmi.MultusNetwork("net2", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject networks with duplicate names", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(&v1.Network{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}),
			libvmi.WithNetwork(&v1.Network{Name: "default", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "default-nad"}}}),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`spec.networks[1]: Duplicate value: {"name":"default"}`))
	})

	It("should reject interfaces with duplicate names", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding())),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding())), //nolint:gocritic // intentional duplicate
			libvmi.WithNetwork(&v1.Network{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`spec.domain.devices.interfaces[1]: Duplicate value: {"name":"default"}`))
	})
})

// These specs exercise the +listType=map/+listMapKey=option structural-schema
// constraint on Interface.DHCPOptions.PrivateOptions, plus the CEL
// XValidation range rule on DHCPPrivateOptions.Option
// (staging/src/kubevirt.io/api/core/v1/schema.go). No KubeVirt webhook runs
// in this environment, so a rejection here can only come from the CRD's
// native constraints or CEL rules.
var _ = Describe("DHCP private options CRD validation", func() {
	It("should accept unique, in-range private options", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithDHCPOptions(v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{
					{Option: 224, Value: "a"},
					{Option: 254, Value: "b"},
				},
			}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject private options with duplicate option codes", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithDHCPOptions(v1.DHCPOptions{
				PrivateOptions: []v1.DHCPPrivateOptions{
					{Option: 240, Value: "a"},
					{Option: 240, Value: "b"},
				},
			}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			`spec.domain.devices.interfaces[0].dhcpOptions.privateOptions[1]: Duplicate value: {"option":240}`,
		))
	})

	DescribeTable("should reject an out-of-range private option code",
		func(ctx SpecContext, option int) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithDHCPOptions(v1.DHCPOptions{
					PrivateOptions: []v1.DHCPPrivateOptions{{Option: option, Value: "a"}},
				}))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].dhcpOptions.privateOptions[0].option"))
			Expect(err.Error()).To(ContainSubstring("provided DHCPPrivateOptions are out of range, must be in range 224 to 254"))
		},
		Entry("below the valid range", 223),
		Entry("above the valid range", 255),
	)
})
