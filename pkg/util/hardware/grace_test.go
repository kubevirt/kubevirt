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

package hardware

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NVIDIA Grace PCI IDs", func() {
	DescribeTable("detects supported Grace GPU PCI IDs", func(vendorID, deviceID string, expected bool) {
		Expect(IsNVIDIAGraceGPU(vendorID, deviceID)).To(Equal(expected))
	},
		Entry("Grace GPU", "10DE", "2342", true),
		Entry("Grace GPU with sysfs prefixes", "0x10de", "0x2348", true),
		Entry("Grace GPU with mixed case", "10de", "2941", true),
		Entry("Grace GPU with surrounding spaces", " 10de ", " 2342 ", true),
		Entry("empty vendor", "", "2342", false),
		Entry("empty device", "10DE", "", false),
		Entry("short vendor ID", "10D", "2342", false),
		Entry("long vendor ID", "010DE", "2342", false),
		Entry("short device ID", "10DE", "234", false),
		Entry("long device ID", "10DE", "02342", false),
		Entry("non-Grace NVIDIA GPU", "10DE", "2330", false),
		Entry("non-NVIDIA device", "1AF4", "2342", false),
	)

	DescribeTable("detects NVIDIA PCI vendor IDs", func(vendorID string, expected bool) {
		Expect(IsNVIDIAPCIVendor(vendorID)).To(Equal(expected))
	},
		Entry("uppercase vendor ID", "10DE", true),
		Entry("lowercase vendor ID", "10de", true),
		Entry("sysfs-prefixed vendor ID", "0x10DE", true),
		Entry("surrounding spaces", " 10de ", true),
		Entry("non-NVIDIA vendor ID", "1AF4", false),
		Entry("empty vendor ID", "", false),
		Entry("short vendor ID", "10D", false),
		Entry("long vendor ID", "010DE", false),
	)
	DescribeTable("parses PCI vendor selectors", func(selector string, expectedSelector PCIVendorSelector, expected bool) {
		parsedSelector, ok := ParsePCIVendorSelector(selector)
		Expect(ok).To(Equal(expected))
		if expected {
			Expect(parsedSelector).To(Equal(expectedSelector))
		}
	},
		Entry("exact selector", "10de:2342", PCIVendorSelector{VendorID: "10DE", DeviceID: "2342"}, true),
		Entry("sysfs prefixes", "0x10DE:0x2348", PCIVendorSelector{VendorID: "10DE", DeviceID: "2348"}, true),
		Entry("wildcard device selector", "10de:*", PCIVendorSelector{VendorID: "10DE", DeviceID: "*"}, true),
		Entry("missing separator", "10de2342", PCIVendorSelector{}, false),
		Entry("invalid vendor", "10dg:2342", PCIVendorSelector{}, false),
		Entry("invalid device", "10de:234g", PCIVendorSelector{}, false),
		Entry("short vendor", "10d:2342", PCIVendorSelector{}, false),
		Entry("long device", "10de:02342", PCIVendorSelector{}, false),
	)

	DescribeTable("detects Grace and ambiguous NVIDIA PCI vendor selectors", func(selector string, expectedGrace bool, expectedAmbiguous bool) {
		Expect(IsNVIDIAGracePCIVendorSelector(selector)).To(Equal(expectedGrace))
		Expect(IsAmbiguousNVIDIAPCIVendorSelector(selector)).To(Equal(expectedAmbiguous))
	},
		Entry("Grace GPU selector", "10de:2342", true, false),
		Entry("Grace GPU selector with prefixes", "0x10de:0x2941", true, false),
		Entry("NVIDIA wildcard selector", "10de:*", false, true),
		Entry("non-Grace NVIDIA selector", "10de:2330", false, false),
		Entry("non-NVIDIA wildcard selector", "1af4:*", false, false),
		Entry("malformed selector", "10de", false, false),
	)

})
