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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Qemu agent poller", func() {
	Context("recieving a reply from the agent", func() {
		JSONInput := `{
            "return": [
                {
                    "name":"lo",
                    "ip-addresses": [
                        {
                            "ip-address-type": "ipv4",
                            "ip-address": "127.0.0.1",
                            "prefix": 8
                        },
                        {
                            "ip-address-type": "ipv6",
                            "ip-address": "::1",
                            "prefix": 128
                        }
                    ],
                    "hardware-address": "00:00:00:00:00:00"
                },
                {
                    "name":"eth0",
                    "ip-addresses": [
                        {
                            "ip-address-type": "ipv4",
                            "ip-address": "10.244.0.81",
                            "prefix": 24
                        },
                        {
                            "ip-address-type": "ipv6",
                            "ip-address": "fe80::858:aff:fef4:51",
                            "prefix": 64
                        }
                    ],
                    "hardware-address": "0a:58:0a:f4:00:51"
                },
                {
                    "name":"eth1",
                    "ip-addresses": [
                        {
                            "ip-address-type": "ipv6",
                            "ip-address": "fe80::ff:feb0:1766",
                            "prefix": 64
                        }
                    ],
                    "hardware-address": "02:00:00:b0:17:66"
                },
                {
                    "name": "eth5",
                    "ip-addresses": [
                        {
                            "ip-address-type": "ipv4",
                            "ip-address": "1.2.3.4",
                            "prefix": 24
                        },
                        {
                            "ip-address-type": "ipv6",
                            "ip-address": "fe80::ff:1111:2222",
                            "prefix":64
                        }
                    ],
                    "hardware-address": "02:00:00:22:11:11"
                }
            ]
        }`

		It("should not parse network interface data into a list of interfaces", func() {
			malformedJSONInput := `{
                "return": [
                    {
                        "name":"lo",
                        "ip-addresses": [
                            {
                                "ip-address-type": "ipv4",
                                "ip-address": "127.0.0.1",
                                "prefix": 8
                            },

                                "ip-address-type": "ipv6",
                                "ip-address": "::1",
                                "prefix": 128
                            }
                        ],
                        "hardware-address": "00:00:00:00:00:00"
                    }
                ]
            }`
			_, err := parseInterfaces(malformedJSONInput)
			Expect(err).To(HaveOccurred(), "should not parse network interfaces")
		})

		It("should parse it into a list of interfaces", func() {
			// eth5 only present in agent data
			interfaceStatuses, err := parseInterfaces(JSONInput)
			Expect(err).ToNot(HaveOccurred(), "should parse network interfaces")

			expectedStatuses := []api.InterfaceStatus{}
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Mac:           "0a:58:0a:f4:00:51",
					Ip:            "10.244.0.81",
					IPs:           []string{"10.244.0.81", "fe80::858:aff:fef4:51"},
					InterfaceName: "eth0",
				})
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Mac:           "02:00:00:b0:17:66",
					Ip:            "fe80::ff:feb0:1766",
					IPs:           []string{"fe80::ff:feb0:1766"},
					InterfaceName: "eth1",
				})
			expectedStatuses = append(expectedStatuses,
				api.InterfaceStatus{
					Mac:           "02:00:00:22:11:11",
					Ip:            "1.2.3.4",
					IPs:           []string{"1.2.3.4", "fe80::ff:1111:2222"},
					InterfaceName: "eth5",
				})
			Expect(interfaceStatuses).To(Equal(expectedStatuses))
		})

		It("should parse Guest OS Info", func() {
			JSONInput := `{
                "return": {
                    "name": "TestGuestOSName",
                    "kernel-release": "1.1.0-Generic",
                    "version": "1.0.0",
                    "pretty-name": "TestGuestOSName 1.0.0",
                    "version-id": "1.0.0",
                    "kernel-version": "1.1.0",
                    "machine": "x86_64",
                    "id": "testguestos"
                }
            }`

			guestOSInfoStatus, err := parseGuestOSInfo(JSONInput)
			Expect(err).ToNot(HaveOccurred(), "Should parse the info")

			expectedGuestOSInfo := api.GuestOSInfo{
				Name:          "TestGuestOSName",
				KernelRelease: "1.1.0-Generic",
				Version:       "1.0.0",
				PrettyName:    "TestGuestOSName 1.0.0",
				VersionId:     "1.0.0",
				KernelVersion: "1.1.0",
				Machine:       "x86_64",
				Id:            "testguestos",
			}
			Expect(guestOSInfoStatus).To(Equal(expectedGuestOSInfo))
		})

		It("should not parse Guest OS Info", func() {
			malformedJSONInput := `{
                "return": {{
                    "name": "TestGuestOSName",
                    "kernel-release": "1.1.0-Generic",
                    "version": "1.0.0"
                    "pretty-name": "TestGuestOSName 1.0.0",
                    "version-id": "1.0.0",
                    "kernel-version": "1.1.0",
                    "machine": "x86_64",
                    "id": "testguestos"
                }
            }`

			_, err := parseGuestOSInfo(malformedJSONInput)
			Expect(err).To(HaveOccurred(), "Should not parse the info")
		})

		It("should parse FSFreezeStatus", func() {
			jsonInput := `{"return":"frozen"}`
			expectedFSFreezeStatus := api.FSFreeze{Status: "frozen"}
			Expect(ParseFSFreezeStatus(jsonInput)).To(Equal(expectedFSFreezeStatus))
		})

		It("should not parse FSFreezeStatus", func() {
			malformedJSONInput := `{"return": {{frozen}`

			_, err := ParseFSFreezeStatus(malformedJSONInput)
			Expect(err).To(HaveOccurred(), "FSFreezeStatus should not be parsed")

			malformedJSONInput = `{"return": frozen}`

			_, err = ParseFSFreezeStatus(malformedJSONInput)
			Expect(err).To(HaveOccurred(), "FSFreezeStatus should not be parsed")
		})

		It("should parse Hostname", func() {
			jsonInput := `{
                "return":{
                    "host-name":"TestHost"
                }
            }`

			Expect(parseHostname(jsonInput)).To(Equal("TestHost"))
		})

		It("should parse Agent", func() {
			jsonInput := `{
                "return":{
                    "version":"4.1"
                }
            }`

			expectedAgent := AgentInfo{Version: "4.1"}
			Expect(parseAgent(jsonInput)).To(Equal(expectedAgent))
		})

		It("should strip Agent response", func() {
			jsonInput := `{"return":{"version":"4.1"}}`

			response := stripAgentResponse(jsonInput)
			expectedResponse := `{"version":"4.1"}`

			Expect(response).To(Equal(expectedResponse))
		})

		It("should parse Timezone", func() {
			jsonInput := `{
                "return":{
                    "zone":"Prague",
                    "offset":2
                }
            }`

			expectedTimezone := api.Timezone{
				Zone:   "Prague",
				Offset: 2,
			}
			Expect(parseTimezone(jsonInput)).To(Equal(expectedTimezone))
		})

		It("should parse Filesystem", func() {
			jsonInput := `{
                "return":[
                    {
                        "name":"main",
                        "mountpoint":"/",
                        "type":"ext",
                        "total-bytes":99999,
                        "used-bytes":33333,
                        "disk":[
                            {
                                "serial":"testserial-1234",
                                "bus-type":"scsi"
                            }
                        ]
                    }
                ]
            }`

			expectedFilesystem := []api.Filesystem{
				{
					Name:       "main",
					Mountpoint: "/",
					Type:       "ext",
					TotalBytes: 99999,
					UsedBytes:  33333,
					Disk: []api.FSDisk{
						{
							Serial:  "testserial-1234",
							BusType: "scsi",
						},
					},
				},
			}
			Expect(parseFilesystem(jsonInput)).To(Equal(expectedFilesystem))
		})

		It("should parse Users", func() {
			jsonInput := `{
                "return":[
                    {
                        "user":"bob",
                        "domain":"bobs",
                        "login-time":99999
                    }
                ]
            }`

			expectedUsers := []api.User{
				{
					Name:      "bob",
					Domain:    "bobs",
					LoginTime: 99999,
				},
			}
			Expect(parseUsers(jsonInput)).To(Equal(expectedUsers))
		})
	})
})
