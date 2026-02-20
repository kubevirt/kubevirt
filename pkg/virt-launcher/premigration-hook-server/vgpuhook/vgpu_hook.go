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
)

const TargetMdevUUIDAnnotation = "kubevirt.io/target-mdev-uuid"

// VGPUDedicatedHook mutates the mdev uuid for the target's domain XML in vGPU live migrations
func VGPUDedicatedHook(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if len(vmi.Spec.Domain.Devices.GPUs) == 0 {
		return nil
	}
	if len(vmi.Spec.Domain.Devices.GPUs) != 1 {
		return fmt.Errorf("the migrating vmi can only have one vGPU")
	}
	if len(vmi.Spec.Domain.Devices.HostDevices) != 0 {
		return fmt.Errorf("the migrating vmi cannot have any non vGPU hostdevices")
	}

	mdevUUID, ok := vmi.Annotations[TargetMdevUUIDAnnotation]
	if !ok {
		return fmt.Errorf("missing vmi annotation target-mdev-uuid")
	}

	// need to check for type=mdev so we don't try to migrate a passthrough GPU
	if len(domain.Devices.Hostdevs) == 1 && domain.Devices.Hostdevs[0].SubsysMDev != nil {
		domain.Devices.Hostdevs[0].SubsysMDev.Source.Address.UUID = mdevUUID
	} else {
		return fmt.Errorf("failed to retrieve mdev vGPU from domain")
	}

	log.Log.Object(vmi).Info("vGPU-hook: mdev uuid mutation completed")
	return nil
}
