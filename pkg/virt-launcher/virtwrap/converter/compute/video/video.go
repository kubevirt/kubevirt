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

package video

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	DefaultHeads uint = 1
	DefaultVRAM  uint = 16384
)

func ConfigureAMD64Device(vmi *v1.VirtualMachineInstance, domain *api.Domain, useBochsForEFIGuests bool) {
	// For AMD64 + EFI, use bochs. For BIOS, use VGA
	if useBochsForEFIGuests && vmi.IsBootloaderEFI() {
		domain.Spec.Devices.Video = []api.Video{
			{
				Model: api.VideoModel{
					Type:  "bochs",
					Heads: new(DefaultHeads),
				},
			},
		}
	} else {
		domain.Spec.Devices.Video = []api.Video{
			{
				Model: api.VideoModel{
					Type:  "vga",
					Heads: new(DefaultHeads),
					VRam:  new(DefaultVRAM),
				},
			},
		}
	}
}

func ConfigureVirtioDevice(_ *v1.VirtualMachineInstance, domain *api.Domain, _ bool) {
	domain.Spec.Devices.Video = []api.Video{
		{
			Model: api.VideoModel{
				Type:  v1.VirtIO,
				Heads: new(DefaultHeads),
			},
		},
	}
}
