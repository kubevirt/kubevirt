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

// These specs exercise the Interface-level +kubebuilder:validation:XValidation (CEL)
// rules guarding portRanges' structural shape (mutual exclusivity with ports, the
// masquerade-only requirement, and overlap detection), plus the native
// Minimum/Maximum/MaxItems/enum constraints on PortRange and Interface.PortRanges.
// The feature-gate check (PortRangesSpec) stays webhook-enforced and out of scope
// here (tracked separately in kubevirt/kubevirt#19701 as Tier 2a/Phase 1b); no
// KubeVirt webhook runs in this environment, so none of these specs exercise it.
// Like the port-forwarding structural rules above, these are now enforced
// unconditionally regardless of network type - a deliberate, acknowledged
// tightening of configuration that was previously silently ignored on non-pod
// networks (see the commit message for why).
var _ = Describe("PortRanges structural CRD validation (unconditional)", func() {
	It("should accept a well-formed portRange on a masquerade pod-network interface", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
				libvmi.WithPortRanges(v1.PortRange{Protocol: "TCP", Start: 80, End: 90}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject both ports and portRanges set on the same interface, even on a non-pod network", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(),
				libvmi.WithPorts(v1.Port{Port: 22}),
				libvmi.WithPortRanges(v1.PortRange{Protocol: "TCP", Start: 80, End: 90}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
	})

	It("should reject portRanges on a non-masquerade interface on a non-pod network (previously silently accepted)", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithSRIOVBinding(),
				libvmi.WithPortRanges(v1.PortRange{Protocol: "TCP", Start: 80, End: 90}))),
			libvmi.WithNetwork(libvmi.MultusNetwork("default", "default-nad")),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("should reject a malformed portRange on a masquerade pod-network interface",
		func(ctx SpecContext, portRange v1.PortRange) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
					libvmi.WithPortRanges(portRange))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
		},
		Entry("unsupported protocol", v1.PortRange{Protocol: "SCTP", Start: 80, End: 90}),
		Entry("empty protocol", v1.PortRange{Protocol: "", Start: 80, End: 90}),
		Entry("start below range", v1.PortRange{Protocol: "TCP", Start: 0, End: 90}),
		Entry("end above range", v1.PortRange{Protocol: "TCP", Start: 80, End: 65536}),
		Entry("start greater than end", v1.PortRange{Protocol: "TCP", Start: 100, End: 80}),
	)

	It("should reject more than 10 portRanges on an interface", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		portRanges := make([]v1.PortRange, 11)
		for i := range portRanges {
			portRanges[i] = v1.PortRange{Protocol: "TCP", Start: int32(1000 + i*10), End: int32(1005 + i*10)}
		}
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithPortRanges(portRanges...))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].portRanges"))
	})

	DescribeTable("should accept non-overlapping portRanges",
		func(ctx SpecContext, portRanges []v1.PortRange) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithPortRanges(portRanges...))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("single range", []v1.PortRange{{Protocol: "TCP", Start: 80, End: 90}}),
		Entry("single port (start == end)", []v1.PortRange{{Protocol: "TCP", Start: 22, End: 22}}),
		Entry("two non-overlapping TCP ranges",
			[]v1.PortRange{{Protocol: "TCP", Start: 80, End: 100}, {Protocol: "TCP", Start: 200, End: 300}}),
		Entry("TCP and UDP ranges that overlap numerically (allowed, different protocols)",
			[]v1.PortRange{{Protocol: "TCP", Start: 80, End: 200}, {Protocol: "UDP", Start: 150, End: 300}}),
	)

	DescribeTable("should reject overlapping portRanges of the same protocol",
		func(ctx SpecContext, portRanges []v1.PortRange) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(), libvmi.WithPortRanges(portRanges...))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
		},
		Entry("two overlapping TCP ranges",
			[]v1.PortRange{{Protocol: "TCP", Start: 80, End: 200}, {Protocol: "TCP", Start: 150, End: 300}}),
		Entry("two exactly-duplicate TCP ranges",
			[]v1.PortRange{{Protocol: "TCP", Start: 80, End: 90}, {Protocol: "TCP", Start: 80, End: 90}}),
		Entry("two TCP ranges touching at a single port (end of one equals start of the other)",
			[]v1.PortRange{{Protocol: "TCP", Start: 80, End: 100}, {Protocol: "TCP", Start: 100, End: 150}}),
	)
})
