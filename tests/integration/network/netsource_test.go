package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +kubebuilder:validation:XValidation (CEL) rule on
// Network (staging/src/kubevirt.io/api/core/v1/schema.go) requiring exactly
// one of pod/multus/resourceClaim to be set. No KubeVirt webhook runs in this
// environment, so a rejection here can only come from the CRD-embedded CEL rule.
var _ = Describe("Network source CEL validation", func() {
	DescribeTable("should accept a network with exactly one source", func(ctx SpecContext, network *v1.Network, iface v1.Interface) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(iface),
			libvmi.WithNetwork(network),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	},
		Entry("pod",
			&v1.Network{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			libvmi.NewInterface("default", libvmi.WithBridgeBinding()),
		),
		Entry("multus",
			libvmi.MultusNetwork("default", "default-nad"),
			libvmi.NewInterface("default", libvmi.WithBridgeBinding()),
		),
		Entry("resourceClaim",
			libvmi.DRANetwork("default", "claim1", "request1"),
			libvmi.NewInterface("default", libvmi.WithBindingPlugin(v1.PluginBinding{Name: "netbinding"})),
		),
	)

	DescribeTable("should reject a network with an invalid source count", func(ctx SpecContext, network *v1.Network) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(network),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("network source must have exactly one of pod, multus or resourceClaim"))
	},
		Entry("no source set", &v1.Network{Name: "default"}),
		Entry("both pod and multus set", &v1.Network{
			Name: "default",
			NetworkSource: v1.NetworkSource{
				Pod:    &v1.PodNetwork{},
				Multus: &v1.MultusNetwork{NetworkName: "default-nad"},
			},
		}),
		Entry("both resourceClaim and pod set", &v1.Network{
			Name: "default",
			NetworkSource: v1.NetworkSource{
				ResourceClaim: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "request1"},
				Pod:           &v1.PodNetwork{},
			},
		}),
		Entry("both resourceClaim and multus set", &v1.Network{
			Name: "default",
			NetworkSource: v1.NetworkSource{
				ResourceClaim: &v1.ClaimRequest{ClaimName: "claim1", RequestName: "request1"},
				Multus:        &v1.MultusNetwork{NetworkName: "default-nad"},
			},
		}),
	)
})
