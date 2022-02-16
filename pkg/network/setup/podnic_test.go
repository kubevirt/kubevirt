package network

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"

	v1 "kubevirt.io/api/core/v1"
	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/dhcp"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("podNIC", func() {
	const (
		masqueradeCidr     = "10.0.2.0/30"
		masqueradeIpv6Cidr = "fd10:0:2::0/120"
	)
	var (
		mockNetwork                *netdriver.MockNetworkHandler
		cacheFactory               cache.InterfaceCacheFactory
		mockPodNetworkConfigurator *infraconfigurators.MockPodNetworkInfraConfigurator
		mockDHCPConfigurator       *dhcp.MockConfigurator
		ctrl                       *gomock.Controller
		tmpDir                     string
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
		dutils.MockDefaultOwnershipManager()
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())
		cacheFactory = cache.NewInterfaceCacheFactoryWithBasePath(tmpDir)

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		mockPodNetworkConfigurator = infraconfigurators.NewMockPodNetworkInfraConfigurator(ctrl)
		mockDHCPConfigurator = dhcp.NewMockConfigurator(ctrl)
	})
	AfterEach(func() {
		os.RemoveAll(tmpDir)
		ctrl.Finish()
	})
	When("reading networking configuration succeed", func() {
		var (
			podnic *podNIC
			vmi    *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return("1.2.3.4", "169.254.0.0", nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil)
		})

		BeforeEach(func() {
			mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(primaryPodInterfaceName)
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
				podnic.domainGenerator = &fakeLibvirtSpecGenerator{
					shouldGenerateFail: false,
				}
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
				podnic.domainGenerator = &fakeLibvirtSpecGenerator{
					shouldGenerateFail: false,
				}
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

	Context("state retrieval function", func() {
		var (
			podnic *podNIC
		)
		BeforeEach(func() {
			vmi := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
			var err error
			podnic, err = newPhase1PodNICWithMocks(vmi)
			Expect(err).ToNot(HaveOccurred())
		})
		When("there is no pod interface cache stored", func() {
			It("should should return PodIfaceNetworkPreparationPending", func() {
				Expect(podnic.state()).To(Equal(cache.PodIfaceNetworkPreparationPending))
			})
		})
		DescribeTable("when the pod interface cache is stored", func(storedState cache.PodIfaceState) {
			Expect(podnic.setState(storedState)).To(Succeed())
			Expect(podnic.state()).To(Equal(storedState))
		},
			Entry("when PodIfaceNetworkPreparationStarted is stored should return it", cache.PodIfaceNetworkPreparationStarted),
			Entry("when PodIfaceNetworkPreparationPending is stored should return it", cache.PodIfaceNetworkPreparationPending),
			Entry("when PodIfaceNetworkPreparationFinished is stored should return it", cache.PodIfaceNetworkPreparationFinished),
		)
	})
	Context("state transition", func() {
		When("state is PodIfaceNetworkPreparationPending", func() {
			var (
				podnic *podNIC
			)
			BeforeEach(func() {
				var err error
				vmi := newVMIBridgeInterface("testnamespace", "testVmName")
				podnic, err = newPhase1PodNICWithMocks(vmi)
				Expect(err).ToNot(HaveOccurred())
				pid := 1
				podnic.launcherPID = &pid
				podnic.setState(cache.PodIfaceNetworkPreparationPending)
			})

			BeforeEach(func() {
				mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return("1.2.3.4", "169.254.0.0", nil)
				mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil)
			})

			BeforeEach(func() {
				mockPodNetworkConfigurator.EXPECT().DiscoverPodNetworkInterface(podnic.podInterfaceName)
				mockPodNetworkConfigurator.EXPECT().GenerateNonRecoverableDomainIfaceSpec().Return(&api.Interface{})
				mockPodNetworkConfigurator.EXPECT().GenerateNonRecoverableDHCPConfig().Return(&cache.DHCPConfig{})
			})
			Context("and prepare networking is fine at phase1 calling", func() {
				BeforeEach(func() {
					mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface()
					Expect(podnic.PlugPhase1()).To(Succeed())
				})
				It("should transition to PodIfaceNetworkPreparationFinished state", func() {
					Expect(podnic.state()).To(Equal(cache.PodIfaceNetworkPreparationFinished))
				})
			})
			Context("and prepare networking fails at phase1 calling", func() {
				BeforeEach(func() {
					mockPodNetworkConfigurator.EXPECT().PreparePodNetworkInterface().Return(fmt.Errorf("podnic_test: forced PreparePodNetworkInterface failure"))
					Expect(podnic.PlugPhase1()).ToNot(Succeed())
				})
				It("should set state to PodIfaceNetworkPreparationStarted", func() {
					Expect(podnic.state()).To(Equal(cache.PodIfaceNetworkPreparationStarted))
				})
			})
		})
		When("state is PodIfaceNetworkPreparationStarted", func() {
			var podnic *podNIC
			BeforeEach(func() {
				var err error
				vmi := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
				podnic, err = newPhase1PodNICWithMocks(vmi)
				Expect(err).ToNot(HaveOccurred())
				podnic.setState(cache.PodIfaceNetworkPreparationStarted)
			})
			Context("and phase1 is called", func() {
				It("should fail with CriticalNetworkError", func() {
					err := podnic.PlugPhase1()
					Expect(err).To(HaveOccurred(), "SetupPhase1 should return an error")

					_, ok := err.(*neterrors.CriticalNetworkError)
					Expect(ok).To(BeTrue(), "SetupPhase1 should return an error of type CriticalNetworkError")
				})
			})
		})
		When("state is PodIfaceNetworkPreparationFinished", func() {
			var podnic *podNIC
			BeforeEach(func() {
				var err error
				vmi := newVMIMasqueradeInterface("testnamespace", "testVmName", masqueradeCidr, masqueradeIpv6Cidr)
				podnic, err = newPhase1PodNICWithMocks(vmi)
				Expect(err).ToNot(HaveOccurred())
				podnic.setState(cache.PodIfaceNetworkPreparationFinished)
			})
			Context("and phase1 is called", func() {
				It("should successfully return without calling infra configurator", func() {
					Expect(podnic.PlugPhase1()).To(Succeed())
				})
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
