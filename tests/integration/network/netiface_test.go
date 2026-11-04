package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

// These specs exercise the +kubebuilder:validation:items:MaxLength bound on
// DHCPOptions.NTPServers (staging/src/kubevirt.io/api/core/v1/schema.go). At
// this point in the migration, the admission webhook still enforces the
// actual IPv4 format - only the length bound is schema-enforced.
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

	It("should reject an NTP server address exceeding the max length", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		// 18 non-IP characters: longer than any valid dotted-quad IPv4 string
		// (max 15) and not shaped like one, so this isolates the length bound
		// from the still-active webhook's separate IPv4 format check.
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
})
