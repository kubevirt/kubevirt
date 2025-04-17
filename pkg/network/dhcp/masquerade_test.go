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
 * Copyright The KubeVirt Authors.
 *
 */

package dhcp

import (
	"net"

	"github.com/vishvananda/netlink"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

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

	Context("Generate", func() {
		var (
			ifaceName      string
			subdomain      string
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

		generateExpectedConfig := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string, subdomain string) cache.DHCPConfig {
			expectedConfig := cache.DHCPConfig{Name: ifaceName,
				Mtu:       uint16(mtu),
				Subdomain: subdomain,
			}

			if macString != nil {
				mac, _ := net.ParseMAC(*macString)
				expectedConfig.MAC = mac
			}

			return expectedConfig
		}

		generateExpectedConfigOnlyIPv4Enabled := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string, subdomain string) cache.DHCPConfig {
			expectedConfig := generateExpectedConfig(vmiSpecNetwork, macString, mtu, ifaceName, subdomain)
			ipv4, _ := netlink.ParseAddr(expectedIpv4)
			ipv4Gateway, _ := netlink.ParseAddr(expectedIpv4Gateway)

			expectedConfig.IP = *ipv4
			expectedConfig.AdvertisingIPAddr = ipv4Gateway.IP.To4()
			expectedConfig.Gateway = ipv4Gateway.IP.To4()

			return expectedConfig
		}

		generateExpectedConfigOnlyIPv6Enabled := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string, subdomain string) cache.DHCPConfig {
			expectedConfig := generateExpectedConfig(vmiSpecNetwork, macString, mtu, ifaceName, subdomain)
			ipv6, _ := netlink.ParseAddr(expectedIpv6)
			ipv6Gateway, _ := netlink.ParseAddr(expectedIpv6Gateway)

			expectedConfig.IPv6 = *ipv6
			expectedConfig.AdvertisingIPv6Addr = ipv6Gateway.IP.To16()

			return expectedConfig
		}

		generateExpectedConfigOnlyIPv4AndIPv6Enabled := func(vmiSpecNetwork *v1.Network, macString *string, mtu int, ifaceName string, subdomain string) cache.DHCPConfig {
			expectedConfig := generateExpectedConfigOnlyIPv4Enabled(vmiSpecNetwork, macString, mtu, ifaceName, subdomain)
			ipv6ExpectedConfig := generateExpectedConfigOnlyIPv6Enabled(vmiSpecNetwork, macString, mtu, ifaceName, subdomain)

			expectedConfig.IPv6 = ipv6ExpectedConfig.IPv6
			expectedConfig.AdvertisingIPv6Addr = ipv6ExpectedConfig.AdvertisingIPv6Addr

			return expectedConfig
		}

		BeforeEach(func() {
			vmiSpecNetwork = v1.DefaultPodNetwork()
			vmiSpecIface = &v1.Interface{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			ifaceName = "eth0"
			subdomain = "subdomain"

			generator = MasqueradeConfigGenerator{
				handler:          mockHandler,
				vmiSpecIface:     vmiSpecIface,
				vmiSpecNetwork:   vmiSpecNetwork,
				podInterfaceName: ifaceName,
				subdomain:        subdomain,
			}

			mtu = 1410
			iface = &netlink.GenericLink{LinkAttrs: netlink.LinkAttrs{Name: ifaceName, MTU: mtu}}
		})
		BeforeEach(func() {
			mockHandler.EXPECT().LinkByName(ifaceName).Return(iface, nil)
		})

		When("Only Ipv4 is enabled", func() {
			BeforeEach(func() {
				mockHandler.EXPECT().HasIPv4GlobalUnicastAddress(ifaceName).Return(true, nil)
				mockHandler.EXPECT().HasIPv6GlobalUnicastAddress(ifaceName).Return(false, nil)
			})
			It("Should return the dhcp configuration with IPv4 only", func() {
				config, err := generator.Generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(*config).To(Equal(generateExpectedConfigOnlyIPv4Enabled(vmiSpecNetwork, nil, mtu, ifaceName, subdomain)))
			})
		})

		When("Only IPv6 is enabled", func() {
			BeforeEach(func() {
				mockHandler.EXPECT().HasIPv4GlobalUnicastAddress(ifaceName).Return(false, nil)
				mockHandler.EXPECT().HasIPv6GlobalUnicastAddress(ifaceName).Return(true, nil)
			})
			It("Should return the dhcp configuration with IPv6 only", func() {
				config, err := generator.Generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(*config).To(Equal(generateExpectedConfigOnlyIPv6Enabled(vmiSpecNetwork, nil, mtu, ifaceName, subdomain)))
			})
		})

		When("Both Ipv4 and IPv6 are enabled", func() {
			BeforeEach(func() {
				mockHandler.EXPECT().HasIPv4GlobalUnicastAddress(ifaceName).Return(true, nil)
				mockHandler.EXPECT().HasIPv6GlobalUnicastAddress(ifaceName).Return(true, nil)
			})
			It("Should return the dhcp configuration with both IPv4 and IPv6", func() {
				config, err := generator.Generate()
				Expect(err).ToNot(HaveOccurred())
				Expect(*config).To(Equal(generateExpectedConfigOnlyIPv4AndIPv6Enabled(vmiSpecNetwork, nil, mtu, ifaceName, subdomain)))
			})
		})

		When("Config discovering fails", func() {
			BeforeEach(func() {
				mockHandler.EXPECT().HasIPv4GlobalUnicastAddress(ifaceName).Return(true, nil)
			})
			It("Should return an error", func() {
				vmiSpecNetwork.Pod.VMNetworkCIDR = "abc"

				config, err := generator.Generate()
				Expect(err).To(HaveOccurred())
				Expect(config).To(BeNil())
			})
		})
	})
})
