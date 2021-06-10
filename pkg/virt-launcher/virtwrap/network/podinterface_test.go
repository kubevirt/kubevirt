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
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/cache/fake"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var (
		mockNetwork         *netdriver.MockNetworkHandler
		ctrl                *gomock.Controller
		primaryPodInterface *netlink.GenericLink
		addrList            []netlink.Addr
		routeList           []netlink.Route
		routeAddr           netlink.Route
		fakeMac             net.HardwareAddr
		fakeAddr            netlink.Addr
		updateFakeMac       net.HardwareAddr
		bridgeTest          *netlink.Bridge
		bridgeAddr          *netlink.Addr
		tmpDir              string
		pid                 int
		tapDeviceName       string
		queueNumber         uint32
		mtu                 int
		cacheFactory        cache.InterfaceCacheFactory
		libvirtUser         string
		newPodNIC           = func(vmi *v1.VirtualMachineInstance) podNIC {
			return podNIC{
				cacheFactory: cacheFactory,
				handler:      mockNetwork,
				vmi:          vmi,
			}
		}
		createDefaultPodNIC = func(vmi *v1.VirtualMachineInstance) podNIC {
			podnic := newPodNIC(vmi)
			podnic.iface = &vmi.Spec.Domain.Devices.Interfaces[0]
			podnic.network = &vmi.Spec.Networks[0]
			podnic.podInterfaceName = primaryPodInterfaceName
			return podnic
		}
		newVMI = func(namespace, name string) *v1.VirtualMachineInstance {
			vmi := v1.NewMinimalVMIWithNS(namespace, name)
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			return vmi
		}
		newVMIBridgeInterface = func(namespace string, name string) *v1.VirtualMachineInstance {
			vmi := newVMI(namespace, name)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			return vmi
		}

		newDomainWithBridgeInterface = func() *api.Domain {
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
	)
	BeforeEach(func() {
		cacheFactory = fake.NewFakeInMemoryNetworkCacheFactory()

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		testMac := "12:34:56:78:9A:BC"
		updateTestMac := "AF:B3:1F:78:2A:CA"
		mtu = 1410
		primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: primaryPodInterfaceName, MTU: mtu}}
		address := &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}
		gw := net.IPv4(10, 35, 0, 1)
		fakeMac, _ = net.ParseMAC(testMac)
		updateFakeMac, _ = net.ParseMAC(updateTestMac)
		fakeAddr = netlink.Addr{IPNet: address}
		addrList = []netlink.Addr{fakeAddr}
		routeAddr = netlink.Route{Gw: gw}
		routeList = []netlink.Route{routeAddr}
		pid = os.Getpid()
		tapDeviceName = "tap0"
		libvirtUser = netdriver.LibvirtUserAndGroupId

		// Create a bridge
		bridgeTest = &netlink.Bridge{
			LinkAttrs: netlink.LinkAttrs{
				Name: api.DefaultBridgeName,
			},
		}

		bridgeAddr, _ = netlink.ParseAddr(fmt.Sprintf(bridgeFakeIP, 0))
		tapDeviceName = "tap0"

	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	queueNumber = uint32(0)

	Context("on successful setup", func() {

		It("phase1 should return a CriticalNetworkError if pod networking fails to setup", func() {

			domain := newDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return("", "", nil)
			mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil)
			mockNetwork.EXPECT().LinkSetDown(primaryPodInterface).Return(nil)
			mockNetwork.EXPECT().SetRandomMac(primaryPodInterfaceName).Return(updateFakeMac, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
			mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkSetUp(primaryPodInterface).Return(nil)
			mockNetwork.EXPECT().ParseAddr(fmt.Sprintf(bridgeFakeIP, 0)).Return(bridgeAddr, nil)
			mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
			mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)
			mockNetwork.EXPECT().GetMacDetails(primaryPodInterfaceName).Return(fakeMac, nil)
			mockNetwork.EXPECT().LinkSetMaster(primaryPodInterface, bridgeTest).Return(nil)
			mockNetwork.EXPECT().AddrDel(primaryPodInterface, &fakeAddr).Return(errors.New("device is busy"))
			mockNetwork.EXPECT().CreateTapDevice(tapDeviceName, queueNumber, pid, mtu, libvirtUser).Return(nil)
			mockNetwork.EXPECT().BindTapDeviceToBridge(tapDeviceName, "k6t-eth0").Return(nil)
			mockNetwork.EXPECT().DisableTXOffloadChecksum(bridgeTest.Name).Return(nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)

			podnic := createDefaultPodNIC(vm)
			podnic.launcherPID = &pid
			err := podnic.PlugPhase1()
			Expect(err).To(HaveOccurred(), "SetupPhase1 should return an error")

			_, ok := err.(*neterrors.CriticalNetworkError)
			Expect(ok).To(BeTrue(), "SetupPhase1 should return an error of type CriticalNetworkError")
		})
		It("should return an error if the MTU is out or range", func() {
			primaryPodInterface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Index: 1, MTU: 65536}}
			domain := newDomainWithBridgeInterface()
			vm := newVMIBridgeInterface("testnamespace", "testVmName")

			api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().ReadIPAddressesFromLink(primaryPodInterfaceName).Return("", "", nil)
			mockNetwork.EXPECT().LinkByName(primaryPodInterfaceName).Return(primaryPodInterface, nil).Times(2)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_ALL).Return(addrList, nil)
			mockNetwork.EXPECT().AddrList(primaryPodInterface, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().GetMacDetails(primaryPodInterfaceName).Return(fakeMac, nil)
			mockNetwork.EXPECT().IsIpv4Primary().Return(true, nil).Times(1)
			mockNetwork.EXPECT().RouteList(primaryPodInterface, netlink.FAMILY_V4).Return(routeList, nil)

			podnic := createDefaultPodNIC(vm)
			podnic.launcherPID = &pid
			err := podnic.PlugPhase1()
			Expect(err).To(HaveOccurred())
		})
		It("phase2 should panic if DHCP startup fails", func() {
			testDhcpPanic := func() {
				domain := newDomainWithBridgeInterface()
				vm := newVMIBridgeInterface("testnamespace", "testVmName")
				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				mockNetwork.EXPECT().StartDHCP(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to open file"))
				podnic := createDefaultPodNIC(vm)
				Expect(podnic.PlugPhase2(domain)).To(Succeed())
			}
			Expect(testDhcpPanic).To(Panic())
		})
		Context("SRIOV Plug", func() {
			It("Does not crash", func() {
				// Plug doesn't do anything for sriov so it's enough to pass an empty domain
				domain := &api.Domain{}
				// Same for network
				net := &v1.Network{}

				iface := &v1.Interface{
					Name: "sriov",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						SRIOV: &v1.InterfaceSRIOV{},
					},
				}
				vmi := newVMI("testnamespace", "testVmName")
				podnic := newPodNIC(vmi)
				podnic.iface = iface
				podnic.network = net
				podnic.podInterfaceName = "fakeiface"
				podnic.launcherPID = &pid
				err := podnic.PlugPhase1()
				Expect(err).ToNot(HaveOccurred())

				err = podnic.PlugPhase2(domain)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
