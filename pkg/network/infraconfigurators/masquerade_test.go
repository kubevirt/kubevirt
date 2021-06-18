package infraconfigurators

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

var _ = Describe("Masquerade infrastructure configurator", func() {
	var (
		ctrl    *gomock.Controller
		handler *netdriver.MockNetworkHandler
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		handler = netdriver.NewMockNetworkHandler(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	const (
		bridgeIfaceName = "k6t-eth0"
	)

	newVMIMasqueradeInterface := func(namespace string, name string) *v1.VirtualMachineInstance {
		vmi := v1.NewMinimalVMIWithNS(namespace, name)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		return vmi
	}

	Context("discover link information", func() {
		const (
			ifaceName   = "eth0"
			launcherPID = 1000
		)

		var (
			masqueradeConfigurator *MasqueradePodNetworkConfigurator
			podLink                *netlink.GenericLink
			vmi                    *v1.VirtualMachineInstance
		)

		BeforeEach(func() {
			vmi = newVMIMasqueradeInterface("default", "vm1")
			masqueradeConfigurator = NewMasqueradePodNetworkConfigurator(vmi, &vmi.Spec.Domain.Devices.Interfaces[0], bridgeIfaceName, &vmi.Spec.Networks[0], launcherPID, handler)
		})

		When("the pod link is defined", func() {
			BeforeEach(func() {
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 1000}}
				handler.EXPECT().LinkByName(ifaceName).Return(podLink, nil)
			})

			When("the pod interface has an IPv4 address", func() {
				When("and is missing and IPv6 address", func() {
					BeforeEach(func() {
						handler.EXPECT().IsIpv6Enabled(ifaceName).Return(false, nil)
					})

					It("should succeed discovering the pod link info", func() {
						Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
						Expect(masqueradeConfigurator.podNicLink).To(Equal(podLink))
					})
				})

				When("and we fail to understand if there's an IPv6 configuration", func() {
					BeforeEach(func() {
						handler.EXPECT().IsIpv6Enabled(ifaceName).Return(true, fmt.Errorf("failed to check pod's IPv6 configuration"))
					})

					It("should fail to discover the pod's link information", func() {
						Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
					})
				})
			})

			When("the pod interface has both IPv4 and IPv6 addresses", func() {
				BeforeEach(func() {
					handler.EXPECT().IsIpv6Enabled(ifaceName).Return(true, nil)
				})

				It("should succeed reading the pod link info", func() {
					Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(Succeed())
					Expect(masqueradeConfigurator.podNicLink).To(Equal(podLink))
				})
			})
		})

		When("the pod link information cannot be retrieved", func() {
			BeforeEach(func() {
				handler.EXPECT().LinkByName(ifaceName).Return(nil, fmt.Errorf("cannot get pod link"))
			})

			It("should fail to discover the pod's link information", func() {
				Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
			})
		})

		When("the podnic link has invalid information", func() {
			BeforeEach(func() {
				podLink = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: 100000}}
				handler.EXPECT().LinkByName(ifaceName).Return(podLink, nil)
			})

			It("should fail to discover the pod's link information", func() {
				Expect(masqueradeConfigurator.DiscoverPodNetworkInterface(ifaceName)).To(HaveOccurred())
			})
		})
	})
})
