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
	"math"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
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

var _ = Describe("Grace sysfs runtime info", func() {
	It("calculates pcihole64 size from 64-bit prefetchable BAR resources", func() {
		pciDevicesPath := filepath.Join(GinkgoT().TempDir(), "sys", "bus", "pci", "devices")
		devicePath := filepath.Join(pciDevicesPath, "0000:81:00.0")
		Expect(os.MkdirAll(devicePath, 0755)).To(Succeed())
		resource := strings.Join([]string{
			"0x00000000 0x00000000 0x00000000",
			"0x0000000100000000 0x000000013fffffff 0x0000020c",
			"0x0000000200000000 0x000000020fffffff 0x0000020c",
			"0x0000000000002000 0x0000000000002fff 0x00000000",
		}, "\n") + "\n"
		Expect(os.WriteFile(filepath.Join(devicePath, "resource"), []byte(resource), 0644)).To(Succeed())
		provider := sysfsGraceRuntimeInfoProvider{pciDevicesPath: pciDevicesPath}

		size, err := provider.PCIHole64SizeBytes("0000:81:00.0")

		Expect(err).ToNot(HaveOccurred())
		Expect(size).To(Equal(uint64(0x50000000)))
	})

	DescribeTable("uses a Grace pcihole64 device floor",
		func(vendorID, deviceID string) {
			verifiedDevice := verifiedGraceHostDevice{VendorID: vendorID, DeviceID: deviceID}

			Expect(gracePCIHole64SizeBytes(verifiedDevice, 16*1024*1024*1024)).To(Equal(gracePCIHole64FloorBytes))
		},
		Entry("GB200 selector", "10de", "2342"),
		Entry("supported Grace selector", "10de", "2348"),
		Entry("GH200 selector", "10de", "2941"),
	)

	It("keeps BAR-derived pcihole64 sizing when it is larger than the Grace floor", func() {
		barBytes := uint64(2) << 40
		verifiedDevice := verifiedGraceHostDevice{VendorID: "10de", DeviceID: "2342"}

		Expect(gracePCIHole64SizeBytes(verifiedDevice, barBytes)).To(Equal(barBytes))
	})

	It("does not apply the Grace pcihole64 device floor to non-Grace PCI IDs", func() {
		barBytes := uint64(16) * 1024 * 1024 * 1024
		verifiedDevice := verifiedGraceHostDevice{VendorID: "1af4", DeviceID: "1041"}

		Expect(gracePCIHole64SizeBytes(verifiedDevice, barBytes)).To(Equal(barBytes))
	})
})

var _ = Describe("Grace conversion preflight", func() {
	newGraceVMI := func() *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{CPU: &v1.CPU{DedicatedCPUPlacement: true}},
			},
		}
	}

	newGraceContext := func() *convertertypes.ConverterContext {
		return &convertertypes.ConverterContext{
			Architecture:                 archconverter.NewConverter("arm64"),
			GraceIOVirtualizationEnabled: true,
			PCINUMAAwareTopologyEnabled:  true,
			IOMMUFDEnabled:               true,
			GraceHostDeviceAliases:       []string{"gpu-gpu0"},
		}
	}

	It("fails closed when the IOMMUFD file descriptor was not received", func() {
		c := newGraceContext()
		c.IOMMUFDEnabled = false

		err := validateGraceIOVirtualizationConversion(newGraceVMI(), c)

		Expect(err).To(MatchError(ContainSubstring("requires an IOMMUFD file descriptor")))
	})

	It("rejects disabling the 64-bit PCI hole before domain mutation", func() {
		vmi := newGraceVMI()
		vmi.Annotations[v1.DisablePCIHole64] = "true"

		err := validateGraceIOVirtualizationConversion(vmi, newGraceContext())

		Expect(err).To(MatchError(ContainSubstring("requires the 64-bit PCI hole")))
	})

	It("rejects root-complex placement before domain mutation", func() {
		vmi := newGraceVMI()
		vmi.Annotations[v1.PlacePCIDevicesOnRootComplex] = "true"

		err := validateGraceIOVirtualizationConversion(vmi, newGraceContext())

		Expect(err).To(MatchError(ContainSubstring("root-complex placement")))
	})
})

var _ = Describe("Grace domain conversion", func() {
	var fakeRuntime *fakeGraceRuntimeInfoProvider

	BeforeEach(func() {
		fakeRuntime = newFakeGraceRuntimeInfoProvider()
		previousRuntimeInfo := graceRuntimeInfo
		graceRuntimeInfo = fakeRuntime
		DeferCleanup(func() {
			graceRuntimeInfo = previousRuntimeInfo
		})
	})

	It("adds SMMUv3, hostdev ACPI, GI NUMA cells, distances, and pcihole64 for one Grace GPU", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		fakeRuntime.capabilities["0000:81:00.0"] = gracePCICapabilities{SSIDSize: "20", OAS: "48"}
		fakeRuntime.giNodes = uint32Range(2, 9)
		fakeRuntime.addDistances(append([]uint32{0}, fakeRuntime.giNodes...))
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(domainSpec.Devices.HostDevices[0].Driver).To(Equal(&api.HostDevDriver{Iommufd: "yes"}))
		Expect(domainSpec.Devices.HostDevices[0].ACPI).To(Equal(&api.ACPIHostDev{NodeSet: "1-8"}))
		Expect(domainSpec.Devices.HostDevices[0].Address).ToNot(BeNil())
		Expect(domainSpec.Devices.IOMMU).To(HaveLen(1))
		Expect(domainSpec.Devices.IOMMU[0].Model).To(Equal("smmuv3"))
		Expect(domainSpec.Devices.IOMMU[0].Driver.PCIBus).To(Equal(graceExpanderBusIndexes(domainSpec)[0]))
		Expect(domainSpec.Devices.IOMMU[0].Driver.Accel).To(Equal("on"))
		Expect(domainSpec.Devices.IOMMU[0].Driver.ATS).To(Equal("on"))
		Expect(domainSpec.Devices.IOMMU[0].Driver.RIL).To(Equal("off"))
		Expect(domainSpec.Devices.IOMMU[0].Driver.SSIDSize).To(Equal("20"))
		Expect(domainSpec.Devices.IOMMU[0].Driver.OAS).To(Equal("48"))
		Expect(rootPCIController(domainSpec).PCIHole64).To(Equal(&api.PCIHole64{Value: 1073741824, Unit: "KiB"}))
		Expect(domainSpec.CPU.NUMA.Cells).To(HaveLen(9))
		Expect(domainSpec.CPU.NUMA.Cells[1].Memory).ToNot(BeNil())
		Expect(*domainSpec.CPU.NUMA.Cells[1].Memory).To(Equal(uint64(0)))
		Expect(domainSpec.CPU.NUMA.Cells[0].Distances).ToNot(BeNil())
		Expect(domainSpec.CPU.NUMA.Cells[1].Distances).ToNot(BeNil())
		Expect(findGraceSibling(domainSpec.CPU.NUMA.Cells[0].Distances.Siblings, "1").Value).To(Equal(uint64(80)))
		Expect(findGraceSibling(domainSpec.CPU.NUMA.Cells[1].Distances.Siblings, "0").Value).To(Equal(uint64(80)))
	})

	It("creates an explicit PCIe root controller before placing Grace PCI devices", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		fakeRuntime.giNodes = uint32Range(2, 9)
		fakeRuntime.addDistances(append([]uint32{0}, fakeRuntime.giNodes...))
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)
		domainSpec.Devices.Controllers = nil

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(domainSpec.Devices.Controllers[0]).To(Equal(api.Controller{
			Type:  api.ControllerTypePCI,
			Index: "0",
			Model: api.ControllerModelPCIeRoot,
			PCIHole64: &api.PCIHole64{
				Value: 1073741824,
				Unit:  "KiB",
			},
		}))
		Expect(countGraceControllers(domainSpec, api.ControllerModelPCIeRoot)).To(Equal(1))
		Expect(countGraceControllers(domainSpec, api.ControllerModelPCIeExpanderBus)).To(Equal(1))
	})

	It("uses one pcie-expander-bus and SMMUv3 IOMMU per Grace GPU", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		fakeRuntime.addGraceGPU("0000:82:00.0", 0, 16*1024*1024*1024)
		fakeRuntime.giNodes = uint32Range(2, 17)
		fakeRuntime.addDistances(append([]uint32{0}, fakeRuntime.giNodes...))
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
			newGraceTestHostDevice("gpu-gpu1", api.HostDevicePCI, "0x0000", "0x82", "0x00", "0x0"),
		)

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0", "gpu-gpu1"}, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(domainSpec.Devices.IOMMU).To(HaveLen(2))
		Expect(countGraceControllers(domainSpec, api.ControllerModelPCIeExpanderBus)).To(Equal(2))
		Expect([]string{domainSpec.Devices.IOMMU[0].Driver.PCIBus, domainSpec.Devices.IOMMU[1].Driver.PCIBus}).To(ConsistOf(graceExpanderBusIndexes(domainSpec)))
		Expect(domainSpec.Devices.HostDevices[0].ACPI.NodeSet).To(Equal("1-8"))
		Expect(domainSpec.Devices.HostDevices[1].ACPI.NodeSet).To(Equal("9-16"))
		Expect(rootPCIController(domainSpec).PCIHole64).To(Equal(&api.PCIHole64{Value: 2147483648, Unit: "KiB"}))
	})

	It("sets 4 TiB pcihole64 for four Grace GPUs", func() {
		for _, bdf := range []string{"0000:81:00.0", "0000:82:00.0", "0000:83:00.0", "0000:84:00.0"} {
			fakeRuntime.addGraceGPU(bdf, 0, 16*1024*1024*1024)
		}
		fakeRuntime.giNodes = uint32Range(2, 33)
		fakeRuntime.addDistances(append([]uint32{0}, fakeRuntime.giNodes...))
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
			newGraceTestHostDevice("gpu-gpu1", api.HostDevicePCI, "0x0000", "0x82", "0x00", "0x0"),
			newGraceTestHostDevice("gpu-gpu2", api.HostDevicePCI, "0x0000", "0x83", "0x00", "0x0"),
			newGraceTestHostDevice("gpu-gpu3", api.HostDevicePCI, "0x0000", "0x84", "0x00", "0x0"),
		)

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0", "gpu-gpu1", "gpu-gpu2", "gpu-gpu3"}, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(domainSpec.Devices.IOMMU).To(HaveLen(4))
		Expect(rootPCIController(domainSpec).PCIHole64).To(Equal(&api.PCIHole64{Value: 4294967296, Unit: "KiB"}))
	})

	It("fails when the IOMMUFD file descriptor is not available", func() {
		domainSpec := newGraceConversionDomain()

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, false)

		Expect(err).To(MatchError(ContainSubstring("requires an IOMMUFD file descriptor")))
	})

	It("fails when SMMUv3 is not available", func() {
		fakeRuntime.smmuv3 = false
		domainSpec := newGraceConversionDomain()

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("requires SMMUv3")))
	})

	It("fails when there are not enough host Generic Initiator NUMA nodes", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		fakeRuntime.giNodes = uint32Range(2, 4)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("requires 8 host Generic Initiator NUMA nodes")))
	})

	It("fails when guest NUMA cells are missing", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)
		domainSpec.CPU.NUMA = nil

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("requires guest NUMA cells")))
	})

	It("fails when NUMATune memnodes are missing", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)
		domainSpec.NUMATune.MemNodes = nil

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("requires NUMATune memnodes")))
	})

	It("fails when NUMATune memnodes map one host NUMA node to multiple guest cells", func() {
		memoryKiB := uint64(1024)
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)
		domainSpec.CPU.NUMA.Cells = append(domainSpec.CPU.NUMA.Cells, api.NUMACell{ID: "1", CPUs: "2-3", Memory: &memoryKiB, Unit: "KiB"})
		domainSpec.NUMATune.MemNodes = append(domainSpec.NUMATune.MemNodes, api.MemNode{CellID: 1, Mode: "strict", NodeSet: "0"})

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("maps to multiple guest NUMA cells")))
	})

	It("fails when a Grace GPU has no 64-bit prefetchable BARs", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 0)
		fakeRuntime.giNodes = uint32Range(2, 9)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("has no 64-bit prefetchable PCI BARs")))
	})

	It("fails when Grace GPU pcihole64 sizing overflows", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, math.MaxUint64)
		fakeRuntime.addGraceGPU("0000:82:00.0", 0, 1)
		fakeRuntime.giNodes = uint32Range(2, 17)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
			newGraceTestHostDevice("gpu-gpu1", api.HostDevicePCI, "0x0000", "0x82", "0x00", "0x0"),
		)

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0", "gpu-gpu1"}, true)

		Expect(err).To(MatchError(ContainSubstring("pcihole64 size overflows")))
	})

	It("detects Grace pcihole64 sizing overflow before applying the controller value", func() {
		Expect(calculateGracePCIHole64KiB(math.MaxUint64, math.MaxUint64)).To(Equal(uint64(0)))
	})

	It("fails when Grace pcihole64 sizing exceeds the maximum aperture", func() {
		domainSpec := newGraceConversionDomain()

		err := applyGracePCIHole64(domainSpec, graceMaxPCIHole64KiB*1024)

		Expect(err).To(MatchError(ContainSubstring("invalid Grace pcihole64 size")))
	})

	It("fails when Grace pcihole64 conflicts with a disabled root-controller aperture", func() {
		domainSpec := newGraceConversionDomain()
		rootPCIController(domainSpec).PCIHole64 = &api.PCIHole64{Value: 0, Unit: "KiB"}

		err := applyGracePCIHole64(domainSpec, 16*1024*1024*1024)

		Expect(err).To(MatchError(ContainSubstring("conflicts with disabled pcihole64")))
	})

	It("creates a PCIe root controller when applying Grace pcihole64 to an implicit root complex", func() {
		domainSpec := newGraceConversionDomain()
		domainSpec.Devices.Controllers = nil

		err := applyGracePCIHole64(domainSpec, 16*1024*1024*1024)

		Expect(err).ToNot(HaveOccurred())
		Expect(domainSpec.Devices.Controllers).To(ConsistOf(api.Controller{
			Type:  api.ControllerTypePCI,
			Index: "0",
			Model: api.ControllerModelPCIeRoot,
			PCIHole64: &api.PCIHole64{
				Value: 33554432,
				Unit:  "KiB",
			},
		}))
	})

	It("normalizes an existing PCIe root controller with an implicit index", func() {
		domainSpec := newGraceConversionDomain()
		domainSpec.Devices.Controllers = []api.Controller{{Type: api.ControllerTypePCI, Model: api.ControllerModelPCIeRoot}}

		err := applyGracePCIHole64(domainSpec, 16*1024*1024*1024)

		Expect(err).ToNot(HaveOccurred())
		Expect(rootPCIController(domainSpec).Index).To(Equal("0"))
		Expect(rootPCIController(domainSpec).PCIHole64).To(Equal(&api.PCIHole64{Value: 33554432, Unit: "KiB"}))
	})

	It("rejects a PCIe root controller with a non-zero index", func() {
		domainSpec := newGraceConversionDomain()
		domainSpec.Devices.Controllers = []api.Controller{{Type: api.ControllerTypePCI, Index: "3", Model: api.ControllerModelPCIeRoot}}

		err := applyGracePCIHole64(domainSpec, 16*1024*1024*1024)

		Expect(err).To(MatchError(ContainSubstring("requires the PCIe root controller at index 0")))
	})

	It("rejects creating a duplicate PCI controller index zero", func() {
		domainSpec := newGraceConversionDomain()
		domainSpec.Devices.Controllers = []api.Controller{{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRootPort}}

		err := applyGracePCIHole64(domainSpec, 16*1024*1024*1024)

		Expect(err).To(MatchError(ContainSubstring("requires PCI controller index 0 to be pcie-root")))
		Expect(countGraceControllers(domainSpec, api.ControllerModelPCIeRoot)).To(Equal(0))
	})

	It("does not require host mapping or distances for unmapped zero-memory NUMA cells", func() {
		zeroMemoryKiB := uint64(0)
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		fakeRuntime.giNodes = uint32Range(2, 9)
		fakeRuntime.addDistances(append([]uint32{0}, fakeRuntime.giNodes...))
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)
		domainSpec.CPU.NUMA.Cells = append(domainSpec.CPU.NUMA.Cells, api.NUMACell{ID: "99", Memory: &zeroMemoryKiB, Unit: "KiB"})

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).ToNot(HaveOccurred())
		Expect(numaCellByID(domainSpec, "99").Distances).To(BeNil())
	})

	It("requires host mapping for NUMA cells with unspecified memory", func() {
		fakeRuntime.addGraceGPU("0000:81:00.0", 0, 16*1024*1024*1024)
		domainSpec := newGraceConversionDomain(
			newGraceTestHostDevice("gpu-gpu0", api.HostDevicePCI, "0x0000", "0x81", "0x00", "0x0"),
		)
		domainSpec.CPU.NUMA.Cells = append(domainSpec.CPU.NUMA.Cells, api.NUMACell{ID: "99"})

		err := configureGraceIOVirtualization(domainSpec, []string{"gpu-gpu0"}, true)

		Expect(err).To(MatchError(ContainSubstring("requires NUMATune memnode mapping for guest NUMA cell 99")))
	})

	It("documents which NUMA cells require host mapping", func() {
		zeroMemoryKiB := uint64(0)
		nonZeroMemoryKiB := uint64(1024)

		Expect(numaCellRequiresHostMapping(api.NUMACell{ID: "1", Memory: &zeroMemoryKiB})).To(BeFalse())
		Expect(numaCellRequiresHostMapping(api.NUMACell{ID: "1", Memory: &nonZeroMemoryKiB})).To(BeTrue())
		Expect(numaCellRequiresHostMapping(api.NUMACell{ID: "1"})).To(BeTrue())
		Expect(numaCellRequiresHostMapping(api.NUMACell{ID: "1", CPUs: "0"})).To(BeTrue())
	})
})

type fakeGraceRuntimeInfoProvider struct {
	pciIDs       map[string][2]string
	numaNodes    map[string]uint32
	pciHoleBytes map[string]uint64
	capabilities map[string]gracePCICapabilities
	giNodes      []uint32
	distances    map[uint32]map[uint32]uint64
	smmuv3       bool
}

func newFakeGraceRuntimeInfoProvider() *fakeGraceRuntimeInfoProvider {
	return &fakeGraceRuntimeInfoProvider{
		pciIDs:       map[string][2]string{},
		numaNodes:    map[string]uint32{},
		pciHoleBytes: map[string]uint64{},
		capabilities: map[string]gracePCICapabilities{},
		distances:    map[uint32]map[uint32]uint64{},
		smmuv3:       true,
	}
}

func (p *fakeGraceRuntimeInfoProvider) addGraceGPU(bdf string, numaNode uint32, pciHoleBytes uint64) {
	p.pciIDs[bdf] = [2]string{"0x10de", "0x2342"}
	p.numaNodes[bdf] = numaNode
	p.pciHoleBytes[bdf] = pciHoleBytes
}

func (p *fakeGraceRuntimeInfoProvider) addDistances(nodes []uint32) {
	for _, src := range nodes {
		p.distances[src] = map[uint32]uint64{}
		for _, dst := range nodes {
			if src == dst {
				p.distances[src][dst] = 10
			} else {
				p.distances[src][dst] = 80
			}
		}
	}
}

func (p *fakeGraceRuntimeInfoProvider) PCIIDs(bdf string) (string, string, error) {
	ids, exists := p.pciIDs[bdf]
	if !exists {
		return "", "", os.ErrNotExist
	}
	return ids[0], ids[1], nil
}

func (p *fakeGraceRuntimeInfoProvider) SMMUv3Available() (bool, error) { return p.smmuv3, nil }

func (p *fakeGraceRuntimeInfoProvider) PCINUMANode(bdf string) (uint32, error) {
	node, exists := p.numaNodes[bdf]
	if !exists {
		return 0, os.ErrNotExist
	}
	return node, nil
}

func (p *fakeGraceRuntimeInfoProvider) PCIHole64SizeBytes(bdf string) (uint64, error) {
	size, exists := p.pciHoleBytes[bdf]
	if !exists {
		return 0, os.ErrNotExist
	}
	return size, nil
}

func (p *fakeGraceRuntimeInfoProvider) PCICapabilities(bdf string) (gracePCICapabilities, error) {
	return p.capabilities[bdf], nil
}

func (p *fakeGraceRuntimeInfoProvider) GuestInitiatorHostNodes() ([]uint32, error) {
	return p.giNodes, nil
}

func (p *fakeGraceRuntimeInfoProvider) NUMADistances(node uint32) (map[uint32]uint64, error) {
	distances, exists := p.distances[node]
	if !exists {
		return nil, os.ErrNotExist
	}
	return distances, nil
}

func newGraceConversionDomain(hostDevices ...api.HostDevice) *api.DomainSpec {
	memoryKiB := uint64(1024)
	return &api.DomainSpec{
		CPU: api.CPU{NUMA: &api.NUMA{Cells: []api.NUMACell{
			{ID: "0", CPUs: "0-1", Memory: &memoryKiB, Unit: "KiB"},
		}}},
		NUMATune: &api.NUMATune{MemNodes: []api.MemNode{{CellID: 0, Mode: "strict", NodeSet: "0"}}},
		Devices: api.Devices{
			Controllers: []api.Controller{{Type: api.ControllerTypePCI, Index: "0", Model: api.ControllerModelPCIeRoot}},
			HostDevices: hostDevices,
		},
	}
}

func rootPCIController(domainSpec *api.DomainSpec) *api.Controller {
	for index := range domainSpec.Devices.Controllers {
		if domainSpec.Devices.Controllers[index].Model == api.ControllerModelPCIeRoot {
			return &domainSpec.Devices.Controllers[index]
		}
	}
	return nil
}

func countGraceControllers(domainSpec *api.DomainSpec, model string) int {
	count := 0
	for _, controller := range domainSpec.Devices.Controllers {
		if controller.Model == model {
			count++
		}
	}
	return count
}

func graceExpanderBusIndexes(domainSpec *api.DomainSpec) []string {
	var indexes []string
	for _, controller := range domainSpec.Devices.Controllers {
		if controller.Model != api.ControllerModelPCIeExpanderBus {
			continue
		}
		indexes = append(indexes, controller.Index)
	}
	return indexes
}

func numaCellByID(domainSpec *api.DomainSpec, id string) *api.NUMACell {
	for index := range domainSpec.CPU.NUMA.Cells {
		if domainSpec.CPU.NUMA.Cells[index].ID == id {
			return &domainSpec.CPU.NUMA.Cells[index]
		}
	}
	return nil
}

func findGraceSibling(siblings []api.NUMACellSibling, targetID string) *api.NUMACellSibling {
	for index := range siblings {
		if siblings[index].ID == targetID {
			return &siblings[index]
		}
	}
	return nil
}

func uint32Range(start, end uint32) []uint32 {
	var values []uint32
	for value := start; value <= end; value++ {
		values = append(values, value)
	}
	return values
}

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
