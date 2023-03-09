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

package virtwrap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("nic hotplug on virt-launcher", func() {
	const (
		nadName     = "n1n"
		networkName = "n1"
	)

	DescribeTable("networksToHotplugWhoseInterfacesAreNotInTheDomain", func(vmi *v1.VirtualMachineInstance, domainIfaces map[string]api.Interface, expectedNetworks []v1.Network) {
		Expect(
			networksToHotplugWhoseInterfacesAreNotInTheDomain(vmi, domainIfaces),
		).To(ConsistOf(expectedNetworks))
	},
		Entry("vmi with no networks, and no interfaces in the domain",
			&v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Networks: []v1.Network{}}},
			map[string]api.Interface{},
			nil,
		),
		Entry("vmi with 1 network, and an associated interface in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{Networks: []v1.Network{generateNetwork(networkName, nadName)}},
			},
			map[string]api.Interface{networkName: {Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("vmi with 1 network (when the pod interface is *not* ready), with no interfaces in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{Networks: []v1.Network{generateNetwork(networkName, nadName)}},
			},
			map[string]api.Interface{},
			nil,
		),
		Entry("vmi with 1 network (when the pod interface *is* ready), but already present in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{Networks: []v1.Network{generateNetwork(networkName, nadName)}},
				Status: v1.VirtualMachineInstanceStatus{
					Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
						Name:       networkName,
						InfoSource: vmispec.InfoSourceMultusStatus,
					}},
				},
			},
			map[string]api.Interface{networkName: {Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("vmi with 1 network (when the pod interface *is* ready), but no interfaces in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{Networks: []v1.Network{generateNetwork(networkName, nadName)}},
				Status: v1.VirtualMachineInstanceStatus{
					Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
						Name:       networkName,
						InfoSource: vmispec.InfoSourceMultusStatus,
					}},
				},
			},
			map[string]api.Interface{},
			[]v1.Network{generateNetwork(networkName, nadName)},
		),
	)
})

func generateNetwork(name string, nadName string) v1.Network {
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{NetworkName: nadName}},
	}
}
