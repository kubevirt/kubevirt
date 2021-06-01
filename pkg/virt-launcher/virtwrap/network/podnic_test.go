package network

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/vishvananda/netlink"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
)

var _ = Describe("podNIC", func() {
	var (
		podnic podNIC
		vmi    *kubevirtv1.VirtualMachineInstance
		iface  *kubevirtv1.Interface
	)
	BeforeEach(func() {
		uid := types.UID("test-1234")
		vmi = &kubevirtv1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: uid}}
		address1 := &net.IPNet{IP: net.IPv4(1, 2, 3, 4)}
		address2 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0)}
		fakeAddr1 := netlink.Addr{IPNet: address1}
		fakeAddr2 := netlink.Addr{IPNet: address2}
		addrList := []netlink.Addr{fakeAddr1, fakeAddr2}

		iface = &kubevirtv1.Interface{Name: "default", InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{Masquerade: &kubevirtv1.InterfaceMasquerade{}}}

		ctrl := gomock.NewController(GinkgoT())
		mtu := 1410
		primaryPodInterface := &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: primaryPodInterfaceName, MTU: mtu}}
		cacheFactory := fake.NewFakeInMemoryNetworkCacheFactory()
		mockNetwork := netdriver.NewMockNetworkHandler(ctrl)
		mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return(fakeAddr1.IP.String(), fakeAddr2.IP.String(), nil)
		mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
		mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
		mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)
		podnic = podNIC{
			cacheFactory: cacheFactory,
			handler:      mockNetwork,
			vmi:          vmi,
		}
		podnic.iface = iface
		podnic.podInterfaceName = primaryPodInterfaceName

	})
	Context("when setPodInterfaceCache is called with a configured nic", func() {
		BeforeEach(func() {
			err := podnic.setPodInterfaceCache()
			Expect(err).ToNot(HaveOccurred())
		})
		It("should write interface to cache file", func() {
			podData, err := podnic.cacheFactory.CacheForVMI(vmi).Read(iface.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(podData.PodIP).To(Equal("1.2.3.4"))
		})
	})
})
