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

package pci

import (
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func busNrFromTarget(target *api.ControllerTarget) (int, bool) {
	if target == nil || target.BusNr == nil {
		return 0, false
	}
	return int(*target.BusNr), true
}

// DisableHotplugOnOccupiedRootPorts sets hotplug="off" on pcie-root-port
// controllers that have a non-interface device behind them, preventing guests
// from ejecting critical devices (disks, balloon, RNG, etc.) at runtime.
// Interface (NIC) ports are left hotpluggable to preserve NIC hot-unplug.
//
// Must be called after the libvirt read-back so that Target.BusNr is populated.
//
// Note: when the PlacePCIDevicesOnRootComplex annotation is set, devices sit
// directly on the root bus with slot-based addressing (no pcie-root-ports),
// so they remain ejectable regardless of this function.
func DisableHotplugOnOccupiedRootPorts(spec *api.DomainSpec) {
	occupiedBuses := collectOccupiedPCIBuses(spec)

	for i, controller := range spec.Devices.Controllers {
		if controller.Model != api.ControllerModelPCIeRootPort {
			continue
		}
		bus, ok := busNrFromTarget(controller.Target)
		if !ok {
			continue
		}
		if _, occupied := occupiedBuses[bus]; occupied {
			if spec.Devices.Controllers[i].Target == nil {
				spec.Devices.Controllers[i].Target = &api.ControllerTarget{}
			}
			spec.Devices.Controllers[i].Target.Hotplug = "off"
		}
	}
}

func collectOccupiedPCIBuses(spec *api.DomainSpec) map[int]struct{} {
	addrs := collectNonInterfacePCIAddresses(spec)

	buses := make(map[int]struct{}, len(addrs))
	for _, addr := range addrs {
		if addr == nil || addr.Bus == "" {
			continue
		}
		if bus, err := strconv.ParseInt(strings.TrimPrefix(addr.Bus, "0x"), 16, 32); err == nil {
			buses[int(bus)] = struct{}{}
		}
	}
	return buses
}

// collectNonInterfacePCIAddresses returns PCI addresses of all devices except
// network interfaces, which are excluded to keep their root ports hotpluggable.
func collectNonInterfacePCIAddresses(spec *api.DomainSpec) []*api.Address {
	var addrs []*api.Address
	for i, disk := range spec.Devices.Disks {
		if disk.Target.Bus != v1.DiskBusVirtio {
			continue
		}
		addrs = append(addrs, spec.Devices.Disks[i].Address)
	}
	for _, controller := range spec.Devices.Controllers {
		if controller.Model == api.ControllerModelPCIRoot ||
			controller.Model == api.ControllerModelPCIeRoot ||
			controller.Model == api.ControllerModelPCIeRootPort ||
			controller.Model == api.ControllerModelPCIeExpanderBus {
			continue
		}
		addrs = append(addrs, controller.Address)
	}
	for i, input := range spec.Devices.Inputs {
		if input.Bus != v1.VirtIO {
			continue
		}
		addrs = append(addrs, spec.Devices.Inputs[i].Address)
	}
	for i := range spec.Devices.Watchdogs {
		addrs = append(addrs, spec.Devices.Watchdogs[i].Address)
	}
	for _, hostDev := range spec.Devices.HostDevices {
		if hostDev.Type != api.HostDevicePCI {
			continue
		}
		addrs = append(addrs, hostDev.Address)
	}
	if spec.Devices.Ballooning != nil {
		addrs = append(addrs, spec.Devices.Ballooning.Address)
	}
	if spec.Devices.Rng != nil {
		addrs = append(addrs, spec.Devices.Rng.Address)
	}
	if spec.Devices.Memory != nil {
		addrs = append(addrs, spec.Devices.Memory.Address)
	}
	return addrs
}
