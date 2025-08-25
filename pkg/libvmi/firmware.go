/*
Copyright The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package libvmi

import (
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

// WithUefi configures EFI bootloader and SecureBoot.
func WithUefi(secureBoot bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader.EFI == nil {
			vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.EFI.SecureBoot = pointer.P(secureBoot)
		// secureBoot Requires SMM to be enabled
		if secureBoot {
			if vmi.Spec.Domain.Features == nil {
				vmi.Spec.Domain.Features = &v1.Features{}
			}
			if vmi.Spec.Domain.Features.SMM == nil {
				vmi.Spec.Domain.Features.SMM = &v1.FeatureState{}
			}
			vmi.Spec.Domain.Features.SMM.Enabled = pointer.P(secureBoot)
		}
	}
}

// WithPersistentUefi configures the Persistent config in the EFI bootloader.
func WithPersistentUefi(persistent bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader == nil {
			vmi.Spec.Domain.Firmware.Bootloader = &v1.Bootloader{}
		}
		if vmi.Spec.Domain.Firmware.Bootloader.EFI == nil {
			vmi.Spec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
		}
		vmi.Spec.Domain.Firmware.Bootloader.EFI.Persistent = pointer.P(persistent)
	}
}

func WithKernelBootContainer(imageName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Firmware = &v1.Firmware{
			KernelBoot: &v1.KernelBoot{
				Container: &v1.KernelBootContainer{
					Image: imageName,
				},
			},
		}
	}
}

func WithKernelBootContainerImagePullSecret(imagePullSecret string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		if vmi.Spec.Domain.Firmware.KernelBoot == nil {
			vmi.Spec.Domain.Firmware.KernelBoot = &v1.KernelBoot{}
		}
		if vmi.Spec.Domain.Firmware.KernelBoot.Container == nil {
			vmi.Spec.Domain.Firmware.KernelBoot.Container = &v1.KernelBootContainer{}
		}
		vmi.Spec.Domain.Firmware.KernelBoot.Container.ImagePullSecret = imagePullSecret
	}
}

func WithFirmwareUUID(uid types.UID) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}
		vmi.Spec.Domain.Firmware.UUID = uid
	}
}
