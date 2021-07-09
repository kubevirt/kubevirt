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
	type testData struct {
		mockNetwork                *netdriver.MockNetworkHandler
		cacheFactory               cache.InterfaceCacheFactory
		mockPodNetworkConfigurator *infraconfigurators.MockPodNetworkInfraConfigurator
		ctrl                       *gomock.Controller
		tmpDir                     string
	}
	beforeEach := func(td *testData) {
		td.cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
		td.ctrl = gomock.NewController(GinkgoT())
		td.mockNetwork = netdriver.NewMockNetworkHandler(td.ctrl)
		td.mockPodNetworkConfigurator = infraconfigurators.NewMockPodNetworkInfraConfigurator(td.ctrl)

		var err error
		tmpDir, err := ioutil.TempDir("/tmp", "dhcp")
		Expect(err).ToNot(HaveOccurred())
		td.tmpDir = tmpDir
	}
	afterEach := func(td *testData) {
		os.RemoveAll(td.tmpDir)
		td.ctrl.Finish()
	}
	newPodNICWithMocks := func(vmi *v1.VirtualMachineInstance, ifaceIndex int, td *testData) (*podNIC, error) {
		launcherPID := 1
		podnic, err := newPodNICWithoutInfraConfigurator(vmi, &vmi.Spec.Networks[ifaceIndex], td.mockNetwork, td.cacheFactory, &launcherPID)
		if err != nil {
			return nil, err
		}
		podnic.infraConfigurator = td.mockPodNetworkConfigurator
		if podnic.dhcpConfigurator != nil {
			podnic.dhcpConfigurator = dhcpconfigurator.NewConfiguratorWithDHCPStartedDirectory(td.cacheFactory, getPIDString(podnic.launcherPID), podnic.podInterfaceName, td.mockNetwork, td.tmpDir)
		}
		return podnic, nil
	}
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
		vmis                     []*v1.VirtualMachineInstance
		storedDomainIface        *api.Interface
		ipv4Addr                 string
		ipv6Addr                 string
		isIPv4Primary            bool
		ifaceIndex               int
		shouldDiscoverNetworking *result
		shouldPrepareNetworking  *result
		shouldStoreDomainIface   bool
		shouldStoreDHCPConfig    bool
		expectedPodInterfaceName string
		expectedDomainIface      *api.Interface
		expectedDHCPConfig       *cache.DHCPConfig
		expectedPodInterface     *cache.PodCacheInterface
		should                   types.GomegaMatcher
	}
	DescribeTable("PlugPhase1 call", func(c plugPhase1Case) {

		// Create a closure so "defer" do as expected
		runTestForVMI := func(vmi *v1.VirtualMachineInstance) {
			td := &testData{}
			beforeEach(td)
			defer afterEach(td)
			podnic, err := newPodNICWithMocks(vmi, c.ifaceIndex, td)
			Expect(err).ToNot(HaveOccurred())

			Expect(podnic.podInterfaceName).To(Equal(c.expectedPodInterfaceName))
			if c.storedDomainIface != nil {
				podnic.cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Write(podnic.vmiSpecIface.Name, c.storedDomainIface)
			}

			td.mockNetwork.EXPECT().ReadIPAddressesFromLink(c.expectedPodInterfaceName).Return(c.ipv4Addr, c.ipv6Addr, nil).AnyTimes()
			td.mockNetwork.EXPECT().IsIpv4Primary().Return(c.isIPv4Primary, nil).AnyTimes()

			if c.shouldDiscoverNetworking != nil {
				td.mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(c.expectedPodInterfaceName).Times(1).Return(c.shouldDiscoverNetworking.err)
			} else {
				td.mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(c.expectedPodInterfaceName).Times(0)
			}

			if c.shouldPrepareNetworking != nil {
				td.mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Times(1).Return(c.shouldPrepareNetworking.err)
			} else {
				td.mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Times(0)
			}

			if c.expectedDomainIface != nil {
				td.mockPodNetworkConfigurator.EXPECT().GenerateDomainIfaceSpec().Return(*c.expectedDomainIface).Times(1)
			} else {
				td.mockPodNetworkConfigurator.EXPECT().GenerateDomainIfaceSpec().Times(0)
			}

			if c.expectedDHCPConfig != nil {
				td.mockPodNetworkConfigurator.EXPECT().GenerateDHCPConfig().Return(c.expectedDHCPConfig).Times(1)
			} else {
				td.mockPodNetworkConfigurator.EXPECT().GenerateDHCPConfig().Times(0)
			}

			Expect(podnic.PlugPhase1()).Should(c.should)

			if c.expectedPodInterface != nil {
				// Don't modify the case
				expectedPodInterface := *c.expectedPodInterface
				if expectedPodInterface.Iface == nil {
					expectedPodInterface.Iface = &vmi.Spec.Domain.Devices.Interfaces[c.ifaceIndex]
				}
				obtainedPodInterface, err := td.cacheFactory.CacheForVMI(vmi).Read(podnic.vmiSpecIface.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(*obtainedPodInterface).To(Equal(expectedPodInterface))
			} else {
				_, err := td.cacheFactory.CacheForVMI(vmi).Read(podnic.vmiSpecIface.Name)
				Expect(err).To(MatchError(os.ErrNotExist))
			}

			if c.shouldStoreDomainIface && c.expectedDomainIface != nil {
				obtainedDomainIface, err := td.cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Read(podnic.vmiSpecIface.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(obtainedDomainIface).To(Equal(c.expectedDomainIface))
			} else {
				_, err := td.cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Read(podnic.vmiSpecIface.Name)
				Expect(err).To(MatchError(os.ErrNotExist))
			}

			if c.shouldStoreDHCPConfig && c.expectedDHCPConfig != nil {
				obtainedDHCPConfig, err := td.cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Read(c.expectedDHCPConfig.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(obtainedDHCPConfig).To(Equal(c.expectedDHCPConfig))
			} else {
				_, err := td.cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Read(podnic.vmiSpecIface.Name)
				Expect(err).To(MatchError(os.ErrNotExist))
			}
		}
		for _, vmi := range c.vmis {
			runTestForVMI(vmi)
		}
	},
		Entry("should success when interface binding is SRIOV", plugPhase1Case{
			vmis:                     []*v1.VirtualMachineInstance{newVMISRIOVInterface("testnamespace", "testVmName")},
			expectedPodInterfaceName: primaryPodInterfaceName,
			should:                   Succeed(),
		}),
		Entry("should success and be a noop if libvirt domain interface is already generate for all bindings", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIBridgeInterface("testnamespace", "testVmName"),
				newVMIMasqueradeInterface("testnamespace", "testVmName"),
				newVMISlirpInterface("testnamespace", "testVmName"),
				newVMIMacvtapInterface("testnamespace", "testVmName", "default"),
			},
			expectedPodInterfaceName: primaryPodInterfaceName,
			storedDomainIface:        &api.Interface{},
			should:                   Succeed(),
		}),
		Entry("should success and write only the pod interface IPs for slirp binding", plugPhase1Case{
			vmis:                     []*v1.VirtualMachineInstance{newVMISlirpInterface("testnamespace", "testVmName")},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      nil,
			expectedDHCPConfig:       nil,
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			should: Succeed(),
		}),
		Entry("should success with pure ipv4 cluster and store pod IPs correctly", plugPhase1Case{
			vmis:                     []*v1.VirtualMachineInstance{newVMISlirpInterface("testnamespace", "testVmName")},
			ipv4Addr:                 "1.2.3.4",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      nil,
			expectedDHCPConfig:       nil,
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4"},
			},
			should: Succeed(),
		}),
		Entry("should success with pure ipv6 cluster and store pod IPs correctly", plugPhase1Case{
			vmis:                     []*v1.VirtualMachineInstance{newVMISlirpInterface("testnamespace", "testVmName")},
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            false,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      nil,
			expectedDHCPConfig:       nil,
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "::1234:5678",
				PodIPs: []string{"::1234:5678"},
			},
			should: Succeed(),
		}),
		Entry("should select ipv6 if it's the prefered on the cluster", plugPhase1Case{
			vmis:                     []*v1.VirtualMachineInstance{newVMISlirpInterface("testnamespace", "testVmName")},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            false,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      nil,
			expectedDHCPConfig:       nil,
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "::1234:5678",
				PodIPs: []string{"::1234:5678", "1.2.3.4"},
			},
			should: Succeed(),
		}),

		Entry("should success and write the pod interface IPs and libvirt domain interface for macvtap binding", plugPhase1Case{
			vmis:                     []*v1.VirtualMachineInstance{newVMIMacvtapInterface("testnamespace", "testVmName", "default")},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      &api.Interface{},
			expectedDHCPConfig:       nil,
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDomainIface:   true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withSucces(),
			should:                   Succeed(),
		}),
		Entry("should success and write DHCP config, libvirt interface and pod interface for bridge and masquerade binding", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIBridgeInterface("testnamespace", "testVmName"),
				newVMIMasqueradeInterface("testnamespace", "testVmName"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      &api.Interface{},
			expectedDHCPConfig:       &cache.DHCPConfig{Name: primaryPodInterfaceName},
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withSucces(),
			should:                   Succeed(),
		}),
		Entry("should success and write DHCP config, libvirt interface and pod interface for default default interafce with multus", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIDefaultMultusInterface("testnamespace", "testVmName"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      &api.Interface{},
			expectedDHCPConfig:       &cache.DHCPConfig{Name: primaryPodInterfaceName},
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withSucces(),
			should:                   Succeed(),
		}),
		Entry("should success and write DHCP config, libvirt interface and pod interface for first secondary multus network", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIWithBridgeAndTwoSecondaryMultusInterfaces("testnamespace", "testVmName"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			ifaceIndex:               1,
			expectedPodInterfaceName: "net1",
			expectedDomainIface:      &api.Interface{},
			expectedDHCPConfig:       &cache.DHCPConfig{Name: "net1"},
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withSucces(),
			should:                   Succeed(),
		}),
		Entry("should success and write DHCP config, libvirt interface and pod interface for second secondary multus network", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIWithBridgeAndTwoSecondaryMultusInterfaces("testnamespace", "testVmName"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			ifaceIndex:               2,
			expectedPodInterfaceName: "net2",
			expectedDomainIface:      &api.Interface{},
			expectedDHCPConfig:       &cache.DHCPConfig{Name: "net2"},
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withSucces(),
			should:                   Succeed(),
		}),

		Entry("should propagate error if network discovering fails for bridge, masquerade and macvtap binding", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIBridgeInterface("testnamespace", "testVmName"),
				newVMIMasqueradeInterface("testnamespace", "testVmName"),
				newVMIMacvtapInterface("testnamespace", "testVmName", "default"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldDiscoverNetworking: withError(),
			should:                   MatchError(withError().err),
		}),
		Entry("should return CriticalNetworkError if network preparation fails for bridge and masquerade binding", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIBridgeInterface("testnamespace", "testVmName"),
				newVMIMasqueradeInterface("testnamespace", "testVmName"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      &api.Interface{},
			expectedDHCPConfig:       &cache.DHCPConfig{Name: primaryPodInterfaceName},
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldStoreDomainIface:   false,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withError(),
			should:                   MatchError(errors.CreateCriticalNetworkError(withError().err)),
		}),
		Entry("should return CriticalNetworkError if network preparation fails for macvtap binding", plugPhase1Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMIMacvtapInterface("testnamespace", "testVmName", "default"),
			},
			ipv4Addr:                 "1.2.3.4",
			ipv6Addr:                 "::1234:5678",
			isIPv4Primary:            true,
			expectedPodInterfaceName: primaryPodInterfaceName,
			expectedDomainIface:      &api.Interface{},
			expectedPodInterface: &cache.PodCacheInterface{
				PodIP:  "1.2.3.4",
				PodIPs: []string{"1.2.3.4", "::1234:5678"},
			},
			shouldStoreDHCPConfig:    true,
			shouldDiscoverNetworking: withSucces(),
			shouldPrepareNetworking:  withError(),
			should:                   MatchError(errors.CreateCriticalNetworkError(withError().err)),
		}),
	)
	type plugPhase2Case struct {
		vmis                  []*v1.VirtualMachineInstance
		domain                *api.Domain
		dhcpConfig            *cache.DHCPConfig
		domainIface           *api.Interface
		ifaceIndex            int
		shouldStartDHCPServer *result
		should                types.GomegaMatcher
	}
	DescribeTable("PlugPhase2 is call", func(c plugPhase2Case) {
		runTestForVMI := func(vmi *v1.VirtualMachineInstance) {
			td := &testData{}
			beforeEach(td)
			defer afterEach(td)
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(c.domain)
			podnic, err := newPodNICWithMocks(vmi, c.ifaceIndex, td)
			Expect(err).ToNot(HaveOccurred())
			if c.dhcpConfig != nil {
				podnic.cacheFactory.CacheDHCPConfigForPid(getPIDString(podnic.launcherPID)).Write(podnic.podInterfaceName, c.dhcpConfig)
			}
			if c.domainIface != nil {
				podnic.cacheFactory.CacheDomainInterfaceForPID(getPIDString(podnic.launcherPID)).Write(podnic.vmiSpecIface.Name, c.domainIface)
			}
			if c.shouldStartDHCPServer != nil {
				td.mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(c.shouldStartDHCPServer.err)
			}
			switch c.should.(type) {
			case *matchers.PanicMatcher:
				Expect(func() { podnic.PlugPhase2(c.domain) }).To(c.should)
			default:
				Expect(podnic.PlugPhase2(c.domain)).Should(c.should)
			}
		}
		for _, vmi := range c.vmis {
			runTestForVMI(vmi)
		}
	},
		Entry("should success when the interface binding is SRIOV", plugPhase2Case{
			vmis:   []*v1.VirtualMachineInstance{newVMISRIOVInterface("testnamespace", "testVmName")},
			domain: &api.Domain{},
			should: Succeed(),
		}),
		Entry("should success and just decorate libvirt interface when the interface binding is slirp or macvtap", plugPhase2Case{
			vmis: []*v1.VirtualMachineInstance{
				newVMISlirpInterface("testnamespace", "testVmName"),
				newVMIMacvtapInterface("testnamespace", "testVmName", "default"),
			},
			domain:      NewDomainWithSlirpInterface(),
			domainIface: &api.Interface{},
			should:      Succeed(),
		}),
		Entry("should success, decorate libvirt interface and start DHCP server when interface binding is bridge", plugPhase2Case{
			vmis:                  []*v1.VirtualMachineInstance{newVMIBridgeInterface("testnamespace", "testVmName")},
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withSucces(),
			should:                Succeed(),
		}),
		Entry("should panic when starting DHCP server fails on bridge interface binding", plugPhase2Case{
			vmis:                  []*v1.VirtualMachineInstance{newVMIBridgeInterface("testnamespace", "testVmName")},
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withError(),
			should:                Panic(),
		}),
		Entry("should success, decorate libvirt interface and start DHCP server when interface binding is masquerade", plugPhase2Case{
			vmis:                  []*v1.VirtualMachineInstance{newVMIMasqueradeInterface("testnamespace", "testVmName")},
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withSucces(),
			should:                Succeed(),
		}),
		Entry("should panic when starting DHCP server fails on masquerade interface binding", plugPhase2Case{
			vmis:                  []*v1.VirtualMachineInstance{newVMIMasqueradeInterface("testnamespace", "testVmName")},
			domain:                NewDomainWithBridgeInterface(),
			domainIface:           &api.Interface{},
			dhcpConfig:            &cache.DHCPConfig{Name: "default"},
			shouldStartDHCPServer: withError(),
			should:                Panic(),
		}),
	)
})
