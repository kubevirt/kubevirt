package dhcp

import (
	"fmt"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

var _ = Describe("DHCP configurator", func() {
	const (
		advertisingCIDR    = "10.10.10.0/24"
		bridgeName         = "br0"
		ifaceName          = "eth0"
		launcherPID        = "self"
		fakeDhcpStartedDir = "/tmp/dhcpStartedPath"
	)

	var cacheCreator tempCacheCreator

	BeforeEach(func() {
		// make sure the test can write a file in the whatever dir ensure uses.
		Expect(os.MkdirAll(fakeDhcpStartedDir, 0o755)).To(Succeed())
	})

	AfterEach(func() {
		Expect(cacheCreator.New("").Delete()).To(Succeed())
		Expect(os.RemoveAll(fakeDhcpStartedDir)).To(Succeed())
	})

	newBridgeConfigurator := func(advertisingIfaceName string) *configurator {
		configurator := NewBridgeConfigurator(&cacheCreator, launcherPID, advertisingIfaceName, netdriver.NewMockNetworkHandler(gomock.NewController(GinkgoT())), "", nil, nil, "")
		configurator.dhcpStartedDirectory = fakeDhcpStartedDir
		return configurator
	}

	newMasqueradeConfigurator := func(advertisingIfaceName string) *configurator {
		configurator := NewMasqueradeConfigurator(advertisingIfaceName, netdriver.NewMockNetworkHandler(gomock.NewController(GinkgoT())), nil, nil, "", "")
		configurator.dhcpStartedDirectory = fakeDhcpStartedDir
		return configurator
	}

	Context("start DHCP function", func() {
		var advertisingAddr netlink.Addr
		var dhcpConfig cache.DHCPConfig
		var dhcpOptions *v1.DHCPOptions

		BeforeEach(func() {
			addr, err := netlink.ParseAddr(advertisingCIDR)
			Expect(err).NotTo(HaveOccurred())
			advertisingAddr = *addr
		})

		BeforeEach(func() {
			dhcpConfig = cache.DHCPConfig{
				Name:              ifaceName,
				IP:                netlink.Addr{},
				AdvertisingIPAddr: advertisingAddr.IP,
				Mtu:               1400,
				IPAMDisabled:      false,
			}
		})

		DescribeTable("should succeed when DHCP server started", func(f func(advertisingIfaceName string) *configurator) {
			cfg := f(bridgeName)
			cfg.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(nil)

			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			Entry("with bridge configurator", newBridgeConfigurator),
			Entry("with masquerade configurator", newMasqueradeConfigurator),
		)

		DescribeTable("should succeed when DHCP server is started multiple times", func(f func(advertisingIfaceName string) *configurator) {
			cfg := f(bridgeName)
			cfg.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(nil)

			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			Entry("with bridge configurator", newBridgeConfigurator),
			Entry("with masquerade configurator", newMasqueradeConfigurator),
		)

		DescribeTable("should fail when DHCP server failed", func(f func(advertisingIfaceName string) *configurator) {
			cfg := f(bridgeName)
			cfg.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(fmt.Errorf("failed to start DHCP server"))

			Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(HaveOccurred())
		},
			Entry("with bridge configurator", newBridgeConfigurator),
			Entry("with masquerade configurator", newMasqueradeConfigurator),
		)

		When("IPAM is disabled on the DHCPConfig", func() {
			BeforeEach(func() {
				dhcpConfig = cache.DHCPConfig{
					Name:              ifaceName,
					IP:                netlink.Addr{},
					AdvertisingIPAddr: advertisingAddr.IP,
					Mtu:               1400,
					IPAMDisabled:      true,
				}
			})

			DescribeTable("shouldn't fail when DHCP server failed", func(f func(advertisingIfaceName string) *configurator) {
				cfg := f(bridgeName)
				cfg.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(nil).Times(0)

				Expect(cfg.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			},
				Entry("with bridge configurator", newBridgeConfigurator),
				Entry("with masquerade", newMasqueradeConfigurator),
			)
		})
	})
})
