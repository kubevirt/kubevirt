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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Network", func() {
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

	Context("on successful setup", func() {
		It("should define a new VIF", func() {

			Handler = mockNetwork
			domain := &api.Domain{}

			api.SetObjectDefaults_Domain(domain)

			mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
			mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
			mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
			mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
			mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(nil)
			mockNetwork.EXPECT().LinkSetDown(dummy).Return(nil)
			mockNetwork.EXPECT().ChangeMacAddr(podInterface).Return(updateFakeMac, nil)
			mockNetwork.EXPECT().LinkSetUp(dummy).Return(nil)
			mockNetwork.EXPECT().LinkAdd(bridgeTest).Return(nil)
			mockNetwork.EXPECT().LinkByName(api.DefaultBridgeName).Return(bridgeTest, nil)
			mockNetwork.EXPECT().LinkSetUp(bridgeTest).Return(nil)
			mockNetwork.EXPECT().ParseAddr(bridgeFakeIP).Return(bridgeAddr, nil)
			mockNetwork.EXPECT().AddrAdd(bridgeTest, bridgeAddr).Return(nil)
			mockNetwork.EXPECT().StartDHCP(testNic, bridgeAddr)

			err := SetupPodNetwork(domain)
			Expect(err).To(BeNil())
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
			xmlStr, err := xml.Marshal(domain.Spec.Devices.Interfaces)
			Expect(string(xmlStr)).To(Equal(string(interfaceXml)))
			Expect(err).To(BeNil())

			// Calling SetupPodNetwork a second time should result in no
			// mockNetwork function calls and interface should be identical
			err = SetupPodNetwork(domain)

			Expect(err).To(BeNil())
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))
			xmlStr, err = xml.Marshal(domain.Spec.Devices.Interfaces)
			Expect(string(xmlStr)).To(Equal(string(interfaceXml)))
			Expect(err).To(BeNil())

		})
		It("should panic if pod networking fails to setup", func() {
			testNetworkPanic := func() {
				Handler = mockNetwork
				domain := &api.Domain{}

				api.SetObjectDefaults_Domain(domain)

				mockNetwork.EXPECT().LinkByName(podInterface).Return(dummy, nil)
				mockNetwork.EXPECT().AddrList(dummy, netlink.FAMILY_V4).Return(addrList, nil)
				mockNetwork.EXPECT().RouteList(dummy, netlink.FAMILY_V4).Return(routeList, nil)
				mockNetwork.EXPECT().GetMacDetails(podInterface).Return(fakeMac, nil)
				mockNetwork.EXPECT().AddrDel(dummy, &fakeAddr).Return(errors.New("device is busy"))

				SetupPodNetwork(domain)
			}
			Expect(testNetworkPanic).To(Panic())
		})
	})
})
var _ = Describe("Network Functions", func() {
	var tmpDir string
	BeforeEach(func() {
		tmpDir, _ = ioutil.TempDir("", "networktest")
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})
	Context("get dns config", func() {
		It("shoud get nameservers", func() {
			testHandler := &NetworkUtilsHandler{}
			expected := []byte{192, 168, 0, 1, 8, 8, 8, 8}
			configFile := tmpDir + "/resolv.conf"
			content := []byte("# Generated by NetworkManager\nnameserver 192.168.0.1\nnameserver 8.8.8.8")
			ioutil.WriteFile(configFile, content, 0644)
			servers := testHandler.ReadDNSConfig(configFile)
			Expect(servers).To(Equal(expected))
		})
	})
})
