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

package dhcp

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("DHCP", func() {

	Context("check routes", func() {
		It("verify should form correctly", func() {
			expected := []byte{4, 224, 192, 168, 1, 1, 24, 192, 168, 1, 192, 168, 2, 1}
			gateway := net.IPv4(192, 168, 1, 1)
			routes := []netlink.Route{
				netlink.Route{
					LinkIndex: 3,
					Dst: &net.IPNet{
						IP:   net.IPv4(224, 0, 0, 0),
						Mask: net.CIDRMask(4, 32),
					},
				},
				netlink.Route{
					LinkIndex: 12,
					Dst: &net.IPNet{
						IP:   net.IPv4(192, 168, 1, 0),
						Mask: net.CIDRMask(24, 32),
					},
					Src: nil,
					Gw:  net.IPv4(192, 168, 2, 1),
				},
			}

			dhcpRoutes := FormClasslessRoutes(&routes, gateway)
			Expect(dhcpRoutes).To(Equal(expected))
		})
	})
})
