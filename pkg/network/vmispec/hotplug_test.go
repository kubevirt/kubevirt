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
)

const (
	extraNetworkName           = "othernet"
	extraNetworkAttachmentName = "othernad"
)

var _ = Describe("utilitary funcs to identify attachments to hotplug", func() {
	const (
		guestIfaceName = "eno123"
		nadName        = "nad1"
		networkName    = "n1"
	)

	DescribeTable("NetworksToHotplugWhosePodIfacesAreReady", func(vmi *v1.VirtualMachineInstance, networksToHotplug ...v1.Network) {
		Expect(vmispec.NetworksToHotplugWhosePodIfacesAreReady(vmi)).To(ConsistOf(networksToHotplug))
	},
		Entry("VMI without networks in spec does not have anything to hotplug", newVMI()),
		Entry("VMI with networks in spec, but not marked as ready in the status are *not* subject to hotplug",
			dummyVMIWithoutStatus(networkName, nadName),
		),
		Entry("VMI with networks in spec, marked as ready in the status, but not yet available in the domain *is* subject to hotplug",
			dummyVMIWithAttachmentToPlug(networkName, nadName, guestIfaceName),
			v1.Network{
				Name: networkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{
						NetworkName: nadName,
					},
				},
			},
		),
		Entry("VMI with networks in spec, marked as ready in the status, but already present in the domain *not* subject to hotplug",
			dummyVMIWithAttachmentAlreadyAvailableOnDomain(networkName, nadName, guestIfaceName),
		),
	)
})

func dummyVMIWithoutStatus(networkName string, nadName string) *v1.VirtualMachineInstance {
	vmi := newVMI()
	vmi.Spec.Networks = []v1.Network{
		{
			Name:          networkName,
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: nadName}},
		}}
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
		{
			Name:                   networkName,
			InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
		}}
	return vmi
}

func dummyVMIWithOneNetworkAndOneIfaceOnSpecAndStatus(networkName string, nadName string) *v1.VirtualMachineInstance {
	dummyVMI := dummyVMIWithoutStatus(networkName, nadName)
	dummyVMI.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
		{
			Name:          networkName,
			InterfaceName: "eno123",
		},
	}
	return dummyVMI
}

func dummyVMIWithMultipleNetworksAndIfacesOnSpec(networkName string, nadName string) *v1.VirtualMachineInstance {
	dummyVMI := dummyVMIWithoutStatus(networkName, nadName)
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

func dummyVMIWithAttachmentToPlug(networkName string, netAttachDefName string, guestIfaceName string) *v1.VirtualMachineInstance {
	vmi := dummyVMIWithoutStatus(networkName, netAttachDefName)
	vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
		{Name: networkName, InterfaceName: guestIfaceName, InfoSource: vmispec.InfoSourceMultusStatus},
	}
	return vmi
}

func dummyVMIWithAttachmentAlreadyAvailableOnDomain(networkName string, netAttachDefName string, guestIfaceName string) *v1.VirtualMachineInstance {
	vmi := dummyVMIWithAttachmentToPlug(networkName, netAttachDefName, guestIfaceName)
	for i := range vmi.Status.Interfaces {
		vmi.Status.Interfaces[i].InfoSource = vmispec.NewInfoSource(vmispec.InfoSourceDomain, vmispec.InfoSourceMultusStatus)
	}
	return vmi
}

func dummyVMIWithStatusOnly(networkName string, ifaceName string) *v1.VirtualMachineInstance {
	vmi := newVMI()
	vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
		{
			Name:          networkName,
			InterfaceName: ifaceName,
			InfoSource:    vmispec.InfoSourceMultusStatus,
		},
	}
	return vmi
}

func newVMI() *v1.VirtualMachineInstance {
	const vmName = "pepe"
	vmi := v1.NewVMIReferenceFromNameWithNS("", vmName)
	vmi.Spec = v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}
	return vmi
}
