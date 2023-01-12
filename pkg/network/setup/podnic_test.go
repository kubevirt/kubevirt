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
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("podNIC", func() {
	const (
		masqueradeCidr     = "10.0.2.0/30"
		masqueradeIpv6Cidr = "fd10:0:2::0/120"
	)
	var (
		mockNetwork                *netdriver.MockNetworkHandler
		baseCacheCreator           tempCacheCreator
		mockPodNetworkConfigurator *infraconfigurators.MockPodNetworkInfraConfigurator
		mockDHCPConfigurator       *dhcp.MockConfigurator
		ctrl                       *gomock.Controller
	)

	newPhase1PodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		launcherPID := 1
		podnic, err := newPodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, &baseCacheCreator, &launcherPID)
		if err != nil {
			return nil, err
		}
		podnic.infraConfigurator = mockPodNetworkConfigurator
		return podnic, nil
	}
	newPhase2PodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		podnic, err := newPodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, &baseCacheCreator, nil)
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
		mockPodNetworkConfigurator = infraconfigurators.NewMockPodNetworkInfraConfigurator(ctrl)
		mockDHCPConfigurator = dhcp.NewMockConfigurator(ctrl)
	})
	AfterEach(func() {
		Expect(baseCacheCreator.New("").Delete()).To(Succeed())
	})
	When("reading networking configuration succeed", func() {
		var (
			podnic *podNIC
			vmi    *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			mockNetwork.EXPECT().ReadIPAddressesFromLink(namescheme.PrimaryPodInterfaceName).Return("1.2.3.4", "169.254.0.0", nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil)
		})

		BeforeEach(func() {
			mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(namescheme.PrimaryPodInterfaceName)
			mockPodNetworkConfigurator.EXPECT().GenerateNonRecoverableDHCPConfig().Return(&cache.DHCPConfig{})
			mockPodNetworkConfigurator.EXPECT().GenerateNonRecoverableDomainIfaceSpec()
		})

		BeforeEach(func() {
			domain := NewDomainWithBridgeInterface()
			vmi = newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			var err error
			podnic, err = newPhase1PodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())

		})
		Context("and networking preparation fails", func() {
			BeforeEach(func() {
				mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Return(fmt.Errorf("podnic_test: forcing prepare networking failure"))
			})
			It("should return a CriticalNetworkError at phase1", func() {
				err := podnic.PlugPhase1()
				Expect(err).To(HaveOccurred(), "SetupPhase1 should return an error")

				_, ok := err.(*neterrors.CriticalNetworkError)
				Expect(ok).To(BeTrue(), "SetupPhase1 should return an error of type CriticalNetworkError")
			})
		})
		Context("and networking preparation success", func() {
			BeforeEach(func() {
				mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface()
			})
			It("should return no error at phase1 and store pod interface", func() {
				Expect(podnic.PlugPhase1()).To(Succeed())
				var podData *cache.PodIfaceCacheData
				podData, err := cache.ReadPodInterfaceCache(podnic.cacheCreator, string(vmi.UID), "default")
				Expect(err).ToNot(HaveOccurred())
				Expect(podData.PodIP).To(Equal("1.2.3.4"))
				Expect(podData.PodIPs).To(ConsistOf("1.2.3.4", "169.254.0.0"))
			})
		})
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
