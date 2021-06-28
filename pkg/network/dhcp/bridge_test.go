package dhcp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
)

const (
	ifaceName   = "eth0"
	launcherPID = "self"
)

var _ = Describe("Bridge DHCP configurator", func() {

	var generator BridgeConfigGenerator

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
