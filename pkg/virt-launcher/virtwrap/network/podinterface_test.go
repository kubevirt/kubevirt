/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"syscall"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var mockNetwork *MockNetworkHandler
	var ctrl *gomock.Controller
	var dummy *netlink.Dummy
	var addrList []netlink.Addr
	var routeList []netlink.Route
	var routeAddr netlink.Route
	var fakeMac net.HardwareAddr
	var fakeAddr netlink.Addr
	var updateFakeMac net.HardwareAddr
	var bridgeTest *netlink.Bridge
	var tuntapTest *netlink.Tuntap
	var bridgeAddr *netlink.Addr
	var testNic *VIF
	var interfaceXml []byte
	var tmpDir string
	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		tmpDir, _ := ioutil.TempDir("", "networktest")
		setInterfaceCacheFile(tmpDir + "/cache.json")

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = NewMockNetworkHandler(ctrl)
		Handler = mockNetwork
		testMac := "12:34:56:78:9A:BC"
		updateTestMac := "AF:B3:1F:78:2A:CA"
		dummy = &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1}}
		address := &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}
		gw := net.IPv4(10, 35, 0, 1)
		fakeMac, _ = net.ParseMAC(testMac)
		updateFakeMac, _ = net.ParseMAC(updateTestMac)
		fakeAddr = netlink.Addr{IPNet: address}
		addrList = []netlink.Addr{fakeAddr}
		routeAddr = netlink.Route{Gw: gw}
		routeList = []netlink.Route{routeAddr}

		// Create a bridge
		bridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: api.DefaultBridgeName,
			},
		}

		// Create a Tap device
		tapName := api.DefaultBridgeName + "-nic"
		tuntapTest = &netlink.Tuntap{
			LinkAttrs: netlink.LinkAttrs{
				Name:  tapName,
				Flags: net.FlagUp,
				MTU:   1500,
			},
			Mode: syscall.IFF_TAP,
		}

		bridgeAddr, _ = netlink.ParseAddr(bridgeFakeIP)
		testNic = &VIF{Name: podInterface,
			IP:      fakeAddr,
			MAC:     fakeMac,
			Gateway: gw}
		interfaceXml = []byte(`<Interface type="bridge"><source bridge="br1"></source><model type="e1000"></model><mac address="12:34:56:78:9a:bc"></mac></Interface>`)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	TestPodInterfaceIPBinding := func(vm *v1.VirtualMachine, domain *api.Domain) {

		mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
		mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
		mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
		mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
		mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().LinkSetDown(dummy).Return(nil)
		mockNetwork.EXPECT().SetRandomMacAddr(podInterface).Return(updateFakeMac, nil)
		mockNetwork.EXPECT().LinkSetUp(dummy).Return(nil)
		mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
		mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
		mockNetwork.EXPECT().ParseAddr(bridgeFakeIP).Return(bridgeAddr, nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(testNic, bridgeAddr)

		err := SetupPodNetwork(vm, domain)
		Expect(err).To(BeNil())
		Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
		xmlStr, err := xml.Marshal(domain.Spec.Devices.Interfaces)
		Expect(string(xmlStr)).To(Equal(string(interfaceXml)))
		Expect(err).To(BeNil())

		// Calling SetupPodNetwork a second time should result in no
		// mockNetwork function calls and interface should be identical
		err = SetupPodNetwork(vm, domain)

		Expect(err).To(BeNil())
		Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
		xmlStr, err = xml.Marshal(domain.Spec.Devices.Interfaces)
		Expect(string(xmlStr)).To(Equal(string(interfaceXml)))
		Expect(err).To(BeNil())
	}

	TestPodInterfacePortsBinding := func(vm *v1.VirtualMachine, domain *api.Domain) {
		bridgeFakeIP = "192.168.0.1/24"
		ip, _ := netlink.ParseAddr("192.168.0.2/24")

		gw := net.IPv4(192, 168, 0, 1)
		routes := &[]netlink.Route{}

		testNic = &VIF{Name: podInterface,
			IP:      fakeAddr,
			MAC:     fakeMac,
			Gateway: gw,
			Routes:  routes,
		}
		testNic.IP = *ip

		// create a request to forward ssh port to the virtual interface
		iface := &v1.Interface{
			Name:        "proxy-nic",
			NetworkName: "red2",
			InterfaceBinindMethod: v1.InterfaceBinindMethod{
				Proxy: &v1.InterfaceProxy{Ports: []apiv1.ContainerPort{
					apiv1.ContainerPort{
						HostPort:      22,
						ContainerPort: 54321,
						Name:          "ssh",
						Protocol:      apiv1.ProtocolTCP,
					},
				},
				},
			},
		}

		// a request to connect a pod network to the virtual machine
		vm.Spec.Domain.Devices.Interfaces = []v1.Interface{*iface}
		red2Net := &v1.Network{
			Name: "red2",
			NetworkSource: v1.NetworkSource{
				Pod: &v1.PodNetwork{},
			},
		}
		vm.Spec.Networks = []v1.Network{*red2Net}

		proxyBridgeAddr, _ := netlink.ParseAddr("192.168.0.1/24")
		mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
		mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
		mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(nil)
		mockNetwork.EXPECT().LinkSetDown(dummy).Return(nil)
		mockNetwork.EXPECT().RandomizeMac().Return(fakeMac, nil)

		// setup tap device
		mockNetwork.EXPECT().LinkAdd(tuntapTest).Return(nil)
		mockNetwork.EXPECT().LinkSetUp(tuntapTest).Return(nil)

		// create a bridge on top of the tap
		mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
		mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
		mockNetwork.EXPECT().ParseAddr("192.168.0.1/24").Return(proxyBridgeAddr, nil)
		mockNetwork.EXPECT().AddrAdd(bridgeTest, proxyBridgeAddr).Return(nil)
		mockNetwork.EXPECT().StartDHCP(testNic, proxyBridgeAddr)

		err := SetupPodNetwork(vm, domain)
		Expect(err).To(BeNil())
		Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
		xmlStr, err := xml.Marshal(domain.Spec.Devices.Interfaces)
		Expect(string(xmlStr)).To(Equal(string(interfaceXml)))
		Expect(err).To(BeNil())

		// Calling SetupPodNetwork a second time should result in no
		// mockNetwork function calls and interface should be identical
		err = SetupPodNetwork(vm, domain)

		Expect(err).To(BeNil())
		Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
		xmlStr, err = xml.Marshal(domain.Spec.Devices.Interfaces)
		Expect(string(xmlStr)).To(Equal(string(interfaceXml)))
		Expect(err).To(BeNil())
	}

	Context("on successful setup", func() {
		table.DescribeTable("should define a new VIF bind to ", func(bind string) {

			domain := &api.Domain{}
			vm := newVM("testnamespace", "testVmName")

			api.SetObjectDefaults_Domain(domain)

			switch bind {
			case "IP":
				TestPodInterfaceIPBinding(vm, domain)
			case "Ports":
				TestPodInterfacePortsBinding(vm, domain)
			}

		},
			table.Entry("pod IP", "IP"),
			table.Entry("pod Ports", "Ports"),
		)
		It("should panic if pod networking fails to setup", func() {
			testNetworkPanic := func() {
				domain := &api.Domain{}
				vm := newVM("testnamespace", "testVmName")

				api.SetObjectDefaults_Domain(domain)

				mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
				mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
				mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
				mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
				mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(errors.New("device is busy"))

				SetupPodNetwork(vm, domain)
			}
			Expect(testNetworkPanic).To(Panic())
		})
		Context("func filterPodNetworkRoutes()", func() {
			defRoute := netlink.Route{
				Gw: net.IPv4(10, 35, 0, 1),
			}
			staticRoute := netlink.Route{
				Dst: &net.IPNet{IP: net.IPv4(10, 45, 0, 10), Mask: net.CIDRMask(32, 32)},
				Gw:  net.IPv4(10, 25, 0, 1),
			}
			gwRoute := netlink.Route{
				Dst: &net.IPNet{IP: net.IPv4(10, 35, 0, 1), Mask: net.CIDRMask(32, 32)},
			}
			nicRoute := netlink.Route{Src: net.IPv4(10, 35, 0, 6)}
			emptyRoute := netlink.Route{}
			staticRouteList := []netlink.Route{defRoute, gwRoute, nicRoute, emptyRoute, staticRoute}

			It("should remove empty routes, and routes matching nic, leaving others intact", func() {
				expectedRouteList := []netlink.Route{defRoute, gwRoute, staticRoute}
				Expect(filterPodNetworkRoutes(staticRouteList, testNic)).To(Equal(expectedRouteList))
			})
		})
	})
})

func newVM(namespace string, name string) *v1.VirtualMachine {
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       v1.VirtualMachineSpec{Domain: v1.NewMinimalDomainSpec()},
	}
	v1.SetObjectDefaults_VirtualMachine(vm)
	return vm
}
