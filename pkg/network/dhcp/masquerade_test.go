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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
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
		var (
			ifaceName      string
			iface          *netlink.GenericLink
			mtu            int
			vmiSpecNetwork *v1.Network
			vmiSpecIface   *v1.Interface
		)

		const (
			expectedIpv4Gateway = "10.0.2.1/24"
			expectedIpv4        = "10.0.2.2/24"
			expectedIpv6Gateway = "fd10:0:2::1/120"
			expectedIpv6        = "fd10:0:2::2/120"
		)

		generateExpectedConfigIPv6Disabled := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string) cache.DHCPConfig {
			ipv4, _ := netlink.ParseAddr(expectedIpv4)
			ipv4Gateway, _ := netlink.ParseAddr(expectedIpv4Gateway)

			expectedConfig := cache.DHCPConfig{Name: ifaceName,
				IP:                *ipv4,
				Mtu:               uint16(mtu),
				AdvertisingIPAddr: ipv4Gateway.IP.To4(),
				Gateway:           ipv4Gateway.IP.To4(),
			}

			if macString != nil {
				mac, _ := net.ParseMAC(*macString)
				expectedConfig.MAC = mac
			}

			return expectedConfig
		}

		generateExpectedConfigIPv6Enabled := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string) cache.DHCPConfig {
			expectedConfig := generateExpectedConfigIPv6Disabled(vmiSpecNetwork, macString, mtu, ifaceName)
			ipv6, _ := netlink.ParseAddr(expectedIpv6)
			ipv6Gateway, _ := netlink.ParseAddr(expectedIpv6Gateway)

			expectedConfig.IPv6 = *ipv6
			expectedConfig.AdvertisingIPv6Addr = ipv6Gateway.IP.To16()

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

		When("IPv6 is enabled", func() {
			BeforeEach(func() {
				mockHandler.EXPECT().IsIpv6Enabled(ifaceName).Return(true, nil)
			})
			It("Should return the dhcp configuration", func() {
				config, err := generator.Generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(*config).To(Equal(generateExpectedConfigIPv6Enabled(vmiSpecNetwork, nil, mtu, ifaceName)))
			})
		})

		When("IPv6 is disabled", func() {
			BeforeEach(func() {
				mockHandler.EXPECT().IsIpv6Enabled(ifaceName).Return(false, nil)
			})
			It("Should return the dhcp configuration without IPv6", func() {
				config, err := generator.Generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(*config).To(Equal(generateExpectedConfigIPv6Disabled(vmiSpecNetwork, nil, mtu, ifaceName)))
			})
		})

		It("Should return an error if the config discovering fails", func() {
			vmiSpecNetwork.Pod.VMNetworkCIDR = "abc"

			config, err := generator.Generate()
			Expect(err).To(HaveOccurred())
			Expect(config).To(BeNil())
		})
	})
})
