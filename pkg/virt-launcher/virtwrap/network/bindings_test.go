package network

import (
	"fmt"
	"net"
	"runtime"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("BindingMechanism", func() {
	testDiscoverAndPrepare := func(driver BindMechanism, podInterfaceName string) {
		err := driver.discoverPodNetworkInterface(podInterfaceName)
		Expect(err).ToNot(HaveOccurred())

		Expect(driver.preparePodNetworkInterface()).To(Succeed())
		Expect(driver.decorateConfig(driver.generateDomainIfaceSpec())).To(Succeed())
	}
	testDiscoverAndPrepareWithoutIfaceName := func(driver BindMechanism) {
		testDiscoverAndPrepare(driver, "")
	}
	newVMI := func(namespace, name string) *kubevirtv1.VirtualMachineInstance {
		vmi := kubevirtv1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []kubevirtv1.Network{*v1.DefaultPodNetwork()}
		return vmi
	}
	Context("when slirp binding mechanism is selected", func() {
		var (
			domain *api.Domain
			driver SlirpBindMechanism
			vmi    *kubevirtv1.VirtualMachineInstance
		)
		NewDomainWithSlirpInterface := func() *api.Domain {
			domain := &api.Domain{}
			domain.Spec.Devices.Interfaces = []api.Interface{{
				Model: &api.Model{
					Type: "e1000",
				},
				Type:  "user",
				Alias: api.NewUserDefinedAlias("default"),
			},
			}

			// Create network interface
			if domain.Spec.QEMUCmd == nil {
				domain.Spec.QEMUCmd = &api.Commandline{}
			}

			if domain.Spec.QEMUCmd.QEMUArg == nil {
				domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
			}

			return domain
		}
		newVMISlirpInterface := func(namespace string, name string) *kubevirtv1.VirtualMachineInstance {
			vmi := newVMI(namespace, name)
			vmi.Spec.Domain.Devices.Interfaces = []kubevirtv1.Interface{*kubevirtv1.DefaultSlirpNetworkInterface()}
			kubevirtv1.SetObjectDefaults_VirtualMachineInstance(vmi)
			return vmi
		}
		BeforeEach(func() {
			domain = NewDomainWithSlirpInterface()
			vmi = newVMISlirpInterface("testnamespace", "testVmName")
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			driver = SlirpBindMechanism{iface: &vmi.Spec.Domain.Devices.Interfaces[0], domain: domain}
		})
		It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
			testDiscoverAndPrepareWithoutIfaceName(&driver)
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
		})
		It("Should append MAC address to qemu arguments if set", func() {
			vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
			testDiscoverAndPrepareWithoutIfaceName(&driver)
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default,mac=de-ad-00-00-be-af"}))
		})
		It("Should create an interface in the qemu command line, remove it from the interfaces and leave the other interfaces inplace", func() {
			domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, api.Interface{
				Model: &api.Model{
					Type: "virtio",
				},
				Type: "bridge",
				Source: api.InterfaceSource{
					Bridge: api.DefaultBridgeName,
				},
				Alias: api.NewUserDefinedAlias("default"),
			})
			testDiscoverAndPrepareWithoutIfaceName(&driver)
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
			Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
		})
	})
	Context("when macvtap binding mechanism is selected", func() {
		var (
			domain           *api.Domain
			driver           MacvtapBindMechanism
			vmi              *kubevirtv1.VirtualMachineInstance
			ifaceName        string
			macvtapInterface *netlink.GenericLink
			mtu              int
			mac              net.HardwareAddr
			cacheFactory     cache.InterfaceCacheFactory
			mockNetwork      *netdriver.MockNetworkHandler
		)
		NewDomainWithMacvtapInterface := func(macvtapName string) *api.Domain {
			domain := &api.Domain{}
			domain.Spec.Devices.Interfaces = []api.Interface{{
				Alias: api.NewUserDefinedAlias(macvtapName),
				Model: &api.Model{
					Type: "virtio",
				},
				Type: "ethernet",
			}}
			return domain
		}
		newVMIMacvtapInterface := func(namespace string, vmiName string, ifaceName string) *kubevirtv1.VirtualMachineInstance {
			vmi := newVMI(namespace, vmiName)
			vmi.Spec.Domain.Devices.Interfaces = []kubevirtv1.Interface{*kubevirtv1.DefaultMacvtapNetworkInterface(ifaceName)}
			kubevirtv1.SetObjectDefaults_VirtualMachineInstance(vmi)
			return vmi
		}
		BeforeEach(func() {
			cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()
			ctrl := gomock.NewController(GinkgoT())
			mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
			mtu = 1410
			var err error
			mac, err = net.ParseMAC("12:34:56:78:9A:BC")
			Expect(err).ToNot(HaveOccurred())
			ifaceName = "macvtap0"
			domain = NewDomainWithMacvtapInterface(ifaceName)
			vmi = newVMIMacvtapInterface("testnamespace", "default", ifaceName)
			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
			driver = MacvtapBindMechanism{
				vmi:          vmi,
				iface:        &vmi.Spec.Domain.Devices.Interfaces[0],
				domain:       domain,
				mac:          &mac,
				cacheFactory: cacheFactory,
				handler:      mockNetwork,
			}
			macvtapInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu, HardwareAddr: mac}}
		})
		It("Should pass a non-privileged macvtap interface to qemu", func() {
			mockNetwork.EXPECT().LinkByName(ifaceName).Return(macvtapInterface, nil)
			testDiscoverAndPrepare(&driver, ifaceName)
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(domain.Spec.Devices.Interfaces[0].Target).To(Equal(&api.InterfaceTarget{Device: ifaceName, Managed: "no"}), "should have an unmanaged interface")
			Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(&api.MAC{MAC: mac.String()}), "should have the expected MAC address")
			Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(&api.MTU{Size: fmt.Sprintf("%d", mtu)}), "should have the expected MTU")

		})
	})

})
