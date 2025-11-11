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
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type ChannelsDomainConfigurator struct{}

func (c ChannelsDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	domain.Spec.Devices.Channels = append(domain.Spec.Devices.Channels, newGuestAgentChannel())

	if downwardmetrics.HasDevice(&vmi.Spec) {
		// Handle downwardMetrics
		domain.Spec.Devices.Channels = append(domain.Spec.Devices.Channels, convertDownwardMetricsChannel())
	}

	return nil
}

func newGuestAgentChannel() api.Channel {
	return api.Channel{
		Type:   "unix",
		Source: nil, // let libvirt decide which path to use
		Target: &api.ChannelTarget{
			Name: "org.qemu.guest_agent.0",
			Type: v1.VirtIO,
		},
	}
}

func convertDownwardMetricsChannel() api.Channel {
	return api.Channel{
		Type: "unix",
		Source: &api.ChannelSource{
			Mode: "bind",
			Path: downwardmetrics.DownwardMetricsChannelSocket,
		},
		Target: &api.ChannelTarget{
			Type: v1.VirtIO,
			Name: downwardmetrics.DownwardMetricsSerialDeviceName,
		},
	}
}
