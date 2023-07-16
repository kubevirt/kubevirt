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
	"encoding/xml"
	"fmt"

	"kubevirt.io/kubevirt/pkg/network/namescheme"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirt"

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
		Entry("vmi with 1 network marked for removal, pod interface ready and no interfaces in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Networks: []v1.Network{generateNetwork(networkName, nadName)},
					Domain: v1.DomainSpec{Devices: v1.Devices{Interfaces: []v1.Interface{{
						Name:                   networkName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
						State:                  v1.InterfaceStateAbsent,
					}}}},
				},
				Status: v1.VirtualMachineInstanceStatus{
					Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
						Name:       networkName,
						InfoSource: vmispec.InfoSourceMultusStatus,
					}},
				},
			},
			map[string]api.Interface{},
			nil,
		),
		Entry("vmi with 1 network (when the pod interface *is* ready), but no interfaces in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Networks: []v1.Network{generateNetwork(networkName, nadName)},
					Domain: v1.DomainSpec{Devices: v1.Devices{Interfaces: []v1.Interface{{
						Name:                   networkName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
					}}}},
				},
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
		Entry("vmi with 1 SR-IOV network (when the pod interface is ready) and no interfaces in the domain",
			&v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Networks: []v1.Network{generateNetwork(networkName, nadName)},
					Domain: v1.DomainSpec{Devices: v1.Devices{Interfaces: []v1.Interface{{
						Name:                   networkName,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
					}}}},
				},
				Status: v1.VirtualMachineInstanceStatus{
					Interfaces: []v1.VirtualMachineInstanceNetworkInterface{{
						Name:       networkName,
						InfoSource: vmispec.InfoSourceMultusStatus,
					}},
				},
			},
			map[string]api.Interface{},
			[]v1.Network{},
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

var _ = Describe("nic hot-unplug on virt-launcher", func() {
	const (
		networkName   = "n1"
		ordinalDevice = "tap2"

		sriovNetworkName = "n2-sriov"
	)

	hashedDevice := "tap" + namescheme.GenerateHashedInterfaceName(networkName)[3:]

	DescribeTable("domain interfaces to hot-unplug",
		func(vmiSpecIfaces []v1.Interface, domainSpecIfaces []api.Interface, expectedDomainSpecIfaces []api.Interface) {
			Expect(interfacesToHotUnplug(vmiSpecIfaces, domainSpecIfaces)).To(ConsistOf(expectedDomainSpecIfaces))
		},
		Entry("given no VMI interfaces and no domain interfaces", nil, nil, nil),
		Entry("given no VMI interfaces and 1 domain interface",
			nil,
			[]api.Interface{{Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("given 1 VMI non-absent interface and an associated interface in the domain",
			[]v1.Interface{{Name: networkName}},
			[]api.Interface{{Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("given 1 VMI absent interface and an associated interface in the domain is using ordinal device",
			[]v1.Interface{{Name: networkName, State: v1.InterfaceStateAbsent, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}},
			[]api.Interface{
				{Target: &api.InterfaceTarget{Device: ordinalDevice}, Alias: api.NewUserDefinedAlias(networkName)},
			},
			nil,
		),
		Entry("given 1 VMI absent interface and an associated interface in the domain is using hashed device",
			[]v1.Interface{{Name: networkName, State: v1.InterfaceStateAbsent, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}},
			[]api.Interface{{
				Target: &api.InterfaceTarget{Device: hashedDevice}, Alias: api.NewUserDefinedAlias(networkName)},
			},
			[]api.Interface{
				{Target: &api.InterfaceTarget{Device: hashedDevice}, Alias: api.NewUserDefinedAlias(networkName)},
			},
		),
	)
})

var _ = Describe("domain network interfaces resources", func() {

	DescribeTable("are ignored when",
		func(vmiSpecIfaces []v1.Interface) {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Spec.Domain.Devices.Interfaces = vmiSpecIfaces
			domainSpec := &api.DomainSpec{}
			countCalls := 0
			_, _ = withNetworkIfacesResources(vmi, domainSpec, func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error) {
				countCalls++
				return nil, nil
			})
			// The counter tracks the tested function behavior.
			// It is expected that the callback function is called only once when there is no need
			// to add placeholders interfaces.
			Expect(countCalls).To(Equal(1))
		},
		Entry("no defined interfaces exist", nil),
		Entry("the reserved interfaces count is less than defined interfaces", []v1.Interface{{}, {}, {}, {}, {}}),
		Entry("the interface resource-request is equal to the defined interfaces", []v1.Interface{{}, {}, {}, {}}),
	)

	It("are ignored when placePCIDevicesOnRootComplex annotation is used on the VMI", func() {
		vmi := &v1.VirtualMachineInstance{}
		vmi.Annotations = map[string]string{v1.PlacePCIDevicesOnRootComplex: "true"}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{}}
		domainSpec := &api.DomainSpec{}
		countCalls := 0
		_, _ = withNetworkIfacesResources(vmi, domainSpec, func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error) {
			countCalls++
			return nil, nil
		})
		// The counter tracks the tested function behavior.
		// It is expected that the callback function is called only once when there is no need
		// to add placeholders interfaces.
		Expect(countCalls).To(Equal(1))
	})

	It("are reserved when the default reserved interfaces count is larger than the defined interfaces", func() {
		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{}}
		domainSpec := &api.DomainSpec{}
		for range vmi.Spec.Domain.Devices.Interfaces {
			domainSpec.Devices.Interfaces = append(domainSpec.Devices.Interfaces, api.Interface{})
		}

		ctrl := gomock.NewController(GinkgoT())
		mockDomain := cli.NewMockVirDomain(ctrl)
		domxml, err := xml.MarshalIndent(domainSpec, "", "\t")
		Expect(err).ToNot(HaveOccurred())
		mockDomain.EXPECT().GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE).Return(string(domxml), nil)

		originalDomainSpec := domainSpec.DeepCopy()
		countCalls := 0
		_, err = withNetworkIfacesResources(vmi, domainSpec, func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error) {
			// Tracking the behavior of the tested function.
			// It is expected that the callback function is called twice when placeholders are needed.
			// The first time it is called with the placeholders in place.
			// The second time it is called without the placeholders.
			countCalls++
			if countCalls == 1 {
				Expect(s.Devices.Interfaces).To(HaveLen(ReservedInterfaces))
			} else {
				Expect(s.Devices.Interfaces).To(Equal(originalDomainSpec.Devices.Interfaces))
			}

			return mockDomain, nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(countCalls).To(Equal(2))
		Expect(domainSpec.Devices.Interfaces).To(Equal(originalDomainSpec.Devices.Interfaces))
	})
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
