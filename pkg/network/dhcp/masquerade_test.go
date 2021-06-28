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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package dhcp

import (
	"net"

	"github.com/vishvananda/netlink"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/coreos/go-iptables/iptables"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

var _ = Describe("Masquerade DHCP configurator", func() {

	var mockHandler *netdriver.MockNetworkHandler
	var ctrl *gomock.Controller
	var generator MasqueradeConfigGenerator

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockHandler = netdriver.NewMockNetworkHandler(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Generate", func() {
		var ifaceName string
		var iface *netlink.GenericLink
		var mtu int
		var vmiSpecNetwork *v1.Network
		var vmiSpecIface *v1.Interface

		generateExpectedConfig := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string) cache.DHCPConfig {
			ipv4Gateway, ipv4, _ := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(vmiSpecNetwork, iptables.ProtocolIPv4)
			ipv6Gateway, ipv6, _ := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(vmiSpecNetwork, iptables.ProtocolIPv6)
			expectedConfig := cache.DHCPConfig{Name: ifaceName,
				IP:                  *ipv4,
				IPv6:                *ipv6,
				Mtu:                 uint16(mtu),
				AdvertisingIPAddr:   ipv4Gateway.IP.To4(),
				AdvertisingIPv6Addr: ipv6Gateway.IP.To16(),
				Gateway:             ipv4Gateway.IP.To4(),
			}

			if macString != nil {
				mac, _ := net.ParseMAC(*macString)
				expectedConfig.MAC = mac
			}
			return expectedConfig
		}

		BeforeEach(func() {
			vmiSpecNetwork = v1.DefaultPodNetwork()
			vmiSpecIface = &v1.Interface{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			ifaceName = "eth0"

			generator = MasqueradeConfigGenerator{
				handler:          mockHandler,
				vmiSpecIface:     vmiSpecIface,
				vmiSpecNetwork:   vmiSpecNetwork,
				podInterfaceName: ifaceName,
			}

			mtu = 1410
			iface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
		})
		BeforeEach(func() {
			mockHandler.EXPECT().LinkByName(ifaceName).Return(iface, nil)
		})
		It("Should return the dhcp configuration without mac", func() {
			config, err := generator.Generate()
			Expect(err).ToNot(HaveOccurred())
			Expect(*config).To(Equal(generateExpectedConfig(vmiSpecNetwork, nil, mtu, ifaceName)))
		})
		It("Should return the dhcp configuration with mac", func() {
			macString := "de-ad-00-00-be-af"
			vmiSpecIface.MacAddress = macString

			config, err := generator.Generate()
			Expect(err).ToNot(HaveOccurred())
			Expect(*config).To(Equal(generateExpectedConfig(vmiSpecNetwork, &macString, mtu, ifaceName)))
		})
		It("Should return an error if the config discovering fails", func() {
			vmiSpecNetwork.Pod.VMNetworkCIDR = "abc"

			config, err := generator.Generate()
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
		})
	})
})
