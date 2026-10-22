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
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// pciHoleTestBARStart is a non-zero placeholder start address: a sysfs
// resource line with a 0 start is treated as an unused BAR slot and skipped.
const pciHoleTestBARStart = uint64(0x100000000)

// 64-bit, prefetchable, memory BAR flags (see PCI64BitPrefetchableBARBytes).
const pciHoleTestBARFlags = uint64(0x0000020c)

// writeCPUInfo points hardware.CPUInfoPath at a fake /proc/cpuinfo so
// host-passthrough tests do not depend on the real test machine's CPU.
func writeCPUInfo(vendorID string, physBits uint64) {
	cpuinfo := fmt.Sprintf("processor\t: 0\nvendor_id\t: %s\naddress sizes\t: %d bits physical, 48 bits virtual\n", vendorID, physBits)
	path := filepath.Join(GinkgoT().TempDir(), "cpuinfo")
	Expect(os.WriteFile(path, []byte(cpuinfo), 0644)).To(Succeed())
	hardware.CPUInfoPath = path
}

var _ = Describe("passthrough PCI 64-bit hole sizing", func() {
	var devicesPath string

	BeforeEach(func() {
		devicesPath = GinkgoT().TempDir()
		previousPath := passthroughPCIDevicesPath
		passthroughPCIDevicesPath = devicesPath
		DeferCleanup(func() {
			passthroughPCIDevicesPath = previousPath
		})

		previousCPUInfoPath := hardware.CPUInfoPath
		DeferCleanup(func() {
			hardware.CPUInfoPath = previousCPUInfoPath
		})
		// Default to a wide, non-AMD host so host-passthrough tests do not
		// depend on the real test machine's /proc/cpuinfo.
		writeCPUInfo("GenuineIntel", 52)
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

	It("includes SR-IOV host device BARs in the sizing", func() {
		writeBAR("0000:81:00.0", 256<<30)
		writeBAR("0000:83:00.0", 256<<30)
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough))
		gpuDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}
		sriovDevices := []api.HostDevice{pciHostDevice("0000:83:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, gpuDevices, nil, sriovDevices)).To(Equal(uint(1) << 30)) // 1 TiB in KiB
	})

	It("skips the override on a non-host-passthrough CPU when the hole would exceed the default phys-bits", func() {
		writeBAR("0000:81:00.0", 256<<30)
		writeBAR("0000:82:00.0", 256<<30)
		vmi := libvmi.New(libvmi.WithGuestMemory("64Gi"))
		gpuDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}
		genericDevices := []api.HostDevice{pciHostDevice("0000:82:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, gpuDevices, genericDevices)).To(Equal(uint(0)))
	})

	// These two cases share the same phys-bits (42, a 4 TiB ceiling), the
	// same 1 TiB hole (from a 512 GiB BAR) and the same 2100 GiB of guest
	// memory (2096 GiB of it above 4G, which crosses the 1012 GiB start of
	// the AMD HT reserved range). They differ only in host CPU vendor, to
	// isolate the effect of AMD's reserved-range relocation:
	//
	//   Intel (no relocation): holeStart = roundUp(4Gi+2096Gi, 1Ti) = 3Ti,
	//     top = 4Ti, fits exactly under the 4 TiB ceiling.
	//   AMD (relocated):       holeStart = roundUp(1Ti+2096Gi, 1Ti) = 4Ti,
	//     top = 5Ti, exceeds the 4 TiB ceiling.
	//
	// Without modeling the relocation, the AMD case would incorrectly be
	// accepted as fitting.
	It("accounts for the AMD HyperTransport reserved range on a host-passthrough AMD CPU", func() {
		writeCPUInfo("AuthenticAMD", 42)
		writeBAR("0000:81:00.0", 512<<30) // 512 GiB of BARs -> 1 TiB hole
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough), libvmi.WithGuestMemory("2100Gi"))
		hostDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(0)))
	})

	It("does not apply the AMD reserved-range relocation on a host-passthrough Intel CPU", func() {
		writeCPUInfo("GenuineIntel", 42)
		writeBAR("0000:81:00.0", 512<<30) // 512 GiB of BARs -> 1 TiB hole
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough), libvmi.WithGuestMemory("2100Gi"))
		hostDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}

		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(1) << 30)) // 1 TiB in KiB
	})

	It("falls back to the default phys-bits when the host CPU cannot be read", func() {
		hardware.CPUInfoPath = filepath.Join(GinkgoT().TempDir(), "does-not-exist")
		writeBAR("0000:81:00.0", 512<<30) // 512 GiB of BARs -> 1 TiB hole
		vmi := libvmi.New(libvmi.WithCPUModel(v1.CPUModeHostPassthrough), libvmi.WithGuestMemory("64Gi"))
		hostDevices := []api.HostDevice{pciHostDevice("0000:81:00.0")}

		// Falls back to the 40-bit default: holeStart = roundUp(4Gi+60Gi,
		// 1Ti) = 1Ti, top = 2Ti, which exceeds the 1 TiB (40-bit) ceiling.
		Expect(passthroughPCIHole64KiB(vmi, hostDevices)).To(Equal(uint(0)))
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

	It("does not report a fit when converting the hole to bytes would overflow uint64", func() {
		vmi := libvmi.New()

		// pciHoleKiB<<10 overflows a uint64 here; a naive comparison would
		// wrap around to a small value and incorrectly report a fit.
		fits, _ := fitsGuestPhysAddressSpace(vmi, uint64(1)<<60)

		Expect(fits).To(BeFalse())
	})
})
