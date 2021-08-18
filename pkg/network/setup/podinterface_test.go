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
	"io/ioutil"
	"net"
	"os"
	"runtime"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Pod Network", func() {
	var mockNetwork *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var fakeMac net.HardwareAddr
	var fakeAddr netlink.Addr
	var testNic *cache.DHCPConfig
	var tmpDir string
	var pid int
	var mtu int
	var cacheFactory cache.InterfaceCacheFactory

	BeforeEach(func() {
		dutils.MockDefaultOwnershipManager()
		var err error
		tmpDir, err = ioutil.TempDir("/tmp", "interface-cache")
		Expect(err).ToNot(HaveOccurred())
		cacheFactory = cache.NewInterfaceCacheFactoryWithBasePath(tmpDir)

		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = netdriver.NewMockNetworkHandler(ctrl)
		testMac := "12:34:56:78:9A:BC"
		mtu = 1410
		address := &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}
		gw := net.IPv4(10, 35, 0, 1)
		fakeMac, _ = net.ParseMAC(testMac)
		fakeAddr = netlink.Addr{IPNet: address}
		pid = os.Getpid()

		testNic = &cache.DHCPConfig{Name: primaryPodInterfaceName,
			IP:                fakeAddr,
			MAC:               fakeMac,
			Mtu:               uint16(mtu),
			AdvertisingIPAddr: gw,
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("on successful setup", func() {
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
				Expect(netdriver.FilterPodNetworkRoutes(staticRouteList, testNic)).To(Equal(expectedRouteList))
			})
		})
		Context("Slirp Plug", func() {
			It("Should create an interface in the qemu command line and remove it from the interfaces", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				podnic, err := newPhase2PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, domain)
				Expect(err).ToNot(HaveOccurred())
				driver := podnic.newLibvirtSpecGenerator(domain)
				Expect(driver.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
			It("Should append MAC address to qemu arguments if set", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				vmi.Spec.Domain.Devices.Interfaces[0].MacAddress = "de-ad-00-00-be-af"
				podnic, err := newPhase2PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, domain)
				Expect(err).ToNot(HaveOccurred())
				driver := podnic.newLibvirtSpecGenerator(domain)
				Expect(driver.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(0))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default,mac=de-ad-00-00-be-af"}))
			})
			It("Should create an interface in the qemu command line, remove it from the interfaces and leave the other interfaces inplace", func() {
				domain := NewDomainWithSlirpInterface()
				vmi := newVMISlirpInterface("testnamespace", "testVmName")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

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
				podnic, err := newPhase1PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
				Expect(err).ToNot(HaveOccurred())
				driver := podnic.newLibvirtSpecGenerator(domain)
				Expect(driver.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
				Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
				Expect(domain.Spec.QEMUCmd.QEMUArg[0]).To(Equal(api.Arg{Value: "-device"}))
				Expect(domain.Spec.QEMUCmd.QEMUArg[1]).To(Equal(api.Arg{Value: "e1000,netdev=default,id=default"}))
			})
		})
		Context("Macvtap plug", func() {
			var podnic *podNIC

			BeforeEach(func() {
				vmi := newVMIMacvtapInterface("testnamespace", "testVmName", "default")
				var err error
				podnic, err = newPhase1PodNIC(vmi, &vmi.Spec.Networks[0], mockNetwork, cacheFactory, &pid)
				Expect(err).ToNot(HaveOccurred())
				macvtapInterface := &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: podnic.podInterfaceName, MTU: mtu, HardwareAddr: fakeMac}}
				mockNetwork.EXPECT().LinkByName(podnic.podInterfaceName).Return(macvtapInterface, nil)
			})

			It("Should pass a non-privileged macvtap interface to qemu", func() {
				domain := NewDomainWithMacvtapInterface("default")

				api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)
				specGenerator := podnic.newLibvirtSpecGenerator(domain)

				Expect(specGenerator.generate()).To(Succeed())

				Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1), "should have a single interface")
				Expect(domain.Spec.Devices.Interfaces[0].Target).To(Equal(&api.InterfaceTarget{Device: podnic.podInterfaceName, Managed: "no"}), "should have an unmanaged interface")
				Expect(domain.Spec.Devices.Interfaces[0].MAC).To(Equal(&api.MAC{MAC: fakeMac.String()}), "should have the expected MAC address")
				Expect(domain.Spec.Devices.Interfaces[0].MTU).To(Equal(&api.MTU{Size: "1410"}), "should have the expected MTU")
			})
		})
	})
})
