package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"

	"github.com/golang/mock/gomock"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	dhcpconfigurator "kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("podNIC", func() {
	var (
		mockNetwork                *netdriver.MockNetworkHandler
		cacheFactory               cache.InterfaceCacheFactory
		mockPodNetworkConfigurator *infraconfigurators.MockPodNetworkInfraConfigurator
		ctrl                       *gomock.Controller
		tmpDir                     string
	)
	newPodNICWithMocks := func(vmi *v1.VirtualMachineInstance) (*podNIC, error) {
		launcherPID := 1
		podnic, err := newPodNICWithoutInfraConfigurator(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &launcherPID)
		if err != nil {
			return nil, err
		}
		podnic.podInterfaceName = primaryPodInterfaceName
		podnic.infraConfigurator = mockPodNetworkConfigurator
		if podnic.dhcpConfigurator != nil {
			podnic.dhcpConfigurator = dhcpconfigurator.NewConfiguratorWithDHCPStartedDirectory(cacheFactory, getPIDString(podnic.launcherPID), primaryPodInterfaceName, mockNetwork, tmpDir)
		}
		return podnic, nil
	}
	BeforeEach(func() {
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mockPodNetworkConfigurator = infraconfigurators.NewMockPodNetworkInfraConfigurator(ctrl)
	})
	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "dhcp")
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		ctrl.Finish()
		os.RemoveAll(tmpDir)
	})
	type result struct {
		err error
	}
	withSucces := func() *result {
		return &result{}
	}
	withError := func() *result {
		return &result{
			err: fmt.Errorf("forced failure"),
		}
	}
	type plugPhase1Case struct {
		vmi                      *v1.VirtualMachineInstance
		storedDomainIface        *api.Interface
		ipv4Addr                 string
		ipv6Addr                 string
		isIPv4Primary            bool
		shouldDiscoverNetworking *result
		shouldPrepareNetworking  *result
		shouldStoreDomainIface   bool
		shouldStoreDHCPConfig    bool
		expectedDomainIface      *api.Interface
		expectedDHCPConfig       *cache.DHCPConfig
		expectedPodInterface     *cache.PodCacheInterface
		should                   types.GomegaMatcher
	}
	DescribeTable("PlugPhase1 call", func(c plugPhase1Case) {
		podnic, err := newPodNICWithMocks(c.vmi)
		Expect(err).ToNot(HaveOccurred())

		if c.storedDomainIface != nil {
			podnic.cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Write(podnic.vmiSpecIface.Name, c.storedDomainIface)
		}

		mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return(c.ipv4Addr, c.ipv6Addr, nil).AnyTimes()
		mockNetwork.EXPECT().IsIpv4Primary().Return(c.isIPv4Primary, nil).AnyTimes()

		if c.shouldDiscoverNetworking != nil {
			mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(primaryPodInterfaceName).Times(1).Return(c.shouldDiscoverNetworking.err)
		} else {
			mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(primaryPodInterfaceName).Times(0)
		}

		if c.shouldPrepareNetworking != nil {
			mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Times(1).Return(c.shouldPrepareNetworking.err)
		} else {
			mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Times(0)
		}

		if c.expectedDomainIface != nil {
			mockPodNetworkConfigurator.EXPECT().GenerateDomainIfaceSpec().Return(*c.expectedDomainIface).Times(1)
		} else {
			mockPodNetworkConfigurator.EXPECT().GenerateDomainIfaceSpec().Times(0)
		}

		if c.expectedDHCPConfig != nil {
			mockPodNetworkConfigurator.EXPECT().GenerateDHCPConfig().Return(c.expectedDHCPConfig).Times(1)
		} else {
			mockPodNetworkConfigurator.EXPECT().GenerateDHCPConfig().Times(0)
		}

		Expect(podnic.PlugPhase1()).Should(c.should)

		if c.expectedPodInterface != nil {
			obtainedPodInterface, err := cacheFactory.CacheForVMI(c.vmi).Read(podnic.vmiSpecIface.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedPodInterface).To(Equal(c.expectedPodInterface))
		} else {
			_, err := cacheFactory.CacheForVMI(c.vmi).Read(podnic.vmiSpecIface.Name)
			Expect(err).To(MatchError(os.ErrNotExist))
		}

		if c.shouldStoreDomainIface && c.expectedDomainIface != nil {
			obtainedDomainIface, err := cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Read(podnic.vmiSpecIface.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedDomainIface).To(Equal(c.expectedDomainIface))
		} else {
			_, err := cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Read(podnic.vmiSpecIface.Name)
			Expect(err).To(MatchError(os.ErrNotExist))
		}

		if c.shouldStoreDHCPConfig && c.expectedDHCPConfig != nil {
			obtainedDHCPConfig, err := cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Read(c.expectedDHCPConfig.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(obtainedDHCPConfig).To(Equal(c.expectedDHCPConfig))
		} else {
			_, err := cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Read(podnic.vmiSpecIface.Name)
			Expect(err).To(MatchError(os.ErrNotExist))
		}
	},
		Entry("should success when interface binding is SRIOV", plugPhase1Case{
			vmi:    newVMISRIOVInterface("testnamespace", "testVmName"),
			should: Succeed(),
		}),
		Entry("should success and be a noop if libvirt domain interface is already generate for bridge binding", plugPhase1Case{
			vmi:               newVMIBridgeInterface("testnamespace", "testVmName"),
			storedDomainIface: &api.Interface{},
			should:            Succeed(),
		}),
		Entry("should success and be a noop if libvirt domain interface is already generate for masquerade binding", plugPhase1Case{
			vmi:               newVMIMasqueradeInterface("testnamespace", "testVmName"),
			storedDomainIface: &api.Interface{},
			should:            Succeed(),
		}),
		Entry("should success and be a noop if libvirt domain interface is already generate for macvtap binding", plugPhase1Case{
			vmi:               newVMIMacvtapInterface("testnamespace", "testVmName", "default"),
			storedDomainIface: &api.Interface{},
			should:            Succeed(),
		}),
		Entry("should success and be a noop if libvirt domain interface is already generate for slirp binding", plugPhase1Case{
			vmi:               newVMISlirpInterface("testnamespace", "testVmName"),
			storedDomainIface: &api.Interface{},
			should:            Succeed(),
		}),
		Entry("should success and write only the pod interface IPs for slirp binding", plugPhase1Case{
			vmi:                 newVMISlirpInterface("testnamespace", "testVmName"),
			ipv4Addr:            "1.2.3.4",
			ipv6Addr:            "::1234:5678",
			isIPv4Primary:       true,
			expectedDomainIface: nil,
			expectedDHCPConfig:  nil,
			expectedPodInterface: &cache.PodCacheInterface{
				Iface:  v1.DefaultSlirpNetworkInterface(),
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			should: Succeed(),
		}),
		Entry("should success and write DHCP config, libvirt interface and pod interface for bridge binding", plugPhase1Case{
			vmi:                 newVMIBridgeInterface("testnamespace", "testVmName"),
			ipv4Addr:            "1.2.3.4",
			ipv6Addr:            "::1234:5678",
			isIPv4Primary:       true,
			expectedDomainIface: &api.Interface{},
			expectedDHCPConfig:  &cache.DHCPConfig{Name: primaryPodInterfaceName},
			expectedPodInterface: &cache.PodCacheInterface{
				Iface:  v1.DefaultBridgeNetworkInterface(),
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withSucces(),
			should:                   Succeed(),
		}),
		Entry("should propagate error if network discovering fails for bridge binding", plugPhase1Case{
			vmi:           newVMIBridgeInterface("testnamespace", "testVmName"),
			ipv4Addr:      "1.2.3.4",
			ipv6Addr:      "::1234:5678",
			isIPv4Primary: true,
			expectedPodInterface: &cache.PodCacheInterface{
				Iface:  v1.DefaultBridgeNetworkInterface(),
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldDiscoverNetworking: withError(),
			should:                   MatchError(withError().err),
		}),
		Entry("should return CriticalNetworkError if network preparation fails for bridge binding", plugPhase1Case{
			vmi:                 newVMIBridgeInterface("testnamespace", "testVmName"),
			ipv4Addr:            "1.2.3.4",
			ipv6Addr:            "::1234:5678",
			isIPv4Primary:       true,
			expectedDomainIface: &api.Interface{},
			expectedDHCPConfig:  &cache.DHCPConfig{Name: primaryPodInterfaceName},
			expectedPodInterface: &cache.PodCacheInterface{
				Iface:  v1.DefaultBridgeNetworkInterface(),
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   false,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withError(),
			should:                   MatchError(errors.CreateCriticalNetworkError(withError().err)),
		}),
	)
	type plugPhase2Case struct {
		vmi                   *v1.VirtualMachineInstance
		domain                *api.Domain
		dhcpConfig            *cache.DHCPConfig
		domainIface           *api.Interface
		shouldStartDHCPServer *result
		should                types.GomegaMatcher
	}
	DescribeTable("PlugPhase2 is call", func(c plugPhase2Case) {
		api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(c.domain)
		podnic, err := newPodNICWithMocks(c.vmi)
		Expect(err).ToNot(HaveOccurred())
		if c.dhcpConfig != nil {
			podnic.cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Write(podnic.podInterfaceName, c.dhcpConfig)
		}
		if c.domainIface != nil {
			podnic.cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Write(podnic.vmiSpecIface.Name, c.domainIface)
		}
		if c.shouldStartDHCPServer != nil {
			mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(c.shouldStartDHCPServer.err)
		}
		switch c.should.(type) {
		case *matchers.PanicMatcher:
			Expect(func() { podnic.PlugPhase2(c.domain) }).To(c.should)
		default:
			Expect(podnic.PlugPhase2(c.domain)).Should(c.should)
		}
	},
		Entry("should success when the interface binding is SRIOV", plugPhase2Case{
			vmi:    newVMISRIOVInterface("testnamespace", "testVmName"),
			domain: &api.Domain{},
			should: Succeed(),
		}),
		Entry("should success and decorate libvirt interface when the interface binding is slirp", plugPhase2Case{
			vmi:         newVMISlirpInterface("testnamespace", "testVmName"),
			domain:      NewDomainWithSlirpInterface(),
			domainIface: &api.Interface{},
			should:      Succeed(),
		}),
		Entry("should sucess and decorate libvirt interface when the interface binding is macvtap", plugPhase2Case{
			vmi:         newVMIMacvtapInterface("testnamespace", "testVmName", "default"),
			domain:      NewDomainWithMacvtapInterface("default"),
			domainIface: &api.Interface{},
			should:      Succeed(),
		}),
		Entry("should success, decorate libvirt interface and start DHCP server when interface binding is bridge", plugPhase2Case{
			vmi:                   newVMIBridgeInterface("testnamespace", "testVmName"),
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withSucces(),
			should:                Succeed(),
		}),
		Entry("should panic when starting DHCP server fails on bridge interface binding", plugPhase2Case{
			vmi:                   newVMIBridgeInterface("testnamespace", "testVmName"),
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withError(),
			should:                Panic(),
		}),
		Entry("should success, decorate libvirt interface and start DHCP server when interface binding is masquerade", plugPhase2Case{
			vmi:                   newVMIMasqueradeInterface("testnamespace", "testVmName"),
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withSucces(),
			should:                Succeed(),
		}),
		Entry("should panic when starting DHCP server fails on masquerade interface binding", plugPhase2Case{
			vmi:                   newVMIMasqueradeInterface("testnamespace", "testVmName"),
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withError(),
			should:                Panic(),
		}),
	)
})
