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
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
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

	DescribeTable(
		"hotplugVirtioInterface SUCCEEDS for",
		func(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain, updatedDomain *api.Domain, result libvirtClientResult) {
			networkInterfaceManager := newVirtIOInterfaceManager(
				mockLibvirtClient(gomock.NewController(GinkgoT()), result),
				&fakeVMConfigurator{},
			)
			Expect(networkInterfaceManager.hotplugVirtioInterface(vmi, currentDomain, updatedDomain)).To(Succeed())
		},
		Entry(
			"VMI without networks, whose domain also doesn't have interfaces, does **not** attach any device",
			&v1.VirtualMachineInstance{Spec: v1.VirtualMachineInstanceSpec{Networks: []v1.Network{}}},
			dummyDomain(),
			dummyDomain(),
			libvirtClientResult{expectedAttachedDevices: 0},
		),
		Entry("VMI with 1 network (with the pod interface ready), not present in the domain",
			vmiWithSingleBridgeInterfaceWithPodInterfaceReady(networkName, nadName),
			dummyDomain(),
			dummyDomain(networkName),
			libvirtClientResult{expectedAttachedDevices: 1},
		),
	)

	DescribeTable(
		"hotplugVirtioInterface FAILS when",
		func(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain, updatedDomain *api.Domain, configurator vmConfigurator, result libvirtClientResult) {
			networkInterfaceManager := newVirtIOInterfaceManager(
				mockLibvirtClient(gomock.NewController(GinkgoT()), result),
				configurator,
			)
			Expect(networkInterfaceManager.hotplugVirtioInterface(vmi, currentDomain, updatedDomain)).To(MatchError("boom"))
		},
		Entry("the VM network configurator ERRORs invoking setup networking phase#2",
			vmiWithSingleBridgeInterfaceWithPodInterfaceReady(networkName, nadName),
			dummyDomain(),
			dummyDomain(),
			&fakeVMConfigurator{expectedError: fmt.Errorf("boom")},
			libvirtClientResult{},
		),
		Entry("the VM network configurator ERRORs invoking libvirt's attach device",
			vmiWithSingleBridgeInterfaceWithPodInterfaceReady(networkName, nadName),
			dummyDomain(),
			dummyDomain(networkName),
			&fakeVMConfigurator{},
			libvirtClientResult{expectedError: fmt.Errorf("boom")},
		),
	)
})

type libvirtClientResult struct {
	expectedError           error
	expectedAttachedDevices int
}

func mockLibvirtClient(mockController *gomock.Controller, clientResult libvirtClientResult) *cli.MockVirDomain {
	mockClient := cli.NewMockVirDomain(mockController)
	if clientResult.expectedError != nil {
		mockClient.EXPECT().AttachDeviceFlags(gomock.Any(), gomock.Any()).Return(clientResult.expectedError)
		return mockClient
	}
	mockClient.EXPECT().AttachDeviceFlags(gomock.Any(), gomock.Any()).Times(clientResult.expectedAttachedDevices).Return(nil)
	return mockClient
}

func vmiWithSingleBridgeInterfaceWithPodInterfaceReady(ifaceName string, nadName string) *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		Spec: v1.VirtualMachineInstanceSpec{
			Networks: []v1.Network{generateNetwork(ifaceName, nadName)},
			Domain: v1.DomainSpec{
				Devices: v1.Devices{
					Interfaces: []v1.Interface{{
						Name: ifaceName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{},
						},
					}},
				},
			},
		},
		Status: v1.VirtualMachineInstanceStatus{
			Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
				Name:       ifaceName,
				InfoSource: vmispec.InfoSourceMultusStatus,
			}},
		},
	}
}

func generateNetwork(name string, nadName string) v1.Network {
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{NetworkName: nadName}},
	}
}

func dummyDomain(ifaceNames ...string) *api.Domain {
	var ifaces []api.Interface
	for _, ifaceName := range ifaceNames {
		ifaces = append(ifaces, api.Interface{Alias: api.NewUserDefinedAlias(ifaceName)})
	}
	return &api.Domain{
		Spec: api.DomainSpec{
			Devices: api.Devices{
				Interfaces: ifaces,
			},
		},
	}
}

type fakeVMConfigurator struct {
	expectedError error
}

func (fvc *fakeVMConfigurator) SetupPodNetworkPhase2(*api.Domain, []v1.Network) error {
	return fvc.expectedError
}
