package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +kubebuilder:validation:Minimum/+kubebuilder:validation:Maximum
// constraints on Port.Port (staging/src/kubevirt.io/api/core/v1/schema.go). Unlike the
// webhook check it replaces, this is enforced unconditionally, regardless of whether the
// interface's network is a pod network or not (a deliberate, acknowledged tightening -
// see the commit message for why). No KubeVirt webhook runs in this environment, so a
// rejection here can only come from the CRD-embedded constraint.
var _ = Describe("Port number range CRD validation (unconditional)", func() {
	It("should accept a well-formed port on a pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithPorts(v1.Port{Port: 80}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should accept a well-formed port on a non-pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Port: 80}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject a zero port on a pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithPorts(v1.Port{Port: 0}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].ports[0].port"))
	})

	DescribeTable("should reject an out-of-range port on a non-pod network (previously silently accepted)",
		func(ctx SpecContext, port int32) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Port: port}))),
				libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].ports[0].port"))
		},
		Entry("zero", int32(0)),
		Entry("above the valid range", int32(65536)),
	)
})

// These specs exercise a +kubebuilder:validation:MaxLength/+kubebuilder:validation:XValidation
// (CEL) pair on Port.Name, replicating k8s.io/apimachinery's IsValidPortName end-to-end
// (including tolerating an empty name). As above, this is enforced unconditionally
// regardless of network type.
var _ = Describe("Port name format CRD validation (unconditional)", func() {
	It("should accept an unnamed port on a non-pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Port: 80}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should accept a well-formed port name on a non-pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Name: "http-alt", Port: 80}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should reject a malformed port name on a non-pod network (previously silently accepted)",
		func(ctx SpecContext, name string) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Name: name, Port: 80}))),
				libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].ports[0].name"))
		},
		Entry("leading and trailing hyphens", "-bad-name-"),
		Entry("too long (16 characters)", "a234567890123456"),
	)
})

// These specs exercise a +kubebuilder:validation:XValidation (CEL) allow-list
// on Port.Protocol, enforced unconditionally regardless of network type.
var _ = Describe("Port protocol CRD validation (unconditional)", func() {
	It("should accept an unset protocol on a non-pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Port: 80}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should accept TCP and UDP on a non-pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(),
				libvmi.WithPorts(v1.Port{Port: 80, Protocol: "TCP"}, v1.Port{Port: 81, Protocol: "UDP"}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject an unsupported protocol on a non-pod network (previously silently accepted)", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(), libvmi.WithPorts(v1.Port{Port: 80, Protocol: "SCTP"}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].ports[0].protocol"))
	})
})

// This spec exercises the +kubebuilder:validation:MaxItems constraint on
// Interface.Ports, newly added alongside this PR's unconditional port
// validation (see commit message for why 32 specifically).
var _ = Describe("Ports list size CRD validation", func() {
	It("should reject more than 32 ports on an interface", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		ports := make([]v1.Port, 33)
		for i := range ports {
			ports[i] = v1.Port{Port: int32(1000 + i)}
		}
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithPorts(ports...))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].ports"))
	})
})
