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

package agentpoller

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Qemu agent poller", func() {
	Context("recieving a reply from the agent", func() {
		It("should parse it into a list of interfaces", func() {
			agentPoller := AgentPoller{}
			agentPoller.domainData = &DomainData{
				// net2 only present in DomainData
				aliasByMac: map[string]string{"0a:58:0a:f4:00:51": "ovs", "02:00:00:b0:17:66": "net1", "02:11:11:b0:17:66": "net2"},
			}

			// eth5 only present in agent data
			json_input := "{\"return\":[{\"name\":\"lo\",\"ip-addresses\":[{\"ip-address-type\":\"ipv4\",\"ip-address\":\"127.0.0.1\",\"prefix\":8},{\"ip-address-type\":\"ipv6\",\"ip-address\":\"::1\",\"prefix\":128}],\"hardware-address\":\"00:00:00:00:00:00\"},{\"name\":\"eth0\",\"ip-addresses\":[{\"ip-address-type\":\"ipv4\",\"ip-address\":\"10.244.0.81\",\"prefix\":24},{\"ip-address-type\":\"ipv6\",\"ip-address\":\"fe80::858:aff:fef4:51\",\"prefix\":64}],\"hardware-address\":\"0a:58:0a:f4:00:51\"},{\"name\":\"eth1\",\"ip-addresses\":[{\"ip-address-type\":\"ipv6\",\"ip-address\":\"fe80::ff:feb0:1766\",\"prefix\":64}],\"hardware-address\":\"02:00:00:b0:17:66\"}, {\"name\":\"eth5\",\"ip-addresses\":[{\"ip-address-type\":\"ipv4\",\"ip-address\":\"1.2.3.4\",\"prefix\":24},{\"ip-address-type\":\"ipv6\",\"ip-address\":\"fe80::ff:1111:2222\",\"prefix\":64}],\"hardware-address\":\"02:00:00:22:11:11\"}]}"
			interfaceStatuses := agentPoller.GetInterfaceStatuses(json_input)
			expectedStatuses := []api.InterfaceStatus{}
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Name:          "ovs",
					Mac:           "0a:58:0a:f4:00:51",
					Ip:            "10.244.0.81/24",
					IPs:           []string{"10.244.0.81/24", "fe80::858:aff:fef4:51/64"},
					InterfaceName: "eth0",
				})
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Name:          "net1",
					Mac:           "02:00:00:b0:17:66",
					Ip:            "fe80::ff:feb0:1766/64",
					IPs:           []string{"fe80::ff:feb0:1766/64"},
					InterfaceName: "eth1",
				})
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Mac:           "02:00:00:22:11:11",
					Ip:            "1.2.3.4/24",
					IPs:           []string{"1.2.3.4/24", "fe80::ff:1111:2222/64"},
					InterfaceName: "eth5",
				})
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Name: "net2",
					Mac:  "02:11:11:b0:17:66",
				})
			Expect(interfaceStatuses).To(Equal(expectedStatuses))
		})
	})
})
