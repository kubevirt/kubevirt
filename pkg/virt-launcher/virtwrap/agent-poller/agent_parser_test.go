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

			response := stripAgentResponse(jsonInput)
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

		It("should parse Devices", func() {
			const jsonInput = `{
                "return":[
                    {
                        "driver-date": 1721001600000000000,
                        "driver-name": "Red Hat VirtIO Ethernet Adapter",
                        "driver-version": "100.95.104.26200",
                        "id": {
                            "device-id": 4161,
                            "vendor-id": 6900,
                            "type": "pci"
                        }
                    },
                    {
                        "driver-date": 1577836800000000000,
                        "driver-name": "Red Hat VirtIO SCSI controller",
                        "driver-version": "100.80.104.17800",
                        "id": {
                            "device-id": 4164,
                            "vendor-id": 6900,
                            "type": "pci"
                        }
                    }
                ]
            }`

			expectedDevices := []api.GuestDevice{
				{
					DriverName:    "Red Hat VirtIO Ethernet Adapter",
					DriverVersion: "100.95.104.26200",
					DriverDate:    1721001600000000000,
					DeviceID:      4161,
				},
				{
					DriverName:    "Red Hat VirtIO SCSI controller",
					DriverVersion: "100.80.104.17800",
					DriverDate:    1577836800000000000,
					DeviceID:      4164,
				},
			}

			devices, err := parseDevices(jsonInput)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(Equal(expectedDevices))
		})

		It("should skip Devices missing driver-date", func() {
			const jsonInput = `{
                "return":[
                    {
                        "driver-name": "Device without driver date",
                        "driver-version": "1.0.0.0",
                        "id": {
                            "device-id": 4161,
                            "vendor-id": 6900,
                            "type": "pci"
                        }
                    },
                    {
                        "driver-date": 1577836800000000000,
                        "driver-name": "Red Hat VirtIO SCSI controller",
                        "driver-version": "100.80.104.17800",
                        "id": {
                            "device-id": 4164,
                            "vendor-id": 6900,
                            "type": "pci"
                        }
                    }
                ]
            }`

			// The device without driver-date is skipped, so only the complete
			// device is returned.
			expectedDevices := []api.GuestDevice{
				{
					DriverName:    "Red Hat VirtIO SCSI controller",
					DriverVersion: "100.80.104.17800",
					DriverDate:    1577836800000000000,
					DeviceID:      4164,
				},
			}

			devices, err := parseDevices(jsonInput)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(Equal(expectedDevices))
		})

		It("should skip Devices missing id", func() {
			const jsonInput = `{
                "return":[
                    {
                        "driver-date": 1721001600000000000,
                        "driver-name": "Device without id",
                        "driver-version": "2.0.0.0"
                    },
                    {
                        "driver-date": 1577836800000000000,
                        "driver-name": "Red Hat VirtIO SCSI controller",
                        "driver-version": "100.80.104.17800",
                        "id": {
                            "device-id": 4164,
                            "vendor-id": 6900,
                            "type": "pci"
                        }
                    }
                ]
            }`

			// The device without id is skipped, so only the complete device is
			// returned.
			expectedDevices := []api.GuestDevice{
				{
					DriverName:    "Red Hat VirtIO SCSI controller",
					DriverVersion: "100.80.104.17800",
					DriverDate:    1577836800000000000,
					DeviceID:      4164,
				},
			}

			devices, err := parseDevices(jsonInput)
			Expect(err).ToNot(HaveOccurred())
			Expect(devices).To(Equal(expectedDevices))
		})
	})
})
