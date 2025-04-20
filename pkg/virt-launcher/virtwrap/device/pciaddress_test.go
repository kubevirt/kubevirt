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

package device_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

var _ = Describe("PCI Address", func() {

	It("is parsed into a domain PCI Address spec", func() {
		Expect(device.NewPciAddressField("0000:81:11.1")).To(Equal(
			&api.Address{
				Type:     api.AddressPCI,
				Domain:   "0x0000",
				Bus:      "0x81",
				Slot:     "0x11",
				Function: "0x1",
			}))
	})

	It("fails to parse an invalid PCI address", func() {
		address, err := device.NewPciAddressField("0000:81:11:1")
		Expect(err).To(HaveOccurred())
		Expect(address).To(BeNil())
	})
})
