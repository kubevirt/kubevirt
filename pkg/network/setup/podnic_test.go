package network

import (
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("podNIC", func() {
	const (
		masqueradeCidr     = "10.0.2.0/30"
		masqueradeIpv6Cidr = "fd10:0:2::0/120"
	)
	var (
		mockNetwork          *netdriver.MockNetworkHandler
		baseCacheCreator     tempCacheCreator
		mockDHCPConfigurator *dhcp.MockConfigurator
		ctrl                 *gomock.Controller
	)

	newPhase2PodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		podnic, err := newPodNIC(vmi, &vmi.Spec.Networks[0], &vmi.Spec.Domain.Devices.Interfaces[0], mockNetwork, &baseCacheCreator, nil)
		if err != nil {
			return nil, err
		}
		podnic.dhcpConfigurator = mockDHCPConfigurator
		return podnic, nil
	}
	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mockDHCPConfigurator = dhcp.NewMockConfigurator(ctrl)
	})
	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})

	When("DHCP config is correctly read", func() {
		var (
			podnic *podNIC
			domain *api.Domain
			vmi    *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			var err error
			domain = NewDomainWithBridgeInterface()
			vmi = newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			podnic, err = newPhase2PodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(
				cache.WriteDHCPInterfaceCache(
					podnic.cacheCreator,
					getPIDString(podnic.launcherPID),
					podnic.podInterfaceName,
					&cache.DHCPConfig{Name: podnic.podInterfaceName},
				),
			).To(Succeed())
			Expect(
				cache.WriteDomainInterfaceCache(
					podnic.cacheCreator,
					getPIDString(podnic.launcherPID),
					podnic.vmiSpecIface.Name,
					&domain.Spec.Devices.Interfaces[0],
				),
			).To(Succeed())
		})
		Context("and starting the DHCP server fails", func() {
			BeforeEach(func() {
				mockDHCPConfigurator.EXPECT().Generate().Return(&cache.DHCPConfig{}, nil)
				mockDHCPConfigurator.EXPECT().EnsureDHCPServerStarted(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("Fake EnsureDHCPServerStarted failure"))
				podnic.domainGenerator = &fakeLibvirtSpecGenerator{
					shouldGenerateFail: false,
				}
			})
			It("phase2 should panic", func() {
				Expect(func() { _ = podnic.PlugPhase2(domain) }).To(Panic())
			})
		})
		Context("and starting the DHCP server succeed", func() {
			BeforeEach(func() {
				dhcpConfig := &cache.DHCPConfig{}
				mockDHCPConfigurator.EXPECT().Generate().Return(dhcpConfig, nil)
				mockDHCPConfigurator.EXPECT().EnsureDHCPServerStarted(namescheme.PrimaryPodInterfaceName, *dhcpConfig, vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions).Return(nil)
				podnic.domainGenerator = &fakeLibvirtSpecGenerator{
					shouldGenerateFail: false,
				}
				podnic.podInterfaceName = namescheme.PrimaryPodInterfaceName
			})
			It("phase2 should succeed", func() {
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			})

		})
	})
})

type fakeLibvirtSpecGenerator struct {
	shouldGenerateFail bool
}

func (b *fakeLibvirtSpecGenerator) Generate() error {
	if b.shouldGenerateFail {
		return fmt.Errorf("Fake LibvirtSpecGenerator.Generate failure")
	}
	return nil

}
