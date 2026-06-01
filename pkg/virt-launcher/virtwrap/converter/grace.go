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
	"sort"
	"strings"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const sysfsPCIDevicesPath = "/sys/bus/pci/devices"

type verifiedGraceHostDevice struct {
	Alias         string
	SourceAddress string
	HostDevice    *api.HostDevice
	VendorID      string
	DeviceID      string
}

type graceRuntimeInfoProvider interface {
	PCIIDs(bdf string) (vendorID, deviceID string, err error)
}

type sysfsGraceRuntimeInfoProvider struct {
	pciDevicesPath string
}

var graceRuntimeInfo graceRuntimeInfoProvider = sysfsGraceRuntimeInfoProvider{pciDevicesPath: sysfsPCIDevicesPath}

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

func readSysfsValue(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
