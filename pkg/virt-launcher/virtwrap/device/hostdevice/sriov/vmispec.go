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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package sriov

import (
	v1 "kubevirt.io/api/core/v1"
)

func filterVMISRIOVInterfaces(vmi *v1.VirtualMachineInstance) []v1.Interface {
	var interfaces []v1.Interface
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			interfaces = append(interfaces, iface)
		}
	}
	return interfaces
}

func filterMultusNonDefaultNetworks(vmi *v1.VirtualMachineInstance) []v1.Network {
	var multusNetworks []v1.Network
	for _, network := range vmi.Spec.Networks {
		if network.Multus != nil {
			if network.Multus.Default {
				continue
			}
			multusNetworks = append(multusNetworks, network)
		}
	}
	return multusNetworks
}
