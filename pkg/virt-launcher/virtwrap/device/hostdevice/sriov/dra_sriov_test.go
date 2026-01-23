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

package sriov_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	netsriov "kubevirt.io/kubevirt/pkg/network/deviceinfo"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
)

var _ = Describe("DRA SR-IOV HostDevice", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "dra-sriov-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Context("CreateDRASRIOVHostDevicesFromPath", func() {
		It("returns error when file does not exist", func() {
			nonExistentPath := filepath.Join(tmpDir, "non-existent.json")

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(nonExistentPath)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
			Expect(devices).To(BeNil())
		})

		It("returns error when file is empty", func() {
			emptyFile := filepath.Join(tmpDir, "empty.json")
			Expect(os.WriteFile(emptyFile, []byte(""), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(emptyFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
			Expect(devices).To(BeNil())
		})

		It("returns error when file contains invalid JSON", func() {
			invalidJSONFile := filepath.Join(tmpDir, "invalid.json")
			Expect(os.WriteFile(invalidJSONFile, []byte("not valid json"), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(invalidJSONFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse"))
			Expect(devices).To(BeNil())
		})

		It("returns error when results map is empty", func() {
			emptyMapFile := filepath.Join(tmpDir, "empty-map.json")
			Expect(os.WriteFile(emptyMapFile, []byte(`{}`), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(emptyMapFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no entries"))
			Expect(devices).To(BeNil())
		})

		It("returns error when PCI addresses array is empty", func() {
			emptyArrayFile := filepath.Join(tmpDir, "empty-array.json")
			Expect(os.WriteFile(emptyArrayFile, []byte(`{"claim/request": []}`), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(emptyArrayFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no valid PCI addresses"))
			Expect(devices).To(BeNil())
		})

		It("returns error when PCI address is invalid", func() {
			invalidPCIFile := filepath.Join(tmpDir, "invalid-pci.json")
			Expect(os.WriteFile(invalidPCIFile, []byte(`{"claim/request": ["invalid-pci"]}`), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(invalidPCIFile)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse PCI address"))
			Expect(devices).To(BeNil())
		})

		It("creates host device with valid PCI address", func() {
			validFile := filepath.Join(tmpDir, "valid.json")
			Expect(os.WriteFile(validFile, []byte(`{"virt-launcher-test-claim/vf": ["0000:81:01.0"]}`), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(validFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(devices).To(HaveLen(1))

			expectedPCIAddress := api.Address{
				Type:     api.AddressPCI,
				Domain:   "0x0000",
				Bus:      "0x81",
				Slot:     "0x01",
				Function: "0x0",
			}
			expectedDevice := api.HostDevice{
				Alias:   api.NewUserDefinedAlias(netsriov.SRIOVAliasPrefix + "dra-net0"),
				Source:  api.HostDeviceSource{Address: &expectedPCIAddress},
				Type:    api.HostDevicePCI,
				Managed: "no",
			}
			Expect(devices[0]).To(Equal(expectedDevice))
		})

		It("creates multiple host devices from multiple claims", func() {
			validFile := filepath.Join(tmpDir, "multi.json")
			content := `{"claim1/vf": ["0000:81:01.0"], "claim2/vf": ["0000:3b:00.1"]}`
			Expect(os.WriteFile(validFile, []byte(content), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(validFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(devices).To(HaveLen(2))

			// Verify both devices have correct type and managed settings
			for _, dev := range devices {
				Expect(dev.Type).To(Equal(api.HostDevicePCI))
				Expect(dev.Managed).To(Equal("no"))
			}
		})

		It("creates multiple host devices from single claim with multiple PCI addresses", func() {
			validFile := filepath.Join(tmpDir, "multi-pci.json")
			content := `{"claim/vf": ["0000:81:01.0", "0000:81:02.0"]}`
			Expect(os.WriteFile(validFile, []byte(content), 0644)).To(Succeed())

			devices, err := sriov.CreateDRASRIOVHostDevicesFromPath(validFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(devices).To(HaveLen(2))
		})
	})
})
