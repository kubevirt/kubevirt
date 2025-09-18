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

package network

import (
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

const defaultNet = "default"

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

	It("hotplugVirtioInterface SUCCEEDS with link state down", func() {
		networkInterfaceManager := newVirtIOInterfaceManager(
			expectAttachDeviceLinkStateDown(gomock.NewController(GinkgoT())).VirtDomain,
			&fakeVMConfigurator{},
		)

		vmi := libvmi.New(
			libvmi.WithInterface(v1.Interface{
				Name:  networkName,
				State: v1.InterfaceStateLinkDown,
			}),
			libvmi.WithNetwork(&v1.Network{Name: networkName}),
			libvmistatus.WithStatus(
				libvmistatus.New(libvmistatus.WithInterfaceStatus(
					v1.VirtualMachineInstanceNetworkInterface{
						Name:       networkName,
						InfoSource: vmispec.InfoSourceMultusStatus,
					},
				)),
			),
		)
		Expect(networkInterfaceManager.hotplugVirtioInterface(
			vmi,
			dummyDomain(),
			newDomain(newDeviceInterface(networkName, libvirtInterfaceLinkStateDown)),
		)).To(Succeed())
	})

	DescribeTable(
		"hotplugVirtioInterface SUCCEEDS for",
		func(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain, updatedDomain *api.Domain, result libvirtClientResult) {
			networkInterfaceManager := newVirtIOInterfaceManager(
				mockLibvirtClient(gomock.NewController(GinkgoT()), result).VirtDomain,
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
				mockLibvirtClient(gomock.NewController(GinkgoT()), result).VirtDomain,
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
		func(vmiSpecIfaces []v1.Interface, vmiSpecNets []v1.Network, domainSpecIfaces []api.Interface, expectedDomainSpecIfaces []api.Interface) {
			Expect(interfacesToHotUnplug(vmiSpecIfaces, vmiSpecNets, domainSpecIfaces)).To(ConsistOf(expectedDomainSpecIfaces))
		},
		Entry("given no VMI interfaces and no domain interfaces", nil, nil, nil, nil),
		Entry("given no VMI interfaces and 1 domain interface",
			nil,
			nil,
			[]api.Interface{{Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("given 1 VMI non-absent interface and an associated interface in the domain",
			[]v1.Interface{{Name: networkName}},
			[]v1.Network{{Name: networkName, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}},
			[]api.Interface{{Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("given 1 VMI absent interface and an associated interface in the domain is using ordinal device",
			[]v1.Interface{{Name: networkName, State: v1.InterfaceStateAbsent, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}},
			[]v1.Network{{Name: networkName, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}},
			[]api.Interface{
				{Target: &api.InterfaceTarget{Device: ordinalDevice}, Alias: api.NewUserDefinedAlias(networkName)},
			},
			nil,
		),
		Entry("given 1 VMI absent interface and an associated interface in the domain is using hashed device",
			[]v1.Interface{{Name: networkName, State: v1.InterfaceStateAbsent, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}},
			[]v1.Network{{Name: networkName, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}},
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

	It("are ignored when 0 count is specified", func() {
		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{}}
		domainSpec := &api.DomainSpec{}
		countCalls := 0
		_, _ = WithNetworkIfacesResources(vmi, domainSpec, 0, func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error) {
			countCalls++
			return nil, nil
		})
		// The counter tracks the tested function behavior.
		// It is expected that the callback function is called only once when there is no need
		// to add placeholders interfaces.
		Expect(countCalls).To(Equal(1))
	})

	It("are reserved when the default reserved interfaces count is 3", func() {
		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{}}
		domainSpec := &api.DomainSpec{}
		for range vmi.Spec.Domain.Devices.Interfaces {
			domainSpec.Devices.Interfaces = append(domainSpec.Devices.Interfaces, api.Interface{})
		}

		ctrl := gomock.NewController(GinkgoT())
		mockLibvirt := testing.NewLibvirt(ctrl)
		domxml, err := xml.MarshalIndent(domainSpec, "", "\t")
		Expect(err).ToNot(HaveOccurred())
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE).Return(string(domxml), nil)
		mockLibvirt.DomainEXPECT().Free()

		originalDomainSpec := domainSpec.DeepCopy()
		countCalls := 0
		_, err = WithNetworkIfacesResources(vmi, domainSpec, 3, func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error) {
			// Tracking the behavior of the tested function.
			// It is expected that the callback function is called twice when placeholders are needed.
			// The first time it is called with the placeholders in place.
			// The second time it is called without the placeholders.
			countCalls++
			if countCalls == 1 {
				Expect(s.Devices.Interfaces).To(HaveLen(4))
			} else {
				Expect(s.Devices.Interfaces).To(Equal(originalDomainSpec.Devices.Interfaces))
			}

			return mockLibvirt.VirtDomain, nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(countCalls).To(Equal(2))
		Expect(domainSpec.Devices.Interfaces).To(Equal(originalDomainSpec.Devices.Interfaces))
	})
})

var _ = Describe("interface link state update", func() {
	DescribeTable("no change in state",
		func(domainFrom *api.Domain,
			domainTo *api.Domain,
			expectMockFunc func(*gomock.Controller) *testing.Libvirt) {

			networkInterfaceManager := newVirtIOInterfaceManager(
				expectMockFunc(gomock.NewController(GinkgoT())).VirtDomain,
				&fakeVMConfigurator{})
			Expect(networkInterfaceManager.updateDomainLinkState(domainFrom, domainTo)).To(Succeed())
		},

		Entry("none to none",
			dummyDomain(defaultNet),
			dummyDomain(defaultNet),
			expectUpdateDeviceNotCalled,
		),
		Entry("down to down",
			newDomain(newDeviceInterface(defaultNet, libvirtInterfaceLinkStateDown)),
			newDomain(newDeviceInterface(defaultNet, libvirtInterfaceLinkStateDown)),
			expectUpdateDeviceNotCalled,
		),
		Entry("down to none",
			newDomain(newDeviceInterface(defaultNet, libvirtInterfaceLinkStateDown)),
			dummyDomain(defaultNet),
			expectUpdateDeviceLinkStateNone,
		),
		Entry("none to down",
			dummyDomain(defaultNet),
			newDomain(newDeviceInterface(defaultNet, libvirtInterfaceLinkStateDown)),
			expectUpdateDeviceLinkStateDown,
		),
	)
})

type libvirtClientResult struct {
	expectedError           error
	expectedAttachedDevices int
}

func mockLibvirtClient(mockController *gomock.Controller, clientResult libvirtClientResult) *testing.Libvirt {
	mockClient := testing.NewLibvirt(mockController)
	if clientResult.expectedError != nil {
		mockClient.DomainEXPECT().AttachDeviceFlags(gomock.Any(), gomock.Any()).Return(clientResult.expectedError)
		return mockClient
	}
	mockClient.DomainEXPECT().AttachDeviceFlags(gomock.Any(), gomock.Any()).Times(clientResult.expectedAttachedDevices).Return(nil)
	return mockClient
}

func expectAttachDeviceLinkStateDown(mockController *gomock.Controller) *testing.Libvirt {
	const interfaceWithLinkStateDownXML = `<interface type=""><source></source><link state="down"></link><alias name="ua-n1"></alias></interface>`
	mockClient := testing.NewLibvirt(mockController)
	mockClient.DomainEXPECT().AttachDeviceFlags(interfaceWithLinkStateDownXML, gomock.Any()).Times(1).Return(nil)

	return mockClient
}

func expectUpdateDeviceNotCalled(mockController *gomock.Controller) *testing.Libvirt {
	mockClient := testing.NewLibvirt(mockController)
	mockClient.DomainEXPECT().UpdateDeviceFlags(gomock.Any(), gomock.Any()).Times(0).Return(nil)

	return mockClient
}

func expectUpdateDeviceLinkStateDown(mockController *gomock.Controller) *testing.Libvirt {
	mockClient := testing.NewLibvirt(mockController)

	const interfaceWithLinkStateDownXML = `<interface type=""><source></source><link state="down"></link><alias name="ua-default"></alias></interface>`
	mockClient.DomainEXPECT().UpdateDeviceFlags(interfaceWithLinkStateDownXML, gomock.Any()).Times(1).Return(nil)

	return mockClient
}

func expectUpdateDeviceLinkStateNone(mockController *gomock.Controller) *testing.Libvirt {
	mockClient := testing.NewLibvirt(mockController)

	const interfaceWithoutLinkStateXML = `<interface type=""><source></source><alias name="ua-default"></alias></interface>`
	mockClient.DomainEXPECT().UpdateDeviceFlags(interfaceWithoutLinkStateXML, gomock.Any()).Times(1).Return(nil)
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

func newDomain(netInterfaces ...api.Interface) *api.Domain {
	return &api.Domain{
		Spec: api.DomainSpec{
			Devices: api.Devices{Interfaces: netInterfaces},
		},
	}
}

func newDeviceInterface(ifaceName, state string) api.Interface {
	return api.Interface{
		Alias:     api.NewUserDefinedAlias(ifaceName),
		LinkState: &api.LinkState{State: state},
	}
}
