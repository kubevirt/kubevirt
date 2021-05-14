package dhcp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
)

var _ = Describe("DHCP configurator", func() {
	var dhcpConfigurationFactory cache.InterfaceCacheFactory
	var dhcpConfigurator Configurator

	const (
		ifaceName   = "eth0"
		launcherPID = "self"
	)

	BeforeEach(func() {
		dhcpConfigurationFactory = fake.NewFakeInMemoryNetworkCacheFactory()
	})

	Context("dhcp configurator configuration handling", func() {
		BeforeEach(func() {
			dhcpConfigurator = NewConfigurator(dhcpConfigurationFactory, launcherPID)
		})

		It("should fail when nothing to load", func() {
			_, err := dhcpConfigurator.ImportConfiguration(ifaceName)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed when cache file present", func() {
			dhcpConfig := cache.DhcpConfig{
				Name:         ifaceName,
				IP:           netlink.Addr{},
				Mtu:          1400,
				IPAMDisabled: false,
			}
			Expect(dhcpConfigurator.ExportConfiguration(dhcpConfig)).To(Succeed())

			readConfig, err := dhcpConfigurator.ImportConfiguration(ifaceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*readConfig).To(Equal(dhcpConfig))
		})
	})
})
