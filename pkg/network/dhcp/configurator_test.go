package dhcp

import (
	"fmt"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
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

	BeforeEach(func() {
		// make sure the test can write a file in the whatever dir ensure uses.
		Expect(os.MkdirAll(fakeDhcpStartedDir, 0755)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(fakeDhcpStartedDir)).To(Succeed())
	})

	newConfiguratorWithClientFilter := func(launcherPID string, advertisingIfaceName string) *Configurator {
		configurator := NewConfiguratorWithClientFilter(fake.NewFakeInMemoryNetworkCacheFactory(), launcherPID, advertisingIfaceName, netdriver.NewMockNetworkHandler(gomock.NewController(GinkgoT())))
		configurator.dhcpStartedDirectory = fakeDhcpStartedDir
		return configurator
	}

	newConfigurator := func(launcherPID string, advertisingIfaceName string) *Configurator {
		configurator := NewConfigurator(fake.NewFakeInMemoryNetworkCacheFactory(), launcherPID, advertisingIfaceName, netdriver.NewMockNetworkHandler(gomock.NewController(GinkgoT())))
		configurator.dhcpStartedDirectory = fakeDhcpStartedDir
		return configurator
	}

	Context("importing / exporting configuration", func() {
		table.DescribeTable("should fail when nothing to load", func(configurator *Configurator) {
			_, err := configurator.ImportConfiguration(ifaceName)
			Expect(err).To(HaveOccurred())
		},
			table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
			table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
		)

		table.DescribeTable("should succeed when cache file present", func(configurator *Configurator) {
			dhcpConfig := cache.DhcpConfig{
				Name:         ifaceName,
				IP:           netlink.Addr{},
				Mtu:          1400,
				IPAMDisabled: false,
			}
			Expect(configurator.ExportConfiguration(dhcpConfig)).To(Succeed())

			readConfig, err := configurator.ImportConfiguration(ifaceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(*readConfig).To(Equal(dhcpConfig))
		},
			table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
			table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
		)
	})

	Context("start DHCP function", func() {
		var advertisingAddr netlink.Addr
		var dhcpConfig cache.DhcpConfig
		var dhcpOptions *v1.DHCPOptions

		BeforeEach(func() {
			addr, err := netlink.ParseAddr(advertisingCIDR)
			Expect(err).NotTo(HaveOccurred())
			advertisingAddr = *addr
		})

		BeforeEach(func() {
			dhcpConfig = cache.DhcpConfig{
				Name:         ifaceName,
				IP:           netlink.Addr{},
				Gateway:      advertisingAddr.IP,
				Mtu:          1400,
				IPAMDisabled: false,
			}
		})

		table.DescribeTable("should succeed when DHCP server started", func(configurator *Configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, advertisingAddr.IP, bridgeName, nil, configurator.filterByMac).Return(nil)

			Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
			table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
		)

		table.DescribeTable("should succeed when DHCP server started", func(configurator *Configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, advertisingAddr.IP, bridgeName, nil, configurator.filterByMac).Return(nil)

			Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
			table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
		)

		table.DescribeTable("should succeed when DHCP server is started multiple times", func(configurator *Configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, advertisingAddr.IP, bridgeName, nil, configurator.filterByMac).Return(nil)

			Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
		},
			table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
			table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
		)

		table.DescribeTable("should fail when DHCP server failed", func(configurator *Configurator) {
			configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, advertisingAddr.IP, bridgeName, nil, configurator.filterByMac).Return(fmt.Errorf("failed to start DHCP server"))

			Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(HaveOccurred())
		},
			table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
			table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
		)

		When("IPAM is disabled on the DhcpConfig", func() {
			BeforeEach(func() {
				dhcpConfig = cache.DhcpConfig{
					Name:         ifaceName,
					IP:           netlink.Addr{},
					Gateway:      advertisingAddr.IP,
					Mtu:          1400,
					IPAMDisabled: true,
				}
			})

			table.DescribeTable("should fail when DHCP server failed", func(configurator *Configurator) {
				configurator.handler.(*netdriver.MockNetworkHandler).EXPECT().StartDHCP(&dhcpConfig, advertisingAddr.IP, bridgeName, nil, configurator.filterByMac).Return(nil).Times(0)

				Expect(configurator.EnsureDhcpServerStarted(ifaceName, dhcpConfig, dhcpOptions)).To(Succeed())
			},
				table.Entry("with active client filtering", newConfiguratorWithClientFilter(launcherPID, bridgeName)),
				table.Entry("without client filtering", newConfigurator(launcherPID, bridgeName)),
			)
		})
	})
})
