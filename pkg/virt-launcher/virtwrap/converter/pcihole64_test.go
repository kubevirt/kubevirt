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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// pciHoleTestBARStart is a non-zero placeholder start address: a sysfs
// resource line with a 0 start is treated as an unused BAR slot and skipped.
const pciHoleTestBARStart = uint64(0x100000000)

// 64-bit, prefetchable, memory BAR flags (see PCI64BitPrefetchableBARBytes).
const pciHoleTestBARFlags = uint64(0x0000020c)

var _ = Describe("passthrough PCI 64-bit hole sizing", func() {
	var devicesPath string

	BeforeEach(func() {
		devicesPath = GinkgoT().TempDir()
		previousPath := passthroughPCIDevicesPath
		passthroughPCIDevicesPath = devicesPath
		DeferCleanup(func() {
			passthroughPCIDevicesPath = previousPath
		})
	})

	writeBAR := func(bdf string, barBytes uint64) {
		devicePath := filepath.Join(devicesPath, bdf)
		Expect(os.MkdirAll(devicePath, 0755)).To(Succeed())
		end := pciHoleTestBARStart + barBytes - 1
		resource := fmt.Sprintf("0x%016x 0x%016x 0x%08x\n", pciHoleTestBARStart, end, pciHoleTestBARFlags)
		Expect(os.WriteFile(filepath.Join(devicePath, "resource"), []byte(resource), 0644)).To(Succeed())
	}

	// pciHostDevice builds an api.HostDevice whose Source.Address renders
	// back to bdf via hardware.PCIAddressToString, e.g. "0000:81:00.0".
	pciHostDevice := func(bdf string) api.HostDevice {
		parts := strings.SplitN(bdf, ":", 3)
		slotFunc := strings.SplitN(parts[2], ".", 2)
		return api.HostDevice{
			Type: api.HostDevicePCI,
			Source: api.HostDeviceSource{
				Address: &api.Address{
					Type:     api.AddressPCI,
					Domain:   "0x" + parts[0],
					Bus:      "0x" + parts[1],
					Slot:     "0x" + slotFunc[0],
					Function: "0x" + slotFunc[1],
				},
			},
		}
	}

	It("returns 0 when there are no passthrough devices", func() {
		vmi := libvmi.New()

		Expect(passthroughPCIHole64KiB(vmi)).To(Equal(uint(0)))
	})

	It("returns 0 when the required hole fits in QEMU's default", func() {
		writeBAR("0000:81:00.0", 16<<30) // 16 GiB
		vmi := libvmi.New()
		hostDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(0)))
	})

	It("sizes the hole from a single device's BARs on a host-passthrough CPU", func() {
		writeBAR("0000:81:00.0", 256<<30) // 256 GiB
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough))
		hostDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(512) << 20)) // 512 GiB in KiB
	})

	It("sums BARs across GPU and generic host device lists on a host-passthrough CPU", func() {
		writeBAR("0000:81:00.0", 256<<30)
		writeBAR("0000:82:00.0", 256<<30)
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough))
		gpuDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}
		genericDevices := []api.HostDevice{pciHostDevice("0000:82:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, gpuDevices, genericDevices)).To(Equal(uint(1) << 30)) // 1 TiB in KiB
	})

	It("skips the override on a non-host-passthrough CPU when the hole would exceed the default phys-bits", func() {
		writeBAR("0000:81:00.0", 256<<30)
		writeBAR("0000:82:00.0", 256<<30)
		vmi := libvmi.New(libvmi.WithGuestMemory("64Gi"))
		gpuDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}
		genericDevices := []api.HostDevice{pciHostDevice("0000:82:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, gpuDevices, genericDevices)).To(Equal(uint(0)))
	})

	It("ignores non-PCI (e.g. mdev) host devices", func() {
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough))
		hostDevices := []api.HostDevice{{Type: "mdev"}}

		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(0)))
	})

	It("returns 0 without panicking when a device's BARs cannot be read", func() {
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough))
		hostDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")} // no resource file written

		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(0)))
	})
})
