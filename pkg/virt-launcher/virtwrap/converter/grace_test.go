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

package converter

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Grace host device verification", func() {
	var pciDevicesPath string

	BeforeEach(func() {
		pciDevicesPath = filepath.Join(GinkgoT().TempDir(), "sys", "bus", "pci", "devices")
		previousRuntimeInfo := graceRuntimeInfo
		graceRuntimeInfo = sysfsGraceRuntimeInfoProvider{pciDevicesPath: pciDevicesPath}
		DeferCleanup(func() {
			graceRuntimeInfo = previousRuntimeInfo
		})
	})

	It("verifies admitted Grace host devices by assigned PCI identity", func() {
		writePCIIdentity(pciDevicesPath, "0000:81:00.0", "0x10de", "0x2342")
		domainSpec := &api.DomainSpec{Devices: api.Devices{HostDevices: []api.HostDevice{
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		}}}

		verifiedDevices, err := verifyGraceHostDevices(domainSpec, []string{"gpu-gpu0"})

		Expect(err).ToNot(HaveOccurred())
		Expect(verifiedDevices).To(HaveLen(1))
		Expect(verifiedDevices[0].Alias).To(Equal("gpu-gpu0"))
		Expect(verifiedDevices[0].SourceAddress).To(Equal("0000:81:00.0"))
		Expect(verifiedDevices[0].VendorID).To(Equal("10DE"))
		Expect(verifiedDevices[0].DeviceID).To(Equal("2342"))
		Expect(verifiedDevices[0].HostDevice).To(Equal(&domainSpec.Devices.HostDevices[0]))
	})

	It("fails when an admitted Grace alias is not assigned", func() {
		domainSpec := &api.DomainSpec{}

		_, err := verifyGraceHostDevices(domainSpec, []string{"gpu-gpu0"})

		Expect(err).To(MatchError(ContainSubstring("expected hostdev aliases gpu-gpu0")))
	})

	It("fails when an admitted Grace alias is not a PCI hostdev", func() {
		domainSpec := &api.DomainSpec{Devices: api.Devices{HostDevices: []api.HostDevice{
			{Type: api.HostDeviceMDev, Alias: api.NewUserDefinedAlias("gpu-gpu0")},
		}}}

		_, err := verifyGraceHostDevices(domainSpec, []string{"gpu-gpu0"})

		Expect(err).To(MatchError(ContainSubstring("requires PCI hostdev")))
	})

	It("fails when an admitted Grace hostdev has no PCI source address", func() {
		domainSpec := &api.DomainSpec{Devices: api.Devices{HostDevices: []api.HostDevice{
			{Type: api.HostDevicePCI, Alias: api.NewUserDefinedAlias("gpu-gpu0")},
		}}}

		_, err := verifyGraceHostDevices(domainSpec, []string{"gpu-gpu0"})

		Expect(err).To(MatchError(ContainSubstring("requires assigned PCI source address")))
	})

	It("fails when the assigned PCI device is not a supported Grace GPU", func() {
		writePCIIdentity(pciDevicesPath, "0000:81:00.0", "0x10de", "0x20b0")
		domainSpec := &api.DomainSpec{Devices: api.Devices{HostDevices: []api.HostDevice{
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		}}}

		_, err := verifyGraceHostDevices(domainSpec, []string{"gpu-gpu0"})

		Expect(err).To(MatchError(ContainSubstring("to be a supported NVIDIA Grace GPU")))
	})
})

func writePCIIdentity(basePath, bdf, vendorID, deviceID string) {
	devicePath := filepath.Join(basePath, bdf)
	Expect(os.MkdirAll(devicePath, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(devicePath, "vendor"), []byte(vendorID+"\n"), 0644)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(devicePath, "device"), []byte(deviceID+"\n"), 0644)).To(Succeed())
}

func newGraceTestHostDevice(alias, deviceType, domain, bus, slot, function string) api.HostDevice {
	return api.HostDevice{
		Type:  deviceType,
		Alias: api.NewUserDefinedAlias(alias),
		Source: api.HostDeviceSource{Address: &api.Address{
			Domain:   domain,
			Bus:      bus,
			Slot:     slot,
			Function: function,
		}},
	}
}
