/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vmispec

import v1 "kubevirt.io/api/core/v1"

// RequiresVirtioNetDevice checks whether a VMI requires the presence of the "virtio" net device.
// This happens when the VMI wants to use a "virtio" network interface, and software emulation is disallowed.
func RequiresVirtioNetDevice(vmi *v1.VirtualMachineInstance, allowEmulation bool) bool {
	return hasVirtioIface(vmi) && !allowEmulation
}

func RequiresTunDevice(vmi *v1.VirtualMachineInstance) bool {
	return (len(vmi.Spec.Domain.Devices.Interfaces) > 0) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface == nil) ||
		(*vmi.Spec.Domain.Devices.AutoattachPodInterface)
}
