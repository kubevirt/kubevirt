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

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/network"
)

var _ = Describe("Network Domain Configurator", func() {
	const (
		network1Name = "test-network1"
		nad1Name     = "test-nad1"

		tapBasedBindingPluginName = "tap-based-binding"
	)

	const (
		sockets = 2
		cores   = 4
		threads = 2

		expectedQueueCountForVirtio = uint(sockets * cores * threads)
	)

	const virtioModel = "virtio-non-transitional"

	DescribeTable("Should not configure interfaces",
		func(
			vmi *v1.VirtualMachineInstance,
			domainAttachmentByInterfaceName map[string]string,
		) {
			configurator := network.NewDomainConfigurator(
				network.WithDomainAttachmentByInterfaceName(domainAttachmentByInterfaceName),
				network.WithUseLaunchSecuritySEV(false),
				network.WithUseLaunchSecurityPV(false),
				network.WithROMTuningSupport(false),
				network.WithVirtioModel(virtioModel),
			)

			var domain api.Domain
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())
			Expect(domain).To(Equal(api.Domain{}))
		},
		Entry("when no interfaces are specified", libvmi.New(), nil),
		Entry(
			"when only an SR-IOV interface is specified",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(network1Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, nad1Name)),
			),
			nil,
		),
		Entry(
			"when an interface using a non-tap binding plugin is specified",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceWithPasstBindingPlugin()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			nil,
		),
		Entry(
			"when an absent interface is specified",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: network1Name, State: v1.InterfaceStateAbsent}),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, nad1Name)),
			),
			map[string]string{network1Name: string(v1.Tap)},
		),
	)

	DescribeTable("Should configure interfaces with tap-based binding",
		func(vmi *v1.VirtualMachineInstance) {
			networkName := vmi.Spec.Domain.Devices.Interfaces[0].Name
			domainAttachmentByInterfaceName := map[string]string{
				networkName: string(v1.Tap),
			}

			configurator := network.NewDomainConfigurator(
				network.WithDomainAttachmentByInterfaceName(domainAttachmentByInterfaceName),
				network.WithUseLaunchSecuritySEV(false),
				network.WithUseLaunchSecurityPV(false),
				network.WithROMTuningSupport(false),
				network.WithVirtioModel(virtioModel),
			)

			var domain api.Domain
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithIfaces(
				[]api.Interface{
					newDomainInterface(networkName, virtioModel, withTypeEthernet()),
				},
			)
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry(
			"when a primary interface is specified",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
		Entry(
			"when a secondary interface using bridge binding is specified",
			libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(network1Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, nad1Name)),
			),
		),
		Entry(
			"when an interface using a tap based binding plugin is specified",
			libvmi.New(
				libvmi.WithInterface(
					libvmi.InterfaceWithBindingPlugin(
						network1Name,
						v1.PluginBinding{Name: tapBasedBindingPluginName},
					),
				),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, nad1Name)),
			),
		),
	)

	DescribeTable("should configure link state",
		func(linkState v1.InterfaceState, expectedInterface api.Interface) {
			ifaceWithLinkState := libvmi.InterfaceDeviceWithBridgeBinding(network1Name)
			ifaceWithLinkState.State = linkState

			vmi := libvmi.New(
				libvmi.WithInterface(ifaceWithLinkState),
				libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, nad1Name)),
			)

			configurator := network.NewDomainConfigurator(
				network.WithDomainAttachmentByInterfaceName(map[string]string{network1Name: string(v1.Tap)}),
				network.WithUseLaunchSecuritySEV(false),
				network.WithUseLaunchSecurityPV(false),
				network.WithROMTuningSupport(false),
				network.WithVirtioModel(virtioModel),
			)

			var domain api.Domain
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := newDomainWithIfaces([]api.Interface{expectedInterface})
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry(
			"when link state spec is up",
			v1.InterfaceStateLinkUp,
			newDomainInterface(network1Name, virtioModel, withTypeEthernet()),
		),
		Entry(
			"when link state spec is down",
			v1.InterfaceStateLinkDown,
			newDomainInterface(network1Name, virtioModel, withTypeEthernet(), withLinkState("down")),
		),
	)

	DescribeTable("multi-queue", func(model string, expectedInterface api.Interface) {
		ifaceWithModel := libvmi.InterfaceDeviceWithBridgeBinding(network1Name)
		ifaceWithModel.Model = model

		vmi := libvmi.New(
			libvmi.WithCPUCount(cores, threads, sockets),
			libvmi.WithNetworkInterfaceMultiQueue(true),
			libvmi.WithInterface(ifaceWithModel),
			libvmi.WithNetwork(libvmi.MultusNetwork(network1Name, nad1Name)),
		)

		configurator := network.NewDomainConfigurator(
			network.WithDomainAttachmentByInterfaceName(map[string]string{network1Name: string(v1.Tap)}),
			network.WithUseLaunchSecuritySEV(false),
			network.WithUseLaunchSecurityPV(false),
			network.WithROMTuningSupport(false),
			network.WithVirtioModel(virtioModel),
		)

		var domain api.Domain
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := newDomainWithIfaces([]api.Interface{expectedInterface})
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry(
			"should be configured when model is empty (implicit virtio)",
			"",
			newDomainInterface(network1Name, virtioModel, withTypeEthernet(), withVHostDriver(expectedQueueCountForVirtio)),
		),
		Entry(
			"should be configured when model is virtio",
			v1.VirtIO,
			newDomainInterface(network1Name, virtioModel, withTypeEthernet(), withVHostDriver(expectedQueueCountForVirtio)),
		),
		Entry(
			"should not be configured when model is non-virtio",
			"e1000",
			newDomainInterface(network1Name, "e1000", withTypeEthernet()),
		),
	)
})

func newDomainWithIfaces(interfaces []api.Interface) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			Devices: api.Devices{
				Interfaces: interfaces,
			},
		},
	}
}

type option func(iface *api.Interface)

func newDomainInterface(networkName, modelType string, options ...option) api.Interface {
	newIface := api.Interface{
		Alias: api.NewUserDefinedAlias(networkName),
		Model: &api.Model{Type: modelType},
	}

	for _, f := range options {
		f(&newIface)
	}

	return newIface
}

func withTypeEthernet() option {
	return func(iface *api.Interface) {
		iface.Type = "ethernet"
	}
}

func withVHostDriver(queues uint) option {
	return func(iface *api.Interface) {
		iface.Driver = &api.InterfaceDriver{Name: "vhost", Queues: pointer.P(queues)}
	}
}

func withLinkState(state string) option {
	return func(iface *api.Interface) {
		iface.LinkState = &api.LinkState{State: state}
	}
}
