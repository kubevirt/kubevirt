package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +kubebuilder:validation:MaxLength/
// +kubebuilder:validation:XValidation (CEL) pair, plus
// +kubebuilder:validation:MaxItems, on DHCPOptions.NTPServers
// (staging/src/kubevirt.io/api/core/v1/schema.go), replacing the webhook's
// net.ParseIP(ip).To4() check. No KubeVirt webhook runs in this environment,
// so a rejection here can only come from the CRD-embedded constraint.
var _ = Describe("DHCP NTP servers CRD validation", func() {
	It("should accept a well-formed list of IPv4 NTP servers", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
				libvmi.WithDHCPOptions(v1.DHCPOptions{NTPServers: []string{"127.0.0.1", "192.168.1.1"}}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should accept an interface with no NTP servers", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
				libvmi.WithDHCPOptions(v1.DHCPOptions{}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should reject a malformed NTP server address",
		func(ctx SpecContext, ntpServer string) {
			ns := f.CreateNamespace(ctx)

			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
					libvmi.WithDHCPOptions(v1.DHCPOptions{NTPServers: []string{ntpServer}}))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			// The CEL rule is a self.all() comprehension over the whole array, so the
			// apiserver cannot attribute the failure to a specific index - unlike the
			// per-item struct fields (e.g. Port.Port), the error is reported against
			// the array field itself, not an indexed element.
			Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].dhcpOptions.ntpServers"))
			// Assert the CEL rule's own message, not just the field path, so this
			// fails loudly if some other validator ever fires on this same path
			// instead of the intended format rule.
			Expect(err.Error()).To(ContainSubstring("NTP servers must be a list of valid IPv4 addresses"))
		},
		Entry("IPv6 address", "::1"),
		Entry("hostname", "hostname"),
		Entry("out-of-range octet", "999.1.1.1"),
		Entry("leading zero octet", "127.0.0.01"),
		Entry("too many octets", "1.2.3.4.5"),
		Entry("IPv4-mapped IPv6 notation", "::ffff:1.2.3.4"),
	)

	It("should reject an NTP server address exceeding the max length (rejected by length, not the CEL format rule)", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		// 18 non-IP characters: longer than any valid dotted-quad IPv4 string
		// (max 15) and not shaped like one, so this isolates the length bound
		// from the CEL format rule's own message.
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
				libvmi.WithDHCPOptions(v1.DHCPOptions{NTPServers: []string{"not-an-ip-address"}}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Too long"))
	})

	It("should reject more than 32 NTP servers on an interface", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		ntpServers := make([]string, 33)
		for i := range ntpServers {
			ntpServers[i] = "127.0.0.1"
		}
		vmi := libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
			libvmi.WithInterface(libvmi.NewInterface("default", libvmi.WithMasqueradeBinding(),
				libvmi.WithDHCPOptions(v1.DHCPOptions{NTPServers: ntpServers}))),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, vmi, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("spec.domain.devices.interfaces[0].dhcpOptions.ntpServers"))
	})
})
