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

package libnet

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"

	"kubevirt.io/kubevirt/tests/libnet/cloudinit"
)

func WithMasqueradeNetworking(ports ...v1.Port) libvmi.Option {
	networkData := cloudinit.CreateDefaultCloudInitNetworkData()
	return func(vmi *v1.VirtualMachineInstance) {
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding(ports...))(vmi)
		libvmi.WithNetwork(v1.DefaultPodNetwork())(vmi)
		libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudNetworkData(networkData))(vmi)
	}
}
