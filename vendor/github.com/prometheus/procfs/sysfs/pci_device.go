// Copyright 2025 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package sysfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

const pciDevicesPath = "bus/pci/devices"

// PciDeviceLocation represents the location of the device attached.
// "0000:00:00.0" represents Segment:Bus:Device.Function .
type PciDeviceLocation struct {
	Segment  int
	Bus      int
	Device   int
	Function int
}

func (pdl PciDeviceLocation) String() string {
	return fmt.Sprintf("%04x:%02x:%02x:%x", pdl.Segment, pdl.Bus, pdl.Device, pdl.Function)
}

func (pdl PciDeviceLocation) Strings() []string {
	return []string{
		fmt.Sprintf("%04x", pdl.Segment),
		fmt.Sprintf("%02x", pdl.Bus),
		fmt.Sprintf("%02x", pdl.Device),
		fmt.Sprintf("%x", pdl.Function),
	}
}

// PciDevice contains info from files in /sys/bus/pci/devices for a
// single PCI device.
type PciDevice struct {
	Location       PciDeviceLocation
	ParentLocation *PciDeviceLocation

	Class           uint32 // /sys/bus/pci/devices/<Location>/class
	Vendor          uint32 // /sys/bus/pci/devices/<Location>/vendor
	Device          uint32 // /sys/bus/pci/devices/<Location>/device
	SubsystemVendor uint32 // /sys/bus/pci/devices/<Location>/subsystem_vendor
	SubsystemDevice uint32 // /sys/bus/pci/devices/<Location>/subsystem_device
	Revision        uint32 // /sys/bus/pci/devices/<Location>/revision

	MaxLinkSpeed     *float64 // /sys/bus/pci/devices/<Location>/max_link_speed
	MaxLinkWidth     *float64 // /sys/bus/pci/devices/<Location>/max_link_width
	CurrentLinkSpeed *float64 // /sys/bus/pci/devices/<Location>/current_link_speed
	CurrentLinkWidth *float64 // /sys/bus/pci/devices/<Location>/current_link_width
}

func (pd PciDevice) Name() string {
	return pd.Location.String()
}

// PciDevices is a collection of every PCI device in
// /sys/bus/pci/devices .
//
// The map keys are the location of PCI devices.
type PciDevices map[string]PciDevice

// PciDevices returns info for all PCI devices read from
// /sys/bus/pci/devices .
func (fs FS) PciDevices() (PciDevices, error) {
	path := fs.sys.Path(pciDevicesPath)

	dirs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	pciDevs := make(PciDevices, len(dirs))
	for _, d := range dirs {
		device, err := fs.parsePciDevice(d.Name())
		if err != nil {
			return nil, err
		}

		pciDevs[device.Name()] = *device
	}

	return pciDevs, nil
}

func parsePciDeviceLocation(loc string) (*PciDeviceLocation, error) {
	locs := strings.Split(loc, ":")
	if len(locs) != 3 {
		return nil, fmt.Errorf("invalid location '%s'", loc)
	}
	locs = append(locs[0:2], strings.Split(locs[2], ".")...)
	if len(locs) != 4 {
		return nil, fmt.Errorf("invalid location '%s'", loc)
	}

	seg, err := strconv.ParseInt(locs[0], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid segment: %w", err)
	}
	bus, err := strconv.ParseInt(locs[1], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid bus: %w", err)
	}
	device, err := strconv.ParseInt(locs[2], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid device: %w", err)
	}
	function, err := strconv.ParseInt(locs[3], 16, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid function: %w", err)
	}

	return &PciDeviceLocation{
		Segment:  int(seg),
		Bus:      int(bus),
		Device:   int(device),
		Function: int(function),
	}, nil
}

// Parse one PCI device
// Refer to https://docs.kernel.org/PCI/sysfs-pci.html
func (fs FS) parsePciDevice(name string) (*PciDevice, error) {
	path := fs.sys.Path(pciDevicesPath, name)
	// the file must be symbolic link.
	realPath, err := os.Readlink(path)
	if err != nil {
		return nil, fmt.Errorf("failed to readlink: %w", err)
	}

	// parse device location from realpath
	// like "../../../devices/pci0000:00/0000:00:02.5/0000:04:00.0"
	deviceLocStr := filepath.Base(realPath)
	parentDeviceLocStr := filepath.Base(filepath.Dir(realPath))

	deviceLoc, err := parsePciDeviceLocation(deviceLocStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse device location:%q %w", deviceLoc, err)
	}

	// the parent device may have "pci" prefix.
	// this is not pci device like bridges.
	// we ignore such location to avoid confusion.
	// TODO: is it really ok?
	var parentDeviceLoc *PciDeviceLocation
	if !strings.HasPrefix(parentDeviceLocStr, "pci") {
		parentDeviceLoc, err = parsePciDeviceLocation(parentDeviceLocStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent device location %q: %w", parentDeviceLocStr, err)
		}
	}

	device := &PciDevice{
		Location:       *deviceLoc,
		ParentLocation: parentDeviceLoc,
	}

	// These files must exist in a device directory.
	for _, f := range [...]string{"class", "vendor", "device", "subsystem_vendor", "subsystem_device", "revision"} {
		name := filepath.Join(path, f)
		valueStr, err := util.SysReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %w", name, err)
		}
		value, err := strconv.ParseInt(valueStr, 0, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q: %w", valueStr, err)
		}

		switch f {
		case "class":
			device.Class = uint32(value)
		case "vendor":
			device.Vendor = uint32(value)
		case "device":
			device.Device = uint32(value)
		case "subsystem_vendor":
			device.SubsystemVendor = uint32(value)
		case "subsystem_device":
			device.SubsystemDevice = uint32(value)
		case "revision":
			device.Revision = uint32(value)
		default:
			return nil, fmt.Errorf("unknown file %q", f)
		}
	}

	for _, f := range [...]string{"max_link_speed", "max_link_width", "current_link_speed", "current_link_width"} {
		name := filepath.Join(path, f)
		valueStr, err := util.SysReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read file %q: %w", name, err)
		}

		// Some devices may be NULL or contain 'Unknown' as a value
		// values defined in drivers/pci/probe.c pci_speed_string
		if valueStr == "" || strings.HasPrefix(valueStr, "Unknown") {
			continue
		}

		switch f {
		case "max_link_speed", "current_link_speed":
			// example "8.0 GT/s PCIe"
			values := strings.SplitAfterN(valueStr, " ", 2)
			if len(values) != 2 {
				return nil, fmt.Errorf("invalid value for %s %q %s", f, valueStr, device.Location)
			}
			if values[1] != "GT/s PCIe" {
				return nil, fmt.Errorf("unknown unit for %s %q %s", f, valueStr, device.Location)
			}
			value, err := strconv.ParseFloat(strings.TrimSpace(values[0]), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s %q: %w", f, valueStr, err)
			}
			v := float64(value)
			switch f {
			case "max_link_speed":
				device.MaxLinkSpeed = &v
			case "current_link_speed":
				device.CurrentLinkSpeed = &v
			}

		case "max_link_width", "current_link_width":
			value, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %s %q: %w", f, valueStr, err)
			}
			v := float64(value)
			switch f {
			case "max_link_width":
				device.MaxLinkWidth = &v
			case "current_link_width":
				device.CurrentLinkWidth = &v
			}
		}
	}

	return device, nil
}
