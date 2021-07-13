package dhcp

import (
	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

const (
	ifaceName   = "eth0"
	launcherPID = "self"
)

var _ = Describe("Bridge DHCP configurator", func() {

	var mockHandler *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var generator BridgeConfigGenerator

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHandler = netdriver.NewMockNetworkHandler(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Generate", func() {
		var cacheFactory cache.InterfaceCacheFactory
		BeforeEach(func() {
			cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
		})
		It("Should fail", func() {
			generator = BridgeConfigGenerator{
				cacheFactory: cacheFactory,
			}
			config, err := generator.Generate()
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
		})
		It("Should succeed with ipam", func() {
			Expect(cacheFactory.CacheDHCPConfigForPid(launcherPID).Write(ifaceName, &cache.DHCPConfig{IPAMDisabled: false})).To(Succeed())

			iface := v1.Interface{Name: "network"}
			generator = BridgeConfigGenerator{
				cacheFactory:     cacheFactory,
				launcherPID:      launcherPID,
				podInterfaceName: ifaceName,
				vmiSpecIfaces:    []v1.Interface{iface},
				vmiSpecIface:     &iface,
				handler:          mockHandler,
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
			Expect(*config).To(Equal(expectedConfig))
		})
		It("Should succeed with no ipam", func() {
			Expect(cacheFactory.CacheDHCPConfigForPid(launcherPID).Write(ifaceName, &cache.DHCPConfig{IPAMDisabled: true})).To(Succeed())

			generator = BridgeConfigGenerator{
				cacheFactory:     cacheFactory,
				launcherPID:      launcherPID,
				podInterfaceName: ifaceName,
			}
			config, err := generator.Generate()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := cache.DHCPConfig{IPAMDisabled: true}
			Expect(*config).To(Equal(expectedConfig))
		})
	})
})
