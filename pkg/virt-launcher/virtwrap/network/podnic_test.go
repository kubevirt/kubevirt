package network

import (
	"errors"
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
		ctrl                       *gomock.Controller
	)
	newPodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		podnic, err := newPodNICWithoutInfraConfigurator(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, nil)
		if err != nil {
			return nil, err
		}
		podnic.podInterfaceName = primaryPodInterfaceName
		podnic.infraConfigurator = mockPodNetworkConfigurator
		return podnic, nil
	}
	BeforeEach(func() {
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mockPodNetworkConfigurator = infraconfigurators.NewMockPodNetworkInfraConfigurator(ctrl)
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
			mockPodNetworkConfigurator.EXPECT().GenerateDHCPConfig().Return(&cache.DHCPConfig{}).Times(1)
			mockPodNetworkConfigurator.EXPECT().GenerateDomainIfaceSpec().Times(1)
			domain := NewDomainWithBridgeInterface()
			vmi = newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			var err error
			podnic, err = newPodNICWithMocks(vmi)
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
	Context("when DHCP startup fails", func() {
		BeforeEach(func() {
			mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("podnic_test: forcing failure at DHCP start"))
		})
		It("phase2 should panic", func() {
			testDhcpPanic := func() {
				domain := NewDomainWithBridgeInterface()
				vmi := newVMIBridgeInterface("testnamespace", "testVmName")
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				podnic, err := newPodNICWithMocks(vmi)
				Expect(err).ToNot(HaveOccurred())
				podnic.storeCachedDomainIface(domain.Spec.Devices.Interfaces[0])
				podnic.dhcpConfigurator.ExportConfiguration(cache.DHCPConfig{Name: "eth0"})
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			}
			Expect(testDhcpPanic).To(Panic())
		})
	})
	Context("when interface binding is SRIOV", func() {
		var (
			vmi    *v1.VirtualMachineInstance
			podnic *podNIC
		)
		BeforeEach(func() {
			launcherPID := 1
			vmi = newVMI("testnamespace", "testVmName")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name: "default",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			}}
			var err error
			podnic, err = newPodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())
			podnic.podInterfaceName = "fakeiface"
			podnic.launcherPID = &launcherPID

		})
		It("should not crash at phase1", func() {
			Expect(podnic.PlugPhase1()).To(Succeed())
		})
		It("should not crash at phase2", func() {
			Expect(podnic.PlugPhase2(&api.Domain{})).To(Succeed())
		})
	})
})
