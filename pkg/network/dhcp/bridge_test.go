package dhcp

import (
	"github.com/vishvananda/netlink"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

const (
	ifaceName   = "eth0"
	launcherPID = "self"
	subdomain   = "subdomain"
)

var _ = Describe("Bridge DHCP configurator", func() {

	var mockHandler *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var generator BridgeConfigGenerator
	var cacheCreator tempCacheCreator

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		ctrl = gomock.NewController(GinkgoT())
		mockHandler = netdriver.NewMockNetworkHandler(ctrl)
	})

	AfterEach(func() {
		Expect(cacheCreator.New("").Delete()).To(Succeed())
	})

	Context("Generate", func() {
		It("Should fail", func() {
			generator = BridgeConfigGenerator{
				cacheCreator: &cacheCreator,
			}
			config, err := generator.Generate()
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
		})
		It("Should succeed with ipam", func() {
			Expect(cache.WriteDHCPInterfaceCache(
				&cacheCreator, launcherPID, ifaceName, &cache.DHCPConfig{IPAMDisabled: false},
			)).To(Succeed())

			iface := v1.Interface{Name: "network"}
			generator = BridgeConfigGenerator{
				cacheCreator:     &cacheCreator,
				launcherPID:      launcherPID,
				podInterfaceName: ifaceName,
				vmiSpecIfaces:    []v1.Interface{iface},
				vmiSpecIface:     &iface,
				handler:          mockHandler,
				subdomain:        subdomain,
			}

			mtu := 1410
			link := &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
			mockHandler.EXPECT().LinkByName(virtnetlink.GenerateNewBridgedVmiInterfaceName(ifaceName)).Return(link, nil)

			config, err := generator.Generate()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := cache.DHCPConfig{Name: ifaceName}
			expectedConfig.IPAMDisabled = false
			fakeBridgeIP := virtnetlink.GetFakeBridgeIP([]v1.Interface{iface}, &iface)
			advertisingIPAddr, _ := netlink.ParseAddr(fakeBridgeIP)
			expectedConfig.AdvertisingIPAddr = advertisingIPAddr.IP
			expectedConfig.Mtu = 1410
			expectedConfig.Subdomain = subdomain
			Expect(*config).To(Equal(expectedConfig))
		})
		It("Should succeed with no ipam", func() {
			Expect(cache.WriteDHCPInterfaceCache(
				&cacheCreator, launcherPID, ifaceName, &cache.DHCPConfig{IPAMDisabled: true},
			)).To(Succeed())

			generator = BridgeConfigGenerator{
				cacheCreator:     &cacheCreator,
				launcherPID:      launcherPID,
				podInterfaceName: ifaceName,
				subdomain:        subdomain,
			}
			config, err := generator.Generate()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := cache.DHCPConfig{IPAMDisabled: true}
			Expect(*config).To(Equal(expectedConfig))
		})
	})
})
