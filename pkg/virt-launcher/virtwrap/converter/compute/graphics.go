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

package compute

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute/video"
)

type GraphicsDomainConfigurator struct {
	defaultVideoDevice video.DeviceConfigurator
	crossArchEmulation bool
}

func NewGraphicsDomainConfigurator(architecture string, useBochsForEFIGuests, crossArchEmulation bool) GraphicsDomainConfigurator {
	return GraphicsDomainConfigurator{
		defaultVideoDevice: video.NewDeviceConfigurator(architecture, useBochsForEFIGuests),
		crossArchEmulation: crossArchEmulation,
	}
}

func (g GraphicsDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && !*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice {
		return nil
	}

	// Cross-architecture emulation uses a minimal QEMU binary that may not
	// include video device support (e.g. virtio-gpu). Disable graphics to
	// avoid libvirt validation errors; serial console remains available.
	if g.crossArchEmulation {
		return nil
	}

	domain.Spec.Devices.Graphics = []api.Graphics{
		{
			Listen: &api.GraphicsListen{
				Type:   "socket",
				Socket: fmt.Sprintf("/var/run/kubevirt-private/%s/virt-vnc", vmi.ObjectMeta.UID),
			},
			Type: "vnc",
		},
	}

	g.configureVideoDevice(vmi, domain)

	return nil
}

func (g GraphicsDomainConfigurator) configureVideoDevice(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if vmi.Spec.Domain.Devices.Video != nil {
		videoDevice := api.Video{
			Model: api.VideoModel{
				Type:  vmi.Spec.Domain.Devices.Video.Type,
				VRam:  new(video.DefaultVRAM),
				Heads: new(video.DefaultHeads),
			},
		}
		domain.Spec.Devices.Video = []api.Video{videoDevice}
		return
	}

	if g.defaultVideoDevice != nil {
		g.defaultVideoDevice.ConfigureVideoDevice(vmi, domain)
	}
}
