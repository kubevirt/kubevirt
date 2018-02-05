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
	"net"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Network", func() {
	var mockNetwork *MockNetworkHandler
	var ctrl *gomock.Controller

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockNetwork = NewMockNetworkHandler(ctrl)
	})

	Context("on successful Network setup", func() {
		It("define a new VIF", func() {

			Handler = mockNetwork
			domain := &api.Domain{}
			vm := &v1.VirtualMachine{
				Status: v1.VirtualMachineStatus{},
			}

			api.SetObjectDefaults_Domain(domain)
			testMac := "12:34:56:78:9A:BC"
			updateTestMac := "AF:B3:1F:78:2A:CA"
			dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Index: 1}}
			address := &net.IPNet{IP: net.IPv4(10, 35, 0, 6), Mask: net.CIDRMask(24, 32)}
			gw := net.IPv4(10, 35, 0, 1)
			fakeMac, _ := net.ParseMAC(testMac)
			updateFakeMac, _ := net.ParseMAC(updateTestMac)
			fakeAddr := netlink.Addr{IPNet: address}
			addrList := []netlink.Addr{fakeAddr}
			routeAddr := netlink.Route{Gw: gw}
			routeList := []netlink.Route{routeAddr}
			macvlanTest := &netlink.Macvlan{
				LinkAttrs: netlink.LinkAttrs{
					Name:        macVlanIfaceName,
					ParentIndex: 1,
				},
				Mode: netlink.MACVLAN_MODE_BRIDGE,
			}
			macvlanAddr, _ := netlink.ParseAddr(macVlanFakeIP)
			testNic := &VIF{Name: podInterface,
				IP:      fakeAddr,
				MAC:     fakeMac,
				Gateway: gw}
			var interfaceXml = []byte(`<Interface type="direct" trustGuestRxFilters="yes"><source dev="eth0" mode="bridge"></source><model type="virtio"></model><mac address="12:34:56:78:9a:bc"></mac></Interface>`)

			mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
			mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
			mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
			mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(nil)
			mockNetwork.EXPECT().LinkSetDown(dummy).Return(nil)
			mockNetwork.EXPECT().ChangeMacAddr(podInterface).Return(updateFakeMac, nil)
			mockNetwork.EXPECT().LinkSetUp(dummy).Return(nil)
			mockNetwork.EXPECT().LinkAdd(macvlanTest).Return(nil)
			mockNetwork.EXPECT().LinkByName(macVlanIfaceName).Return(macvlanTest, nil)
			mockNetwork.EXPECT().LinkSetUp(macvlanTest).Return(nil)
			mockNetwork.EXPECT().ParseAddr(macVlanFakeIP).Return(macvlanAddr, nil)
			mockNetwork.EXPECT().AddrAdd(macvlanTest, macvlanAddr).Return(nil)
			mockNetwork.EXPECT().StartDHCP(testNic, macvlanAddr)

			err := SetupPodNetwork(vm, domain)
			Expect(err).To(BeNil())
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
			xml, err := xml.Marshal(domain.Spec.Devices.Interfaces)
			Expect(string(xml)).To(Equal(string(interfaceXml)))
			Expect(err).To(BeNil())
		})
	})

	AfterEach(func() {
		ctrl.Finish()
	})
})
