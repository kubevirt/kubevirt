package network

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
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

var _ = Describe("podNIC", func() {
	var (
		mockNetwork                *netdriver.MockNetworkHandler
		cacheFactory               cache.InterfaceCacheFactory
		mockPodNetworkConfigurator *infraconfigurators.MockPodNetworkInfraConfigurator
		ctrl                       *gomock.Controller
		vmi                        *kubevirtv1.VirtualMachineInstance
	)
	newPodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		podnic, err := newPodNICWithWithoutInfraConfigurator(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, nil)
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

			podnic, err := newPodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())
			err = podnic.PlugPhase1()
			Expect(err).To(HaveOccurred(), "SetupPhase1 should return an error")

			_, ok := err.(*neterrors.CriticalNetworkError)
			Expect(ok).To(BeTrue(), "SetupPhase1 should return an error of type CriticalNetworkError")
		})
	})
	Context("when DHCP startup fails", func() {
		var (
			tmpDir string
			podnic *podNIC
			domain *api.Domain
		)
		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir(os.TempDir(), "dhcp")
			Expect(err).ToNot(HaveOccurred())
			mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("podnic_test: forcing failure at DHCP start"))
			domain = NewDomainWithBridgeInterface()
			vmi := newVMIBridgeInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			podnic, err = newPodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())
			podnic.dhcpConfigurator.DHCPStartedDirectory = tmpDir
			podnic.storeCachedDomainIface(domain.Spec.Devices.Interfaces[0])
			podnic.dhcpConfigurator.ExportConfiguration(cache.DHCPConfig{Name: "eth0"})

		})
		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})
		It("phase2 should panic", func() {
			Expect(func() { podnic.PlugPhase2(domain) }).To(Panic())
		})
	})
	Context("when setPodInterfaceCache is called with a configured nic", func() {
		var (
			podnic *podNIC
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

			var err error
			podnic, err = newPodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())
			err = podnic.setPodInterfaceCache()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should write interface to cache file", func() {
			podData, err := podnic.cacheFactory.CacheForVMI(vmi).Read("default")
			Expect(err).ToNot(HaveOccurred())
			Expect(podData.PodIP).To(Equal("1.2.3.4"))
			Expect(podData.PodIPs).To(ConsistOf("1.2.3.4", "169.254.0.0"))
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
