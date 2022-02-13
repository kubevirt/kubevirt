package dhcp

import (
	"fmt"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
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
		Expect(os.MkdirAll(fakeDhcpStartedDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		cacheCreator.New("").Delete()
		Expect(os.RemoveAll(fakeDhcpStartedDir)).To(Succeed())
	})

	newBridgeConfigurator := func(launcherPID string, advertisingIfaceName string) Configurator {
		configurator := NewBridgeConfigurator(&cacheCreator, launcherPID, advertisingIfaceName, netdriver.NewMockNetworkHandler(gomock.NewController(GinkgoT())), "", nil, nil, "")
		configurator.dhcpStartedDirectory = fakeDhcpStartedDir
		return configurator
	}

	newMasqueradeConfigurator := func(advertisingIfaceName string) Configurator {
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

		DescribeTable("should succeed when DHCP server started", func(configurator *configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(nil)

			Expect(configurator.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			Entry("with bridge configurator", newBridgeConfigurator(launcherPID, bridgeName)),
			Entry("with masquerade configurator", newMasqueradeConfigurator(bridgeName)),
		)

		DescribeTable("should succeed when DHCP server is started multiple times", func(configurator *configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(nil)

			Expect(configurator.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			Expect(configurator.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			Entry("with bridge configurator", newBridgeConfigurator(launcherPID, bridgeName)),
			Entry("with masquerade configurator", newMasqueradeConfigurator(bridgeName)),
		)

		DescribeTable("should fail when DHCP server failed", func(configurator *configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(fmt.Errorf("failed to start DHCP server"))

			Expect(configurator.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(HaveOccurred())
		},
			Entry("with bridge configurator", newBridgeConfigurator(launcherPID, bridgeName)),
			Entry("with masquerade configurator", newMasqueradeConfigurator(bridgeName)),
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

			DescribeTable("shouldn't fail when DHCP server failed", func(configurator *configurator) {
				configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, bridgeName, nil).Return(nil).Times(0)

				Expect(configurator.EnsureDHCPServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			},
				Entry("with bridge configurator", newBridgeConfigurator(launcherPID, bridgeName)),
				Entry("with masquerade", newMasqueradeConfigurator(bridgeName)),
			)
		})
	})
})
