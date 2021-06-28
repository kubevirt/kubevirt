package network

import (
	"fmt"
	"net"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	"kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("podNIC", func() {
	var (
		mockNetwork                *netdriver.MockNetworkHandler
		cacheFactory               cache.InterfaceCacheFactory
		mockPodNetworkConfigurator *infraconfigurators.MockPodNetworkInfraConfigurator
		mockDHCPConfigurator       *dhcp.MockConfigurator
		ctrl                       *gomock.Controller
	)

	newPhase1PodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		launcherPID := 1
		podnic, err := newPodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &launcherPID)
		if err != nil {
			return nil, err
		}
		podnic.infraConfigurator = mockPodNetworkConfigurator
		return podnic, nil
	}
	newPhase2PodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		podnic, err := newPodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, nil)
		if err != nil {
			return nil, err
		}
		podnic.dhcpConfigurator = mockDHCPConfigurator
		return podnic, nil
	}
	BeforeEach(func() {
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mockPodNetworkConfigurator = infraconfigurators.NewMockPodNetworkInfraConfigurator(ctrl)
		mockDHCPConfigurator = dhcp.NewMockConfigurator(ctrl)
	})
	AfterEach(func() {
		ctrl.Finish()
	})
	When("reading networking configuration succeed", func() {
		var (
			podnic *podNIC
			vmi    *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			address1 := &net.IPNet{IP: net.IPv4(1, 2, 3, 4)}
			address2 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0)}
			fakeAddr1 := netlink.Addr{IPNet: address1}
			fakeAddr2 := netlink.Addr{IPNet: address2}
			mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return(fakeAddr1.IP.String(), fakeAddr2.IP.String(), nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)
			mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(primaryPodInterfaceName).Times(1)
			mockPodNetworkConfigurator.EXPECT().GenerateNonRecoverableDHCPConfig().Return(&cache.DHCPConfig{}).Times(1)
			mockPodNetworkConfigurator.EXPECT().GenerateDomainIfaceSpec().Times(1)
			domain := NewDomainWithBridgeInterface()
			vmi = newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			var err error
			podnic, err = newPhase1PodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())

		})
		Context("and networking preparation fails", func() {
			BeforeEach(func() {
				mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Return(fmt.Errorf("podnic_test: forcing prepare networking failure")).Times(1)
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
				mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Times(1)
			})
			It("should return no error at phase1 and store pod interface", func() {
				Expect(podnic.PlugPhase1()).To(Succeed())
				podData, err := podnic.cacheFactory.CacheForVMI(vmi).Read("default")
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
			podnic.cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Write(podnic.podInterfaceName, &cache.DHCPConfig{Name: podnic.podInterfaceName})
			podnic.cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Write(podnic.vmiSpecIface.Name, &domain.Spec.Devices.Interfaces[0])
		})
		Context("and starting the DHCP server fails", func() {
			BeforeEach(func() {
				mockDHCPConfigurator.EXPECT().Generate().Return(&cache.DHCPConfig{}, nil)
				mockDHCPConfigurator.EXPECT().EnsureDHCPServerStarted(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("Fake EnsureDHCPServerStarted failure"))
			})
			It("phase2 should panic", func() {
				Expect(func() { podnic.PlugPhase2(domain) }).To(Panic())
			})
		})
		Context("and starting the DHCP server succeed", func() {
			BeforeEach(func() {
				dhcpConfig := &cache.DHCPConfig{}
				mockDHCPConfigurator.EXPECT().Generate().Return(dhcpConfig, nil)
				mockDHCPConfigurator.EXPECT().EnsureDHCPServerStarted(primaryPodInterfaceName, *dhcpConfig, vmi.Spec.Domain.Devices.Interfaces[0].DHCPOptions).Return(nil)
			})
			It("phase2 should succeed", func() {
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			})

		})
	})
	When("interface binding is SRIOV", func() {
		var (
			vmi *v1.VirtualMachineInstance
		)
		BeforeEach(func() {

			vmi = newVMI("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			}}
		})
		Context("phase1", func() {
			var (
				podnic *podNIC
				err    error
			)
			BeforeEach(func() {
				podnic, err = newPhase1PodNICWithMocks(vmi)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should not crash", func() {
				Expect(podnic.PlugPhase1()).To(Succeed())
			})
		})
		Context("phase2", func() {
			var (
				podnic *podNIC
				err    error
			)
			BeforeEach(func() {
				podnic, err = newPhase2PodNICWithMocks(vmi)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should not crash", func() {
				Expect(podnic.PlugPhase2(&api.Domain{})).To(Succeed())
			})
		})
	})
})
