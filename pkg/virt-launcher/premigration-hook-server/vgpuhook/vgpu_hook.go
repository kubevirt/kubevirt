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
 */

package vgpuhook

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func constructHostDevMap(hostDevs []libvirtxml.DomainHostdev) map[string]*libvirtxml.DomainHostdev {
	hostDevMap := make(map[string]*libvirtxml.DomainHostdev, len(hostDevs))
	for i := range hostDevs {
		hostDev := &hostDevs[i]
		if hostDev.Alias == nil {
			continue
		}
		// Keep the first device for a repeated alias, matching the previous linear scan.
		if _, exists := hostDevMap[hostDev.Alias.Name]; exists {
			continue
		}
		hostDevMap[hostDev.Alias.Name] = hostDev
	}
	return hostDevMap
}

func matchHostDevice(gpu api.HostDevice, hostDevsMap map[string]*libvirtxml.DomainHostdev) (*libvirtxml.DomainHostdev, error) {
	if gpu.Alias == nil {
		return nil, fmt.Errorf("GPU host device has no alias")
	}
	want := api.UserAliasPrefix + gpu.Alias.GetName()
	hostDev, ok := hostDevsMap[want]
	if !ok {
		return nil, fmt.Errorf("no matching host device for GPU alias %s", want)
	}
	return hostDev, nil
}

// VGPULiveMigration mutates the mdev uuid for the target's domain XML in vGPU live migrations
func VGPULiveMigration(c *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	gpuDevs := c.GPUHostDevices

	if len(gpuDevs) == 0 || domain.Devices == nil || len(domain.Devices.Hostdevs) == 0 {
		return nil
	}

	hostDevMap := constructHostDevMap(domain.Devices.Hostdevs)
	for _, gpuDev := range gpuDevs {
		if gpuDev.Source.Address == nil {
			return fmt.Errorf("failed to retrieve host GPU address for host device")
		}
		if gpuDev.Type != api.HostDeviceMDev {
			return fmt.Errorf("unsupporting gpu type for migration: %s", gpuDev.Type)
		}
		hostDev, err := matchHostDevice(gpuDev, hostDevMap)
		if err != nil {
			return fmt.Errorf("failed to locate corresponding host device for GPU: %v", err)
		}
		if hostDev.SubsysMDev == nil {
			return fmt.Errorf("failed to retrieve mdev vGPU from domain")
		}
		if hostDev.SubsysMDev.Source.Address == nil {
			return fmt.Errorf("failed to retrieve host GPU address")
		}
		hostDev.SubsysMDev.Source.Address.UUID = gpuDev.Source.Address.UUID
	}

	log.Log.Object(vmi).Info("vGPU-hook: mdev uuid mutation completed")
	return nil
}
