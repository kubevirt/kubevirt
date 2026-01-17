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

package agentpoller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Qemu agent poller", func() {
	Context("receiving a reply from the agent", func() {
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

			response, err := stripAgentResponse(jsonInput)
			Expect(err).To(BeNil())
			expectedResponse := `{"version":"4.1"}`

			Expect(response).To(Equal(expectedResponse))
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
		It("should fail on malformed filesystem agent reply", func() {
			jsonInput := `{dummy input}`

			_, err := parseFilesystem(jsonInput)

			Expect(err).To(HaveOccurred())
		})
	})
})
