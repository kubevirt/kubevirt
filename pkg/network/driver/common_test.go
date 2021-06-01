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

package driver

import (
	"fmt"

	"github.com/coreos/go-iptables/iptables"
	"github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Common Methods", func() {
	Context("GetAvailableAddrsFromCIDR function", func() {
		It("Should return 2 addresses", func() {
			networkHandler := NetworkUtilsHandler{}
			gw, vm, err := networkHandler.GetHostAndGwAddressesFromCIDR("10.0.0.0/30")
			Expect(err).ToNot(HaveOccurred())
			Expect(gw).To(Equal("10.0.0.1/30"))
			Expect(vm).To(Equal("10.0.0.2/30"))
		})
		It("Should return 2 IPV6 addresses", func() {
			networkHandler := NetworkUtilsHandler{}
			gw, vm, err := networkHandler.GetHostAndGwAddressesFromCIDR("fd10:0:2::/120")
			Expect(err).ToNot(HaveOccurred())
			Expect(gw).To(Equal("fd10:0:2::1/120"))
			Expect(vm).To(Equal("fd10:0:2::2/120"))
		})
		It("Should fail when the subnet is too small", func() {
			networkHandler := NetworkUtilsHandler{}
			_, _, err := networkHandler.GetHostAndGwAddressesFromCIDR("10.0.0.0/31")
			Expect(err).To(HaveOccurred())
		})
		It("Should fail when the IPV6 subnet is too small", func() {
			networkHandler := NetworkUtilsHandler{}
			_, _, err := networkHandler.GetHostAndGwAddressesFromCIDR("fd10:0:2::/127")
			Expect(err).To(HaveOccurred())
		})
	})
	Context("composeNftablesLoad function", func() {
		table.DescribeTable("should compose the correct command",
			func(protocol iptables.Protocol, protocolVersionNum string) {
				cmd := composeNftablesLoad(protocol)
				Expect(cmd.Path).To(HaveSuffix("nft"))
				Expect(cmd.Args).To(Equal([]string{
					"nft",
					"-f",
					fmt.Sprintf("/etc/nftables/ipv%s-nat.nft", protocolVersionNum)}))
			},
			table.Entry("ipv4", iptables.ProtocolIPv4, "4"),
			table.Entry("ipv6", iptables.ProtocolIPv6, "6"),
		)
	})
})
