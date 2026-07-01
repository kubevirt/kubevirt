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
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	sysfsPCIDevicesPath = "/sys/bus/pci/devices"
	sysfsNodePath       = "/sys/devices/system/node"
	sysfsIOMMUClassPath = "/sys/class/iommu"

	graceGINodesPerGPU       = 8
	gracePCIHole64MarginKiB  = uint64(1024 * 1024)
	gracePCIHole64FloorBytes = uint64(512) << 30
	graceMaxPCIHole64KiB     = uint64(16) << 30

	// Linux exposes IORESOURCE_* bits in the sysfs PCI resource file along
	// with PCI BAR memory type bits in the low nibble.
	ioResourceMem                = uint64(0x00000200)
	pciBaseAddressMemoryTypeMask = uint64(0x0000000f)
	pciBaseAddressMemoryType64   = uint64(0x00000004)
	pciBaseAddressMemoryPrefetch = uint64(0x00000008)

	graceDefaultIOMMUAccel = "on"
	graceDefaultIOMMUATS   = "on"
	graceDefaultIOMMURIL   = "off"
	// Current Grace Blackwell and Vera Rubin systems support SSIDSize=20 and
	// OAS=48. Keep these defaults explicit until host SMMU capability probing is
	// available, then cap them to the host-supported values.
	graceDefaultIOMMUSSIDSize = "20"
	graceDefaultIOMMUOAS      = "48"
	graceSMMUv3IOMMUModel     = "smmuv3"
	graceHostDeviceIOMMUFDOn  = "yes"
)

type verifiedGraceHostDevice struct {
	Alias         string
	SourceAddress string
	HostDevice    *api.HostDevice
	VendorID      string
	DeviceID      string
}

type graceHostDeviceConversion struct {
	verifiedGraceHostDevice
	guestNUMANode uint32
	guestGINodes  []uint32
	hostGINodes   []uint32
	capabilities  gracePCICapabilities
	pciHoleBytes  uint64
}

type gracePCICapabilities struct {
	Accel    string
	ATS      string
	RIL      string
	SSIDSize string
	OAS      string
}

type graceRuntimeInfoProvider interface {
	PCIIDs(bdf string) (vendorID, deviceID string, err error)
	SMMUv3Available() (bool, error)
	PCINUMANode(bdf string) (uint32, error)
	PCIHole64SizeBytes(bdf string) (uint64, error)
	PCICapabilities(bdf string) (gracePCICapabilities, error)
	GuestInitiatorHostNodes() ([]uint32, error)
	GuestInitiatorHostNodesForDevice(bdf string) ([]uint32, error)
	NUMADistances(node uint32) (map[uint32]uint64, error)
}

type sysfsGraceRuntimeInfoProvider struct {
	pciDevicesPath string
	nodePath       string
	iommuClassPath string
}

var graceRuntimeInfo graceRuntimeInfoProvider = sysfsGraceRuntimeInfoProvider{
	pciDevicesPath: sysfsPCIDevicesPath,
	nodePath:       sysfsNodePath,
	iommuClassPath: sysfsIOMMUClassPath,
}

func configureGraceIOVirtualization(domainSpec *api.DomainSpec, expectedAliases []string, iommufdAvailable bool) error {
	if len(expectedAliases) == 0 {
		return nil
	}
	if !iommufdAvailable {
		return fmt.Errorf("GraceIOVirtualization requires an IOMMUFD file descriptor in virt-launcher")
	}

	smmuv3Available, err := graceRuntimeInfo.SMMUv3Available()
	if err != nil {
		return fmt.Errorf("failed to verify SMMUv3 availability for GraceIOVirtualization: %w", err)
	}
	if !smmuv3Available {
		return fmt.Errorf("GraceIOVirtualization requires SMMUv3 on the host")
	}

	verifiedDevices, err := verifyGraceHostDevices(domainSpec, expectedAliases)
	if err != nil {
		return err
	}
	if len(verifiedDevices) == 0 {
		return nil
	}

	conversionDevices, guestToHostNUMA, pciHoleBytes, err := prepareGraceHostDevices(domainSpec, verifiedDevices)
	if err != nil {
		return err
	}
	if _, err := ensureGracePCIeRootController(domainSpec); err != nil {
		return err
	}
	if err := placePCIDevicesWithGraceIOVirtualization(domainSpec, conversionDevices); err != nil {
		return err
	}
	if err := applyGracePCIHole64(domainSpec, pciHoleBytes); err != nil {
		return err
	}
	return applyGraceNUMADistances(domainSpec, guestToHostNUMA)
}

func verifyGraceHostDevices(domainSpec *api.DomainSpec, expectedAliases []string) ([]verifiedGraceHostDevice, error) {
	if len(expectedAliases) == 0 {
		return nil, nil
	}

	expected := map[string]bool{}
	for _, alias := range expectedAliases {
		if alias == "" {
			continue
		}
		expected[alias] = false
	}
	if len(expected) == 0 {
		return nil, nil
	}

	var verifiedDevices []verifiedGraceHostDevice
	for index := range domainSpec.Devices.HostDevices {
		hostDevice := &domainSpec.Devices.HostDevices[index]
		if hostDevice.Alias == nil {
			continue
		}

		alias := hostDevice.Alias.GetName()
		if _, exists := expected[alias]; !exists {
			continue
		}
		expected[alias] = true

		verifiedDevice, err := verifyGraceHostDevice(hostDevice, alias)
		if err != nil {
			return nil, err
		}
		verifiedDevices = append(verifiedDevices, verifiedDevice)
	}

	var missingAliases []string
	for alias, found := range expected {
		if !found {
			missingAliases = append(missingAliases, alias)
		}
	}
	if len(missingAliases) > 0 {
		sort.Strings(missingAliases)
		return nil, fmt.Errorf("GraceIOVirtualization expected hostdev aliases %s, but no matching host devices were assigned", strings.Join(missingAliases, ", "))
	}

	sort.Slice(verifiedDevices, func(i, j int) bool {
		return verifiedDevices[i].Alias < verifiedDevices[j].Alias
	})
	return verifiedDevices, nil
}

func verifyGraceHostDevice(hostDevice *api.HostDevice, alias string) (verifiedGraceHostDevice, error) {
	if hostDevice.Type != api.HostDevicePCI {
		return verifiedGraceHostDevice{}, fmt.Errorf("GraceIOVirtualization requires PCI hostdev %q, got %q", alias, hostDevice.Type)
	}
	if hostDevice.Source.Address == nil {
		return verifiedGraceHostDevice{}, fmt.Errorf("GraceIOVirtualization requires assigned PCI source address for hostdev %q", alias)
	}
	if hostDevice.Address != nil {
		return verifiedGraceHostDevice{}, fmt.Errorf("GraceIOVirtualization does not support explicit guest PCI address on hostdev %q", alias)
	}

	sourceAddress := hardware.PCIAddressToString(hostDevice.Source.Address)
	vendorID, deviceID, err := graceRuntimeInfo.PCIIDs(sourceAddress)
	if err != nil {
		return verifiedGraceHostDevice{}, fmt.Errorf("failed to read PCI identity for Grace hostdev %q at %s: %w", alias, sourceAddress, err)
	}
	if !hardware.IsNVIDIAGraceGPU(vendorID, deviceID) {
		return verifiedGraceHostDevice{}, fmt.Errorf("GraceIOVirtualization expected hostdev %q at %s to be a supported NVIDIA Grace GPU, got vendor %s device %s", alias, sourceAddress, hardware.NormalizePCIID(vendorID), hardware.NormalizePCIID(deviceID))
	}

	return verifiedGraceHostDevice{
		Alias:         alias,
		SourceAddress: sourceAddress,
		HostDevice:    hostDevice,
		VendorID:      hardware.NormalizePCIID(vendorID),
		DeviceID:      hardware.NormalizePCIID(deviceID),
	}, nil
}

func prepareGraceHostDevices(domainSpec *api.DomainSpec, verifiedDevices []verifiedGraceHostDevice) ([]graceHostDeviceConversion, map[uint32]uint32, uint64, error) {
	guestToHostNUMA, err := cpuGuestToHostNUMAMap(domainSpec)
	if err != nil {
		return nil, nil, 0, err
	}
	hostToGuestNUMA, err := hostToGuestNUMAMap(guestToHostNUMA)
	if err != nil {
		return nil, nil, 0, err
	}

	giHostNodesByDevice, giNodesCorrelated, err := graceHostGINodesByDevice(verifiedDevices)
	if err != nil {
		return nil, nil, 0, err
	}

	var giHostNodes []uint32
	if !giNodesCorrelated {
		giHostNodes, err = graceRuntimeInfo.GuestInitiatorHostNodes()
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to discover Grace Generic Initiator NUMA nodes: %w", err)
		}
		requiredGINodes := len(verifiedDevices) * graceGINodesPerGPU
		if len(giHostNodes) < requiredGINodes {
			return nil, nil, 0, fmt.Errorf("GraceIOVirtualization requires %d host Generic Initiator NUMA nodes for %d Grace GPU(s), found %d", requiredGINodes, len(verifiedDevices), len(giHostNodes))
		}
	}

	nextGuestCellID, err := nextNUMACellID(domainSpec)
	if err != nil {
		return nil, nil, 0, err
	}

	var conversionDevices []graceHostDeviceConversion
	var totalPCIHoleBytes uint64
	for index, verifiedDevice := range verifiedDevices {
		hostNUMANode, err := graceRuntimeInfo.PCINUMANode(verifiedDevice.SourceAddress)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read NUMA node for Grace hostdev %q at %s: %w", verifiedDevice.Alias, verifiedDevice.SourceAddress, err)
		}
		guestNUMANode, exists := hostToGuestNUMA[hostNUMANode]
		if !exists {
			return nil, nil, 0, fmt.Errorf("Grace hostdev %q is on host NUMA node %d, but no guest NUMA cell maps to that host node", verifiedDevice.Alias, hostNUMANode)
		}

		pciHoleBytes, err := graceRuntimeInfo.PCIHole64SizeBytes(verifiedDevice.SourceAddress)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to calculate pcihole64 for Grace hostdev %q at %s: %w", verifiedDevice.Alias, verifiedDevice.SourceAddress, err)
		}
		if pciHoleBytes == 0 {
			// A Grace GPU should expose 64-bit prefetchable BAR resources. Treat an
			// empty BAR footprint as a host/device configuration error instead of using
			// the Grace floor as a blind fallback.
			return nil, nil, 0, fmt.Errorf("Grace hostdev %q at %s has no 64-bit prefetchable PCI BARs for pcihole64 sizing", verifiedDevice.Alias, verifiedDevice.SourceAddress)
		}
		pciHoleBytes = gracePCIHole64SizeBytes(verifiedDevice, pciHoleBytes)
		if pciHoleBytes > math.MaxUint64-totalPCIHoleBytes {
			return nil, nil, 0, fmt.Errorf("Grace pcihole64 size overflows while adding hostdev %q at %s", verifiedDevice.Alias, verifiedDevice.SourceAddress)
		}
		totalPCIHoleBytes += pciHoleBytes

		capabilities, err := graceRuntimeInfo.PCICapabilities(verifiedDevice.SourceAddress)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to read SMMUv3 capabilities for Grace hostdev %q at %s: %w", verifiedDevice.Alias, verifiedDevice.SourceAddress, err)
		}
		capabilities = capabilities.withDefaults()

		guestGINodes := allocateGuestGINodes(nextGuestCellID, graceGINodesPerGPU)
		nextGuestCellID += uint32(graceGINodesPerGPU)
		var hostGINodes []uint32
		if giNodesCorrelated {
			hostGINodes = giHostNodesByDevice[verifiedDevice.SourceAddress]
		} else {
			hostGINodes = giHostNodes[index*graceGINodesPerGPU : (index+1)*graceGINodesPerGPU]
		}
		for giIndex, guestNode := range guestGINodes {
			guestToHostNUMA[guestNode] = hostGINodes[giIndex]
		}

		verifiedDevice.HostDevice.Driver = &api.HostDevDriver{Iommufd: graceHostDeviceIOMMUFDOn}
		verifiedDevice.HostDevice.ACPI = &api.ACPIHostDev{NodeSet: formatNUMANodeSet(guestGINodes)}
		appendGraceGINUMACells(domainSpec, guestGINodes)

		conversionDevices = append(conversionDevices, graceHostDeviceConversion{
			verifiedGraceHostDevice: verifiedDevice,
			guestNUMANode:           guestNUMANode,
			guestGINodes:            guestGINodes,
			hostGINodes:             hostGINodes,
			capabilities:            capabilities,
			pciHoleBytes:            pciHoleBytes,
		})
	}
	return conversionDevices, guestToHostNUMA, totalPCIHoleBytes, nil
}

func gracePCIHole64SizeBytes(verifiedDevice verifiedGraceHostDevice, barBytes uint64) uint64 {
	floorBytes := gracePCIHole64DeviceFloorBytes(verifiedDevice.VendorID, verifiedDevice.DeviceID)
	if barBytes < floorBytes {
		return floorBytes
	}
	return barBytes
}

func gracePCIHole64DeviceFloorBytes(vendorID, deviceID string) uint64 {
	if !hardware.IsNVIDIAGraceGPU(vendorID, deviceID) {
		return 0
	}
	// The NVIDIA Grace I/O guide requires a power-of-two aperture and documents
	// 4 TiB as sufficient for four Grace-Hopper/GB200 GPUs. Use a 512 GiB
	// per-device floor before aggregate rounding so 1, 2, and 4 GPUs produce 1,
	// 2, and 4 TiB root pcihole64 values. If GB300 PCI IDs are added, they need
	// their own larger floor because the guide requires 8 TiB for four GB300 GPUs.
	return gracePCIHole64FloorBytes
}

func graceHostGINodesByDevice(verifiedDevices []verifiedGraceHostDevice) (map[string][]uint32, bool, error) {
	hostGINodesByDevice := map[string][]uint32{}
	hostGINodeOwners := map[uint32]string{}
	anyCorrelated := false

	for _, verifiedDevice := range verifiedDevices {
		hostGINodes, err := graceRuntimeInfo.GuestInitiatorHostNodesForDevice(verifiedDevice.SourceAddress)
		if err != nil {
			return nil, false, fmt.Errorf("failed to discover Grace Generic Initiator NUMA nodes for hostdev %q at %s: %w", verifiedDevice.Alias, verifiedDevice.SourceAddress, err)
		}
		if len(hostGINodes) == 0 {
			continue
		}

		anyCorrelated = true
		sort.Slice(hostGINodes, func(i, j int) bool { return hostGINodes[i] < hostGINodes[j] })
		if len(hostGINodes) != graceGINodesPerGPU {
			return nil, true, fmt.Errorf("GraceIOVirtualization requires exactly %d host Generic Initiator NUMA nodes for hostdev %q at %s, found %d", graceGINodesPerGPU, verifiedDevice.Alias, verifiedDevice.SourceAddress, len(hostGINodes))
		}

		for _, hostGINode := range hostGINodes {
			if owner, exists := hostGINodeOwners[hostGINode]; exists {
				return nil, true, fmt.Errorf("Grace Generic Initiator NUMA node %d is associated with multiple Grace hostdevs (%s and %s)", hostGINode, owner, verifiedDevice.Alias)
			}
			hostGINodeOwners[hostGINode] = verifiedDevice.Alias
		}
		hostGINodesByDevice[verifiedDevice.SourceAddress] = append([]uint32(nil), hostGINodes...)
	}

	if !anyCorrelated {
		return nil, false, nil
	}
	if len(hostGINodesByDevice) != len(verifiedDevices) {
		var missingDevices []string
		for _, verifiedDevice := range verifiedDevices {
			if _, exists := hostGINodesByDevice[verifiedDevice.SourceAddress]; exists {
				continue
			}
			missingDevices = append(missingDevices, fmt.Sprintf("%s at %s", verifiedDevice.Alias, verifiedDevice.SourceAddress))
		}
		return nil, true, fmt.Errorf("Grace Generic Initiator NUMA nodes were correlated for some but not all Grace hostdevs; missing correlations for %s", strings.Join(missingDevices, ", "))
	}
	return hostGINodesByDevice, true, nil
}

func placePCIDevicesWithGraceIOVirtualization(domainSpec *api.DomainSpec, graceDevices []graceHostDeviceConversion) error {
	isolatedDevices := map[string]*api.IOMMUDevice{}
	numaOverrides := map[string]uint32{}
	for _, graceDevice := range graceDevices {
		iommuDevice := &api.IOMMUDevice{
			Model: graceSMMUv3IOMMUModel,
			Driver: &api.IOMMUDriver{
				Accel:    graceDevice.capabilities.Accel,
				ATS:      graceDevice.capabilities.ATS,
				RIL:      graceDevice.capabilities.RIL,
				SSIDSize: graceDevice.capabilities.SSIDSize,
				OAS:      graceDevice.capabilities.OAS,
			},
		}
		isolatedDevices[graceDevice.SourceAddress] = iommuDevice
		numaOverrides[graceDevice.SourceAddress] = graceDevice.guestNUMANode
	}

	assigner := newExpanderBusAssignerWithOptions(domainSpec, isolatedDevices, numaOverrides)
	return assigner.PlaceNumaAlignedDevices()
}

func cpuGuestToHostNUMAMap(domainSpec *api.DomainSpec) (map[uint32]uint32, error) {
	if domainSpec.CPU.NUMA == nil || len(domainSpec.CPU.NUMA.Cells) == 0 {
		return nil, fmt.Errorf("GraceIOVirtualization requires guest NUMA cells")
	}
	if domainSpec.NUMATune == nil || len(domainSpec.NUMATune.MemNodes) == 0 {
		return nil, fmt.Errorf("GraceIOVirtualization requires NUMATune memnodes to map guest NUMA cells to host NUMA nodes")
	}

	mapping := map[uint32]uint32{}
	for _, memNode := range domainSpec.NUMATune.MemNodes {
		hostNode, err := parseSingleNUMANodeSet(memNode.NodeSet)
		if err != nil {
			return nil, fmt.Errorf("invalid NUMATune memnode nodeset %q for guest cell %d: %w", memNode.NodeSet, memNode.CellID, err)
		}
		mapping[memNode.CellID] = hostNode
	}

	for _, cell := range domainSpec.CPU.NUMA.Cells {
		cellID, err := parseNUMACellID(cell.ID)
		if err != nil {
			return nil, err
		}
		if !numaCellRequiresHostMapping(cell) {
			continue
		}
		if _, exists := mapping[cellID]; !exists {
			return nil, fmt.Errorf("GraceIOVirtualization requires NUMATune memnode mapping for guest NUMA cell %d", cellID)
		}
	}
	return mapping, nil
}

func hostToGuestNUMAMap(guestToHost map[uint32]uint32) (map[uint32]uint32, error) {
	hostToGuest := map[uint32]uint32{}
	for guestNode, hostNode := range guestToHost {
		if existingGuestNode, exists := hostToGuest[hostNode]; exists {
			return nil, fmt.Errorf("host NUMA node %d maps to multiple guest NUMA cells (%d and %d)", hostNode, existingGuestNode, guestNode)
		}
		hostToGuest[hostNode] = guestNode
	}
	return hostToGuest, nil
}

func applyGraceNUMADistances(domainSpec *api.DomainSpec, guestToHostNUMA map[uint32]uint32) error {
	if domainSpec.CPU.NUMA == nil {
		return fmt.Errorf("GraceIOVirtualization requires guest NUMA cells for distance mapping")
	}

	guestNodeIDs := sortedUint32Keys(guestToHostNUMA)
	for cellIndex := range domainSpec.CPU.NUMA.Cells {
		cellID, err := parseNUMACellID(domainSpec.CPU.NUMA.Cells[cellIndex].ID)
		if err != nil {
			return err
		}
		hostNode, exists := guestToHostNUMA[cellID]
		if !exists {
			continue
		}

		hostDistances, err := graceRuntimeInfo.NUMADistances(hostNode)
		if err != nil {
			return fmt.Errorf("failed to read host NUMA distances for node %d: %w", hostNode, err)
		}

		siblings := make([]api.NUMACellSibling, 0, len(guestNodeIDs))
		for _, guestSiblingID := range guestNodeIDs {
			hostSiblingID := guestToHostNUMA[guestSiblingID]
			distance, exists := hostDistances[hostSiblingID]
			if !exists {
				return fmt.Errorf("missing host NUMA distance from node %d to node %d", hostNode, hostSiblingID)
			}
			// Distance 10 is the guest-local distance. Host GI nodes can report 10
			// for related but non-identical nodes, so bump non-self entries to keep
			// the guest NUMA distance matrix valid while preserving near-locality.
			if cellID != guestSiblingID && distance == 10 {
				distance = 11
			}
			siblings = append(siblings, api.NUMACellSibling{
				ID:    strconv.FormatUint(uint64(guestSiblingID), 10),
				Value: distance,
			})
		}
		domainSpec.CPU.NUMA.Cells[cellIndex].Distances = &api.NUMACellDistances{Siblings: siblings}
	}
	return nil
}

func applyGracePCIHole64(domainSpec *api.DomainSpec, pciHoleBytes uint64) error {
	pciHoleKiB := calculateGracePCIHole64KiB(pciHoleBytes, gracePCIHole64MarginKiB)
	if pciHoleKiB == 0 || pciHoleKiB > graceMaxPCIHole64KiB || pciHoleKiB > uint64(math.MaxUint) {
		return fmt.Errorf("invalid Grace pcihole64 size %d KiB", pciHoleKiB)
	}

	controller, err := ensureGracePCIeRootController(domainSpec)
	if err != nil {
		return err
	}
	if controller.PCIHole64 != nil && controller.PCIHole64.Value == 0 {
		return fmt.Errorf("GraceIOVirtualization conflicts with disabled pcihole64 on the PCIe root controller")
	}
	controller.PCIHole64 = &api.PCIHole64{Value: uint(pciHoleKiB), Unit: "KiB"}
	return nil
}

func ensureGracePCIeRootController(domainSpec *api.DomainSpec) (*api.Controller, error) {
	for index := range domainSpec.Devices.Controllers {
		controller := &domainSpec.Devices.Controllers[index]
		if controller.Type != api.ControllerTypePCI || controller.Model != api.ControllerModelPCIeRoot {
			continue
		}
		if controller.Index == "" {
			controller.Index = "0"
		}
		if controller.Index != "0" {
			return nil, fmt.Errorf("GraceIOVirtualization requires the PCIe root controller at index 0, found index %q", controller.Index)
		}
		return controller, nil
	}

	for _, controller := range domainSpec.Devices.Controllers {
		if controller.Type == api.ControllerTypePCI && controller.Index == "0" {
			return nil, fmt.Errorf("GraceIOVirtualization requires PCI controller index 0 to be pcie-root, found model %q", controller.Model)
		}
	}

	domainSpec.Devices.Controllers = append(domainSpec.Devices.Controllers, api.Controller{
		Type:  api.ControllerTypePCI,
		Index: "0",
		Model: api.ControllerModelPCIeRoot,
	})
	return &domainSpec.Devices.Controllers[len(domainSpec.Devices.Controllers)-1], nil
}

func calculateGracePCIHole64KiB(sizeBytes, marginKiB uint64) uint64 {
	if sizeBytes == 0 {
		return 0
	}
	sizeKiB := sizeBytes / 1024
	if sizeBytes%1024 != 0 {
		sizeKiB++
	}
	if sizeKiB > math.MaxUint64-marginKiB {
		return 0
	}
	sizeKiB += marginKiB

	var rounded uint64 = 1
	for rounded < sizeKiB {
		if rounded > math.MaxUint64/2 {
			return 0
		}
		rounded <<= 1
	}
	return rounded
}

func appendGraceGINUMACells(domainSpec *api.DomainSpec, guestGINodes []uint32) {
	for _, guestNode := range guestGINodes {
		zeroMemoryKiB := uint64(0)
		domainSpec.CPU.NUMA.Cells = append(domainSpec.CPU.NUMA.Cells, api.NUMACell{
			ID:     strconv.FormatUint(uint64(guestNode), 10),
			Memory: &zeroMemoryKiB,
			Unit:   "KiB",
		})
	}
}

func allocateGuestGINodes(start uint32, count int) []uint32 {
	nodes := make([]uint32, 0, count)
	for index := 0; index < count; index++ {
		nodes = append(nodes, start+uint32(index))
	}
	return nodes
}

func nextNUMACellID(domainSpec *api.DomainSpec) (uint32, error) {
	if domainSpec.CPU.NUMA == nil {
		return 0, fmt.Errorf("GraceIOVirtualization requires guest NUMA cells")
	}

	var maxID uint32
	for _, cell := range domainSpec.CPU.NUMA.Cells {
		cellID, err := parseNUMACellID(cell.ID)
		if err != nil {
			return 0, err
		}
		if cellID > maxID {
			maxID = cellID
		}
	}
	return maxID + 1, nil
}

func parseNUMACellID(cellID string) (uint32, error) {
	id, err := strconv.ParseUint(cellID, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid NUMA cell ID %q: %w", cellID, err)
	}
	return uint32(id), nil
}

func parseSingleNUMANodeSet(nodeSet string) (uint32, error) {
	if strings.ContainsAny(nodeSet, ",- ") {
		return 0, fmt.Errorf("nodeset must contain exactly one host NUMA node")
	}
	node, err := strconv.ParseUint(nodeSet, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(node), nil
}

func numaCellRequiresHostMapping(cell api.NUMACell) bool {
	if cell.CPUs != "" {
		return true
	}
	return cell.Memory == nil || *cell.Memory > 0
}

func formatNUMANodeSet(nodes []uint32) string {
	if len(nodes) == 0 {
		return ""
	}
	if len(nodes) == 1 {
		return strconv.FormatUint(uint64(nodes[0]), 10)
	}
	return fmt.Sprintf("%d-%d", nodes[0], nodes[len(nodes)-1])
}

func (capabilities gracePCICapabilities) withDefaults() gracePCICapabilities {
	if capabilities.Accel == "" {
		capabilities.Accel = graceDefaultIOMMUAccel
	}
	if capabilities.ATS == "" {
		capabilities.ATS = graceDefaultIOMMUATS
	}
	if capabilities.RIL == "" {
		capabilities.RIL = graceDefaultIOMMURIL
	}
	if capabilities.SSIDSize == "" {
		capabilities.SSIDSize = graceDefaultIOMMUSSIDSize
	}
	if capabilities.OAS == "" {
		capabilities.OAS = graceDefaultIOMMUOAS
	}
	return capabilities
}

func (p sysfsGraceRuntimeInfoProvider) PCIIDs(bdf string) (string, string, error) {
	vendorID, err := readSysfsValue(filepath.Join(p.pciDevicesPath, bdf, "vendor"))
	if err != nil {
		return "", "", err
	}
	deviceID, err := readSysfsValue(filepath.Join(p.pciDevicesPath, bdf, "device"))
	if err != nil {
		return "", "", err
	}
	return vendorID, deviceID, nil
}
func (p sysfsGraceRuntimeInfoProvider) SMMUv3Available() (bool, error) {
	entries, err := os.ReadDir(p.iommuClassPath)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		linkPath := filepath.Join(p.iommuClassPath, entry.Name())
		target, err := os.Readlink(linkPath)
		if err == nil && strings.Contains(target, "arm-smmu-v3") {
			return true, nil
		}
		if strings.Contains(entry.Name(), "arm-smmu-v3") {
			return true, nil
		}
	}
	return false, nil
}

func (p sysfsGraceRuntimeInfoProvider) PCINUMANode(bdf string) (uint32, error) {
	nodeValue, err := readSysfsValue(filepath.Join(p.pciDevicesPath, bdf, "numa_node"))
	if err != nil {
		return 0, err
	}
	node, err := strconv.Atoi(nodeValue)
	if err != nil {
		return 0, err
	}
	if node < 0 {
		return 0, fmt.Errorf("invalid NUMA node %d", node)
	}
	return uint32(node), nil
}

func (p sysfsGraceRuntimeInfoProvider) PCIHole64SizeBytes(bdf string) (uint64, error) {
	resourcePath := filepath.Join(p.pciDevicesPath, bdf, "resource")
	file, err := os.Open(resourcePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var total uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 3 {
			continue
		}
		start, err := strconv.ParseUint(strings.TrimPrefix(fields[0], "0x"), 16, 64)
		if err != nil {
			return 0, err
		}
		end, err := strconv.ParseUint(strings.TrimPrefix(fields[1], "0x"), 16, 64)
		if err != nil {
			return 0, err
		}
		flags, err := strconv.ParseUint(strings.TrimPrefix(fields[2], "0x"), 16, 64)
		if err != nil {
			return 0, err
		}
		if start == 0 || end == 0 || start > end {
			continue
		}
		if flags&ioResourceMem != ioResourceMem ||
			flags&pciBaseAddressMemoryTypeMask != pciBaseAddressMemoryType64|pciBaseAddressMemoryPrefetch {
			continue
		}
		total += end - start + 1
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return total, nil
}

func (p sysfsGraceRuntimeInfoProvider) PCICapabilities(_ string) (gracePCICapabilities, error) {
	// Grace Blackwell and Vera Rubin systems require explicit SMMUv3 address
	// capabilities, and the current platform values are SSIDSize=20 and OAS=48.
	// Longer term, these must be capped by the host SMMU capabilities rather
	// than hardcoded. OAS is visible in the arm-smmu-v3 probe log, while SSIDSize
	// requires decoding the SMMU ID registers. Use the known Grace defaults until
	// the kernel exposes these capabilities through a consumable interface.
	return gracePCICapabilities{}.withDefaults(), nil
}

func (p sysfsGraceRuntimeInfoProvider) GuestInitiatorHostNodes() ([]uint32, error) {
	giNodes, err := p.genericInitiatorNUMANodes()
	if err == nil {
		return giNodes, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return p.memorylessNoCPUNodes()
}

func (p sysfsGraceRuntimeInfoProvider) GuestInitiatorHostNodesForDevice(bdf string) ([]uint32, error) {
	giNodes, err := p.GuestInitiatorHostNodes()
	if err != nil {
		return nil, err
	}

	var matchedNodes []uint32
	for _, node := range giNodes {
		referencesDevice, err := p.nodeReferencesPCIAddress(node, bdf)
		if err != nil {
			return nil, err
		}
		if referencesDevice {
			matchedNodes = append(matchedNodes, node)
		}
	}
	return matchedNodes, nil
}

func (p sysfsGraceRuntimeInfoProvider) genericInitiatorNUMANodes() ([]uint32, error) {
	giNodesValue, err := readSysfsValue(filepath.Join(p.nodePath, "has_generic_initiator"))
	if err != nil {
		return nil, err
	}
	if giNodesValue == "" {
		return nil, nil
	}
	return parseHostNUMANodeSet(giNodesValue)
}

func (p sysfsGraceRuntimeInfoProvider) memorylessNoCPUNodes() ([]uint32, error) {
	onlineNodes, err := p.onlineNUMANodes()
	if err != nil {
		return nil, err
	}

	var giNodes []uint32
	for _, node := range onlineNodes {
		memoryless, err := p.isMemorylessNoCPUNode(node)
		if err != nil {
			return nil, err
		}
		if memoryless {
			giNodes = append(giNodes, node)
		}
	}
	sort.Slice(giNodes, func(i, j int) bool { return giNodes[i] < giNodes[j] })
	return giNodes, nil
}

func (p sysfsGraceRuntimeInfoProvider) NUMADistances(node uint32) (map[uint32]uint64, error) {
	onlineNodes, err := p.onlineNUMANodes()
	if err != nil {
		return nil, err
	}
	distanceValue, err := readSysfsValue(filepath.Join(p.nodePath, fmt.Sprintf("node%d", node), "distance"))
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(distanceValue)
	distances := map[uint32]uint64{}
	for index, field := range fields {
		if index >= len(onlineNodes) {
			break
		}
		distance, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return nil, err
		}
		distances[onlineNodes[index]] = distance
	}
	return distances, nil
}

func (p sysfsGraceRuntimeInfoProvider) onlineNUMANodes() ([]uint32, error) {
	onlineValue, err := readSysfsValue(filepath.Join(p.nodePath, "online"))
	if err != nil {
		return nil, err
	}
	return parseHostNUMANodeSet(onlineValue)
}

func (p sysfsGraceRuntimeInfoProvider) nodeReferencesPCIAddress(node uint32, bdf string) (bool, error) {
	devicePath, err := filepath.EvalSymlinks(filepath.Join(p.pciDevicesPath, bdf))
	if err != nil {
		return false, err
	}
	nodeDir := filepath.Join(p.nodePath, fmt.Sprintf("node%d", node))
	return directoryReferencesPCIAddress(nodeDir, devicePath, bdf, 2)
}

func directoryReferencesPCIAddress(dir, devicePath, bdf string, remainingDepth int) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dir, entry.Name())
		if entry.Name() == bdf {
			return true, nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			referencesDevice, err := symlinkReferencesPCIAddress(entryPath, devicePath, bdf)
			if err != nil {
				return false, err
			}
			if referencesDevice {
				return true, nil
			}
		}
		if remainingDepth == 0 || !entry.IsDir() || !shouldSearchNodeSubdirectory(entry.Name()) {
			continue
		}
		referencesDevice, err := directoryReferencesPCIAddress(entryPath, devicePath, bdf, remainingDepth-1)
		if err != nil {
			return false, err
		}
		if referencesDevice {
			return true, nil
		}
	}
	return false, nil
}

func symlinkReferencesPCIAddress(path, devicePath, bdf string) (bool, error) {
	linkTarget, err := os.Readlink(path)
	if err != nil {
		return false, err
	}
	if sysfsPathReferencesPCIAddress(linkTarget, bdf) {
		return true, nil
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false, nil
	}
	return resolvedPath == devicePath || sysfsPathReferencesPCIAddress(resolvedPath, bdf), nil
}

func sysfsPathReferencesPCIAddress(path, bdf string) bool {
	path = filepath.Clean(path)
	return filepath.Base(path) == bdf || strings.Contains(path, string(os.PathSeparator)+bdf+string(os.PathSeparator))
}

func shouldSearchNodeSubdirectory(name string) bool {
	switch name {
	case "hugepages", "memory_failure", "power", "x86":
		return false
	default:
		return true
	}
}

func parseHostNUMANodeSet(nodeSet string) ([]uint32, error) {
	nodes, err := hardware.ParseCPUSetLine(nodeSet, math.MaxInt)
	if err != nil {
		return nil, err
	}
	result := make([]uint32, 0, len(nodes))
	for _, node := range nodes {
		if node < 0 {
			continue
		}
		result = append(result, uint32(node))
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func (p sysfsGraceRuntimeInfoProvider) isMemorylessNoCPUNode(node uint32) (bool, error) {
	meminfo, err := readSysfsValue(filepath.Join(p.nodePath, fmt.Sprintf("node%d", node), "meminfo"))
	if err != nil {
		return false, err
	}
	memoryless := false
	for _, line := range strings.Split(meminfo, "\n") {
		if !strings.Contains(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		for index, field := range fields {
			if field == "MemTotal:" && index+1 < len(fields) {
				memoryless = fields[index+1] == "0"
			}
		}
	}
	if !memoryless {
		return false, nil
	}

	cpus, err := readSysfsValue(filepath.Join(p.nodePath, fmt.Sprintf("node%d", node), "cpulist"))
	if err != nil {
		return false, err
	}
	return cpus == "" || cpus == "none", nil
}

func readSysfsValue(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
