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

package dra

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

const (
	failedCreateGPUHostDeviceFmt = "failed to create DRA GPU host-devices: %v"
	AliasPrefix                  = "dra-gpu-"
	DefaultDisplayOn             = true
)

func CreateDRAGPUHostDevices(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice
	if !hasGPUsWithDRA(vmi) {
		log.Log.Infof("No DRA GPU devices found for vmi %s/%s", vmi.GetNamespace(), vmi.GetName())
		return hostDevices, nil
	}
	draPCIHostDevices, err := getDRAPCIHostDevicesForGPUs(vmi)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}
	draMDEVHostDevices, err := getDRAMDEVHostDevicesForGPUs(vmi, DefaultDisplayOn)
	if err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

	hostDevices = append(hostDevices, draPCIHostDevices...)
	hostDevices = append(hostDevices, draMDEVHostDevices...)

	if err := validateCreationOfDRAGPUDevices(vmi.Spec.Domain.Devices.GPUs, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}
	return hostDevices, nil
}

func getDRAPCIHostDevicesForGPUs(vmi *v1.VirtualMachineInstance) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice
	if vmi.Status.DeviceStatus == nil {
		return hostDevices, fmt.Errorf("vmi has dra gpu devices but no device status found")
	}

	for _, gpu := range vmi.Status.DeviceStatus.GPUStatuses {
		gpu := gpu.DeepCopy()
		if gpu.DeviceResourceClaimStatus != nil && gpu.DeviceResourceClaimStatus.Attributes != nil {
			if gpu.DeviceResourceClaimStatus.Attributes.PCIAddress != nil {
				log.Log.V(2).Infof("Adding DRA PCI GPUdevice for %s", gpu.Name)
				hostAddr, err := device.NewPciAddressField(*gpu.DeviceResourceClaimStatus.Attributes.PCIAddress)
				if err != nil {
					return nil, fmt.Errorf("failed to create PCI device for %s: %v", gpu.Name, err)
				}
				hostDevices = append(hostDevices, api.HostDevice{
					Alias:   api.NewUserDefinedAlias(AliasPrefix + gpu.Name),
					Source:  api.HostDeviceSource{Address: hostAddr},
					Type:    api.HostDevicePCI,
					Managed: "no",
				})
			}
		}
	}
	return hostDevices, nil
}

func getDRAMDEVHostDevicesForGPUs(vmi *v1.VirtualMachineInstance, defaultDisplayOn bool) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice
	if vmi.Status.DeviceStatus == nil {
		return hostDevices, fmt.Errorf("vmi has dra devices but no device status found")
	}

	for _, gpu := range vmi.Status.DeviceStatus.GPUStatuses {
		gpu := gpu.DeepCopy()
		if gpu.DeviceResourceClaimStatus != nil && gpu.DeviceResourceClaimStatus.Attributes != nil {
			if gpu.DeviceResourceClaimStatus.Attributes.PCIAddress != nil {
				log.Log.V(2).Infof("Skipping DRA PCI GPU %s when processing for MDEV device", gpu.Name)
				continue
			}
			if gpu.DeviceResourceClaimStatus.Attributes.MDevUUID != nil {
				log.Log.V(2).Infof("Adding DRA MDEV GPU device for %s", gpu.Name)
				hostDevice := api.HostDevice{
					Alias: api.NewUserDefinedAlias(AliasPrefix + gpu.Name),
					Source: api.HostDeviceSource{
						Address: &api.Address{
							UUID: *gpu.DeviceResourceClaimStatus.Attributes.MDevUUID,
						},
					},
					Type:  api.HostDeviceMDev,
					Mode:  "subsystem",
					Model: "vfio-pci",
				}
				gpuSpec := getGPUSpecFromName(vmi, gpu.Name)
				if gpuSpec != nil && gpuSpec.VirtualGPUOptions != nil {
					if gpuSpec.VirtualGPUOptions.Display != nil {
						displayEnabled := gpuSpec.VirtualGPUOptions.Display.Enabled
						if displayEnabled == nil || *displayEnabled {
							hostDevice.Display = "on"
							if gpuSpec.VirtualGPUOptions.Display.RamFB == nil || *gpuSpec.VirtualGPUOptions.Display.RamFB.Enabled {
								hostDevice.RamFB = "on"
							}
						}
					}
				}
				hostDevices = append(hostDevices, hostDevice)
			}
		}
	}
	if defaultDisplayOn && !isVgpuDisplaySet(vmi.Spec.Domain.Devices.GPUs) && len(hostDevices) > 0 {
		hostDevices[0].Display = "on"
		hostDevices[0].RamFB = "on"
	}
	return hostDevices, nil
}

// hasGPUsWithDRA checks if the VMI has any GPU devices configured with DRA
func hasGPUsWithDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if drautil.IsGPUDRA(gpu) {
			return true
		}
	}
	return false
}

func getGPUSpecFromName(vmi *v1.VirtualMachineInstance, gpu string) *v1.GPU {
	for _, g := range vmi.Spec.Domain.Devices.GPUs {
		g := g
		if g.Name == gpu {
			return &g
		}
	}
	return nil
}

func isVgpuDisplaySet(gpuSpecs []v1.GPU) bool {
	for _, gpu := range gpuSpecs {
		if gpu.VirtualGPUOptions != nil &&
			gpu.VirtualGPUOptions.Display != nil {
			return true
		}
	}
	return false
}

// validateCreationOfDRAGPUDevices validates that all specified DRA GPU/s have a matching host-device.
// On validation failure, an error is returned.
// The validation assumes that the assignment of a device to a specified GPU is correct,
// therefore a simple quantity check is sufficient.
func validateCreationOfDRAGPUDevices(gpus []v1.GPU, hostDevices []api.HostDevice) error {
	gpusWithDRA := []v1.GPU{}
	for _, gpu := range gpus {
		if drautil.IsGPUDRA(gpu) {
			gpusWithDRA = append(gpusWithDRA, gpu)
		}
	}
	if len(gpusWithDRA) > 0 && len(gpusWithDRA) != len(hostDevices) {
		return fmt.Errorf(
			"the number of DRA GPU/s do not match the number of devices:\nGPU: %v\nDevice: %v", gpusWithDRA, hostDevices,
		)
	}
	return nil
}
