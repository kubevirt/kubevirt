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
