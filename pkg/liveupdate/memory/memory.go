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

package memory

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// must be a power of 2 and at least equal
	// to the size of a transparent hugepage (2MiB on x84_64).
	// Recommended value by QEMU is 2MiB
	HotplugBlockAlignmentBytes int64 = 0x200000

	// 1GiB, the size of 1Gi HugePages
	Hotplug1GHugePagesBlockAlignmentBytes int64 = 0x40000000

	// requiredMinGuestMemory is the minimum required memory
	// for a VM to have memory hotplug enabled.
	//
	// The 1GiB mark is chosen as a tradeoff, it is enough
	// memory for the guest kernel to allocate its internal data
	// structures and to allocate the swiotlb, which is usually
	// 64MB. It also means we can memory map all PCI devices
	// as they're memory mapped in the first 1Gi (PCI hole).
	requiredMinGuestMemory = 0x40000000
)

func ValidateLiveUpdateMemory(vmSpec *v1.VirtualMachineInstanceSpec, maxGuest *resource.Quantity) error {
	domain := &vmSpec.Domain

	if domain.CPU != nil && domain.CPU.Realtime != nil {
		return fmt.Errorf("Memory hotplug is not compatible with realtime VMs")
	}

	if domain.CPU != nil &&
		domain.CPU.NUMA != nil &&
		domain.CPU.NUMA.GuestMappingPassthrough != nil {
		return fmt.Errorf("Memory hotplug is not compatible with guest mapping passthrough")
	}

	if domain.LaunchSecurity != nil {
		return fmt.Errorf("Memory hotplug is not compatible with encrypted VMs")
	}

	blockAlignment := HotplugBlockAlignmentBytes
	if domain.Memory != nil &&
		domain.Memory.Hugepages != nil &&
		domain.Memory.Hugepages.PageSize == "1Gi" {
		blockAlignment = Hotplug1GHugePagesBlockAlignmentBytes
	}

	if domain.Memory == nil ||
		domain.Memory.Guest == nil {
		return fmt.Errorf("Guest memory must be configured when memory hotplug is enabled")
	}
	if maxGuest == nil {
		return fmt.Errorf("Max guest memory must be configured when memory hotplug is enabled")
	}

	if domain.Memory.Guest.Cmp(*maxGuest) > 0 {
		return fmt.Errorf("Guest memory is greater than the configured maxGuest memory")
	}
	if domain.Memory.Guest.Value()%blockAlignment != 0 {
		alignment := resource.NewQuantity(blockAlignment, resource.BinarySI)
		return fmt.Errorf("Guest memory must be %s aligned", alignment)
	}

	if maxGuest.Value()%blockAlignment != 0 {
		alignment := resource.NewQuantity(blockAlignment, resource.BinarySI)
		return fmt.Errorf("MaxGuest must be %s aligned", alignment)
	}

	if vmSpec.Architecture != "amd64" &&
		vmSpec.Architecture != "arm64" {
		return fmt.Errorf("Memory hotplug is only available for x86_64 and arm64 VMs")
	}

	if domain.Memory.Guest.Value() < requiredMinGuestMemory {
		return fmt.Errorf("Memory hotplug is only available for VMs with at least 1Gi of guest memory")
	}

	return nil
}

func BuildMemoryDevice(vmi *v1.VirtualMachineInstance) (*api.MemoryDevice, error) {
	domain := vmi.Spec.Domain

	pluggableMemory := domain.Memory.MaxGuest.DeepCopy()
	pluggableMemory.Sub(*vmi.Status.Memory.GuestAtBoot)
	pluggableMemorySize, err := vcpu.QuantityToByte(pluggableMemory)
	if err != nil {
		return nil, err
	}

	requestedHotPlugMemory := domain.Memory.Guest.DeepCopy()
	requestedHotPlugMemory.Sub(*vmi.Status.Memory.GuestAtBoot)
	pluggableMemoryRequested, err := vcpu.QuantityToByte(requestedHotPlugMemory)
	if err != nil {
		return nil, err
	}

	blockAlignment := HotplugBlockAlignmentBytes
	if domain.Memory != nil &&
		domain.Memory.Hugepages != nil &&
		domain.Memory.Hugepages.PageSize == "1Gi" {
		blockAlignment = Hotplug1GHugePagesBlockAlignmentBytes
	}

	return &api.MemoryDevice{
		Model: "virtio-mem",
		Target: &api.MemoryTarget{
			Size:      pluggableMemorySize,
			Node:      "0",
			Block:     api.Memory{Unit: "b", Value: uint64(blockAlignment)},
			Requested: pluggableMemoryRequested,
		},
	}, nil
}
