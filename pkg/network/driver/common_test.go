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
