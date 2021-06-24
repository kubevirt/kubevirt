package podnic

import (
	"errors"
	"fmt"
	"net"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"

	"k8s.io/apimachinery/pkg/types"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("PodNIC", func() {
	var (
		mockNetwork                   *netdriver.MockNetworkHandler
		cacheFactory                  cache.InterfaceCacheFactory
		mockPodNetworkConfigurator    *infraconfigurators.MockPodNetworkInfraConfigurator
		ctrl                          *gomock.Controller
		vmi                           *kubevirtv1.VirtualMachineInstance
		newPodNetworkConfiguratorMock = func(*PodNIC) (infraconfigurators.PodNetworkInfraConfigurator, error) {
			return mockPodNetworkConfigurator, nil
		}
		NewWithMocks = func(vmi *v1.VirtualMachineInstance) *PodNIC {
			podnic, err := New(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, nil)
			Expect(err).ToNot(HaveOccurred())
			podnic.podNetworkConfiguratorFactory = newPodNetworkConfiguratorMock
			podnic.podInterfaceName = primaryPodInterfaceName
			return podnic
		}
	)

	NewDomainWithBridgeInterface := func() *api.Domain {
		domain := &api.Domain{}
		domain.Spec.Devices.Interfaces = []api.Interface{{
			Model: &api.Model{
				Type: "virtio",
			},
			Type: "bridge",
			Source: api.InterfaceSource{
				Bridge: api.DefaultBridgeName,
			},
			Alias: api.NewUserDefinedAlias("default"),
		},
		}
		return domain
	}

	newVMI := func(namespace, name string) *v1.VirtualMachineInstance {
		vmi := v1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		return vmi
	}

	newVMIBridgeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := newVMI(namespace, name)
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}

	newVMIMasqueradeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := newVMI(namespace, name)
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
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

	Context("when pod networking setup fails", func() {
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
			mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Return(fmt.Errorf("podnic_test: forcing prepare networking failure")).Times(1)
		})
		It("should return a CriticalNetworkError at phase1", func() {

			domain := NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			podnic := NewWithMocks(vmi)
			//podnic.launcherPID = &pid
			err := podnic.PlugPhase1()
			Expect(err).To(HaveOccurred(), "SetupPhase1 should return an error")

			_, ok := err.(*neterrors.CriticalNetworkError)
			Expect(ok).To(BeTrue(), "SetupPhase1 should return an error of type CriticalNetworkError")
		})
	})
	Context("when DHCP startup fails", func() {
		BeforeEach(func() {
			mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("podnic_test: forcing failure at DHCP start"))
		})
		It("phase2 should panic", func() {
			testDhcpPanic := func() {
				launcherPID := 1
				domain := NewDomainWithBridgeInterface()
				vmi := newVMIBridgeInterface("testnamespace", "testVmName")
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				podnic := NewWithMocks(vmi)
				podnic.launcherPID = &launcherPID
				podnic.storeCachedDomainIface(domain.Spec.Devices.Interfaces[0])
				podnic.dhcpConfigurator.ExportConfiguration(cache.DHCPConfig{Name: "eth0"})
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			}
			Expect(testDhcpPanic).To(Panic())
		})
	})
	Context("when setPodInterfaceCache is called with a configured nic", func() {
		var (
			podnic *PodNIC
		)
		BeforeEach(func() {
			vmi = newVMIMasqueradeInterface("testnamespace", "testVmName")
			vmi.UID = types.UID("test-1234")
			address1 := &net.IPNet{IP: net.IPv4(1, 2, 3, 4)}
			address2 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0)}
			fakeAddr1 := netlink.Addr{IPNet: address1}
			fakeAddr2 := netlink.Addr{IPNet: address2}

			mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return(fakeAddr1.IP.String(), fakeAddr2.IP.String(), nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			podnic = NewWithMocks(vmi)
			err := podnic.setPodInterfaceCache()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should write interface to cache file", func() {
			podData, err := podnic.cacheFactory.CacheForVMI(vmi).Read("default")
			Expect(err).ToNot(HaveOccurred())
			Expect(podData.PodIP).To(Equal("1.2.3.4"))
			Expect(podData.PodIPs).To(Equal([]string{"1.2.3.4", "169.254.0.0"}))
		})
	})
	Context("when interface binding is SRIOV", func() {
		var (
			vmi    *v1.VirtualMachineInstance
			podnic *PodNIC
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
			podnic = NewWithMocks(vmi)
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
