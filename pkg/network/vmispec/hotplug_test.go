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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/tests/libvmi"
)

const (
	extraNetworkName           = "othernet"
	extraNetworkAttachmentName = "othernad"
)

var _ = Describe("NetworksToHotplug", func() {
	const (
		nadName     = "nad1"
		networkName = "n1"
	)

	DescribeTable("NetworksToHotplug", func(vmi *v1.VirtualMachineInstance, networksToHotplug ...v1.Network) {
		Expect(vmispec.NetworksToHotplug(vmi.Spec.Networks, vmi.Status.Interfaces)).To(ConsistOf(networksToHotplug))
	},
		Entry("with no networks in spec and status, there is nothing to hotplug", libvmi.NewAlpine()),
		Entry("with a network in spec that is missing from the status, hotplug it",
			dummyVMIWithOneNetworkAndOneIfaceOnSpec(networkName, nadName),
			v1.Network{
				Name: networkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: nadName,
					},
				},
			},
		),
		Entry(
			"with a network in spec and status, there is nothing to hotplug",
			dummyVMIWithOneNetworkAndOneIfaceOnSpecAndStatus(networkName, nadName),
		),
		Entry(
			"when multiple networks available in spec are missing from status, they are hot-plugged",
			dummyVMIWithMultipleNetworksAndIfacesOnSpec(networkName, nadName),
			v1.Network{
				Name: networkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: nadName,
					},
				},
			},
			v1.Network{
				Name: extraNetworkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: extraNetworkAttachmentName,
					}},
			},
		),
	)
})

func dummyVMIWithOneNetworkAndOneIfaceOnSpec(networkName string, nadName string) *v1.VirtualMachineInstance {
	return libvmi.NewAlpine(
		libvmi.WithNetwork(&v1.Network{
			Name: networkName,
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{
					NetworkName: nadName,
				},
			},
		}),
		libvmi.WithInterface(
			v1.Interface{
				Name:                   networkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
			},
		),
	)
}

func dummyVMIWithOneNetworkAndOneIfaceOnSpecAndStatus(networkName string, nadName string) *v1.VirtualMachineInstance {
	dummyVMI := dummyVMIWithOneNetworkAndOneIfaceOnSpec(networkName, nadName)
	dummyVMI.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
		{
			Name:          networkName,
			InterfaceName: "eno123",
		},
	}
	return dummyVMI
}

func dummyVMIWithMultipleNetworksAndIfacesOnSpec(networkName string, nadName string) *v1.VirtualMachineInstance {
	dummyVMI := dummyVMIWithOneNetworkAndOneIfaceOnSpec(networkName, nadName)
	dummyVMI.Spec.Domain.Devices.Interfaces = append(dummyVMI.Spec.Domain.Devices.Interfaces, v1.Interface{
		Name:                   extraNetworkName,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
	})
	dummyVMI.Spec.Networks = append(dummyVMI.Spec.Networks, v1.Network{
		Name: extraNetworkName,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: extraNetworkAttachmentName,
			}},
	})
	return dummyVMI
}
