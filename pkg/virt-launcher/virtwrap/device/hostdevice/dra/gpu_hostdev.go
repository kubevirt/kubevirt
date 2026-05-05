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

	k8sv1 "k8s.io/api/core/v1"

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

// CreateDRAGPUHostDevices creates host devices for GPUs allocated via DRA.
func CreateDRAGPUHostDevices(vmi *v1.VirtualMachineInstance, basePath string) ([]api.HostDevice, error) {
	var hostDevices []api.HostDevice
	if !hasGPUsWithDRA(vmi) {
		log.Log.V(3).Infof("No DRA GPU devices found for vmi %s/%s", vmi.GetNamespace(), vmi.GetName())
		return hostDevices, nil
	}

	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if !drautil.IsGPUDRA(gpu) {
			continue
		}

		hostDevice, err := createHostDeviceForGPU(gpu, basePath, vmi.Spec.ResourceClaims)
		if err != nil {
			return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
		}
		if hostDevice != nil {
			hostDevices = append(hostDevices, *hostDevice)
		}
	}

	if err := validateCreationOfDRAGPUDevices(vmi.Spec.Domain.Devices.GPUs, hostDevices); err != nil {
		return nil, fmt.Errorf(failedCreateGPUHostDeviceFmt, err)
	}

	// Set default display on first vGPU if not explicitly set
	if DefaultDisplayOn && !isVgpuDisplaySet(vmi.Spec.Domain.Devices.GPUs) {
		for i := range hostDevices {
			if hostDevices[i].Type == api.HostDeviceMDev {
				hostDevices[i].Display = "on"
				hostDevices[i].RamFB = "on"
				break
			}
		}
	}

	return hostDevices, nil
}

func createHostDeviceForGPU(gpu v1.GPU, basePath string, resourceClaims []k8sv1.PodResourceClaim) (*api.HostDevice, error) {
	if gpu.ClaimRequest == nil || gpu.ClaimRequest.ClaimName == nil || *gpu.ClaimRequest.ClaimName == "" || gpu.ClaimRequest.RequestName == nil || *gpu.ClaimRequest.RequestName == "" {
		return nil, fmt.Errorf("GPU %s has incomplete ClaimRequest", gpu.Name)
	}

	claimName := *gpu.ClaimRequest.ClaimName
	requestName := *gpu.ClaimRequest.RequestName

	// Check mdevUUID first: a device with both pciBusID and mdevUUID is a
	// mediated (vGPU) device whose parent happens to expose pciBusID. Treating
	// it as PCI passthrough would be incorrect.
	mdevUUID, mdevErr := drautil.GetMDevUUIDForClaim(basePath, resourceClaims, claimName, requestName)
	if mdevErr == nil {
		log.Log.V(2).Infof("Adding DRA MDEV GPU device for %s", gpu.Name)
		hostDevice := api.HostDevice{
			Alias: api.NewUserDefinedAlias(AliasPrefix + gpu.Name),
			Source: api.HostDeviceSource{
				Address: &api.Address{
					UUID: mdevUUID,
				},
			},
			Type:  api.HostDeviceMDev,
			Mode:  "subsystem",
			Model: "vfio-pci",
		}

		if gpu.VirtualGPUOptions != nil && gpu.VirtualGPUOptions.Display != nil {
			displayEnabled := gpu.VirtualGPUOptions.Display.Enabled
			if displayEnabled == nil || *displayEnabled {
				hostDevice.Display = "on"
				if gpu.VirtualGPUOptions.Display.RamFB == nil || *gpu.VirtualGPUOptions.Display.RamFB.Enabled {
					hostDevice.RamFB = "on"
				}
			}
		}
		return &hostDevice, nil
	}

	pciAddr, pciErr := drautil.GetPCIAddressForClaim(basePath, resourceClaims, claimName, requestName)
	if pciErr == nil {
		log.Log.V(2).Infof("Adding DRA PCI GPU device for %s", gpu.Name)
		hostAddr, err := device.NewPciAddressField(pciAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to create PCI device for %s: %v", gpu.Name, err)
		}
		return &api.HostDevice{
			Alias:   api.NewUserDefinedAlias(AliasPrefix + gpu.Name),
			Source:  api.HostDeviceSource{Address: hostAddr},
			Type:    api.HostDevicePCI,
			Managed: "no",
		}, nil
	}

	return nil, fmt.Errorf("GPU %s has no mdevUUID or pciBusID in metadata for claim %s request %s (mdev: %v, pci: %v)", gpu.Name, claimName, requestName, mdevErr, pciErr)
}

func hasGPUsWithDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if drautil.IsGPUDRA(gpu) {
			return true
		}
	}
	return false
}

func isVgpuDisplaySet(gpuSpecs []v1.GPU) bool {
	for _, gpu := range gpuSpecs {
		if gpu.VirtualGPUOptions != nil && gpu.VirtualGPUOptions.Display != nil {
			return true
		}
	}
	return false
}

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
