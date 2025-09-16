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

package domain_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/cmd/sidecars/network-passt-binding/domain"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	ifaceTypeVhostUser = "vhostuser"

	defaultPrimaryPodIfaceName = "eth0"
)

var _ = Describe("pod network configurator", func() {
	Context("generate domain spec interface", func() {
		DescribeTable("should fail to create configurator given",
			func(ifaces []vmschema.Interface, networks []vmschema.Network, ifaceStatuses []vmschema.VirtualMachineInstanceNetworkInterface) {
				_, err := domain.NewPasstNetworkConfigurator(
					ifaces,
					networks,
					ifaceStatuses,
					domain.NetworkConfiguratorOptions{},
				)

				Expect(err).To(HaveOccurred())
			},
			Entry("no pod network",
				nil,
				[]vmschema.Network{{Name: "default", NetworkSource: vmschema.NetworkSource{Multus: &vmschema.MultusNetwork{}}}},
				nil,
			),
			Entry("no corresponding iface",
				[]vmschema.Interface{{Name: "not-default", Binding: &vmschema.PluginBinding{Name: "passt"}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
				nil,
			),
			Entry("interface with no passt binding method",
				[]vmschema.Interface{{Name: "default", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
				[]vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}},
			),
			Entry("interface with no passt binding plugin",
				[]vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "no-passt"}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
				[]vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}},
			),
			Entry("Interface with no matching status entry",
				[]vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
				nil,
			),
			Entry("Interface with an empty PodInterfaceName",
				[]vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
				[]vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: ""}},
			),
		)

		It("should fail given interface with invalid PCI address", func() {
			ifaces := []vmschema.Interface{{
				Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"},
				PciAddress: "invalid-pci-address",
			}}
			networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
			ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}}

			testMutator, err := domain.NewPasstNetworkConfigurator(
				ifaces,
				networks,
				ifaceStatuses,
				domain.NetworkConfiguratorOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			_, err = testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("should add interface to domain spec given iface with",
			func(iface *vmschema.Interface, expectedDomainIface *domainschema.Interface) {
				ifaces := []vmschema.Interface{*iface}
				networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
				ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}}

				testMutator, err := domain.NewPasstNetworkConfigurator(
					ifaces,
					networks,
					ifaceStatuses,
					domain.NetworkConfiguratorOptions{},
				)
				Expect(err).ToNot(HaveOccurred())

				mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
			},
			Entry("passt binding plugin",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}},
				&domainschema.Interface{
					Alias:       domainschema.NewUserDefinedAlias("default"),
					Type:        ifaceTypeVhostUser,
					Source:      domainschema.InterfaceSource{Device: "eth0"},
					Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
					Model:       &domainschema.Model{Type: "virtio-non-transitional"},
				},
			),
			Entry("PCI address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"},
					PciAddress: "0000:02:02.0",
				},
				&domainschema.Interface{
					Alias:       domainschema.NewUserDefinedAlias("default"),
					Type:        ifaceTypeVhostUser,
					Source:      domainschema.InterfaceSource{Device: "eth0"},
					Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
					Model:       &domainschema.Model{Type: "virtio-non-transitional"},
					Address:     &domainschema.Address{Type: "pci", Domain: "0x0000", Bus: "0x02", Slot: "0x02", Function: "0x0"},
				},
			),
			Entry("MAC address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"},
					MacAddress: "02:02:02:02:02:02",
				},
				&domainschema.Interface{
					Alias:       domainschema.NewUserDefinedAlias("default"),
					Type:        ifaceTypeVhostUser,
					Source:      domainschema.InterfaceSource{Device: "eth0"},
					Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
					Model:       &domainschema.Model{Type: "virtio-non-transitional"},
					MAC:         &domainschema.MAC{MAC: "02:02:02:02:02:02"},
				},
			),
			Entry("ACPI address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"},
					ACPIIndex: 2,
				},
				&domainschema.Interface{
					Alias:       domainschema.NewUserDefinedAlias("default"),
					Type:        ifaceTypeVhostUser,
					Source:      domainschema.InterfaceSource{Device: "eth0"},
					Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
					Model:       &domainschema.Model{Type: "virtio-non-transitional"},
					ACPI:        &domainschema.ACPI{Index: uint(2)},
				},
			),
			Entry("non virtio model",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"},
					Model: "e1000",
				},
				&domainschema.Interface{
					Alias:       domainschema.NewUserDefinedAlias("default"),
					Type:        ifaceTypeVhostUser,
					Source:      domainschema.InterfaceSource{Device: "eth0"},
					Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
					Model:       &domainschema.Model{Type: "e1000"},
				},
			),
			Entry("tcp ports (should forward tcp ports only)",
				newInterface(
					"default",
					withPasstBindingPlugin(),
					withOpenPort("TCP", 1),
					withOpenPort("TCP", 4),
				),
				&domainschema.Interface{
					Alias:   domainschema.NewUserDefinedAlias("default"),
					Type:    ifaceTypeVhostUser,
					Source:  domainschema.InterfaceSource{Device: "eth0"},
					Backend: &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					Model:   &domainschema.Model{Type: "virtio-non-transitional"},
					PortForward: []domainschema.InterfacePortForward{
						{
							Proto: "tcp",
							Ranges: []domainschema.InterfacePortForwardRange{
								{Start: 1}, {Start: 4},
							},
						},
					},
				},
			),
			Entry("udp ports (should forward udp ports only)",
				newInterface(
					"default",
					withPasstBindingPlugin(),
					withOpenPort("UDP", 2),
					withOpenPort("UDP", 3),
				),
				&domainschema.Interface{
					Alias:   domainschema.NewUserDefinedAlias("default"),
					Type:    ifaceTypeVhostUser,
					Source:  domainschema.InterfaceSource{Device: "eth0"},
					Backend: &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					Model:   &domainschema.Model{Type: "virtio-non-transitional"},
					PortForward: []domainschema.InterfacePortForward{
						{
							Proto: "udp",
							Ranges: []domainschema.InterfacePortForwardRange{
								{Start: 2}, {Start: 3},
							},
						},
					},
				},
			),
			Entry("both tcp and udp ports",
				newInterface(
					"default",
					withPasstBindingPlugin(),
					withOpenPort("", 1),
					withOpenPort("UDP", 2),
					withOpenPort("UDP", 3),
					withOpenPort("TCP", 4),
				),
				&domainschema.Interface{
					Alias:   domainschema.NewUserDefinedAlias("default"),
					Type:    ifaceTypeVhostUser,
					Source:  domainschema.InterfaceSource{Device: "eth0"},
					Backend: &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					Model:   &domainschema.Model{Type: "virtio-non-transitional"},
					PortForward: []domainschema.InterfacePortForward{
						{
							Proto: "tcp",
							Ranges: []domainschema.InterfacePortForwardRange{
								{Start: 1}, {Start: 4},
							},
						},
						{
							Proto: "udp",
							Ranges: []domainschema.InterfacePortForwardRange{
								{Start: 2}, {Start: 3},
							},
						},
					},
				},
			),
		)

		DescribeTable("should add interface to domain spec given iface given the option",
			func(opts *domain.NetworkConfiguratorOptions, expectedDomainIface *domainschema.Interface) {
				ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}}}
				networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
				ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}}

				testMutator, err := domain.NewPasstNetworkConfigurator(ifaces, networks, ifaceStatuses, *opts)
				Expect(err).ToNot(HaveOccurred())

				mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
			},
			Entry("virtio transitional enabled",
				&domain.NetworkConfiguratorOptions{UseVirtioTransitional: true},
				&domainschema.Interface{
					Alias:       domainschema.NewUserDefinedAlias("default"),
					Type:        ifaceTypeVhostUser,
					Source:      domainschema.InterfaceSource{Device: "eth0"},
					Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
					Model:       &domainschema.Model{Type: "virtio-transitional"},
				},
			),
			Entry("isitio proxy injection enabled",
				&domain.NetworkConfiguratorOptions{IstioProxyInjectionEnabled: true},
				&domainschema.Interface{
					Alias:   domainschema.NewUserDefinedAlias("default"),
					Type:    ifaceTypeVhostUser,
					Source:  domainschema.InterfaceSource{Device: "eth0"},
					Backend: &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
					Model:   &domainschema.Model{Type: "virtio-non-transitional"},
					PortForward: []domainschema.InterfacePortForward{
						{Proto: "tcp", Ranges: []domainschema.InterfacePortForwardRange{
							{Start: 15000, Exclude: "yes"},
							{Start: 15001, Exclude: "yes"},
							{Start: 15004, Exclude: "yes"},
							{Start: 15006, Exclude: "yes"},
							{Start: 15008, Exclude: "yes"},
							{Start: 15009, Exclude: "yes"},
							{Start: 15020, Exclude: "yes"},
							{Start: 15021, Exclude: "yes"},
							{Start: 15053, Exclude: "yes"},
							{Start: 15090, Exclude: "yes"},
						}},
					},
				},
			),
		)

		It("should not override other interfaces", func() {
			networks := []vmschema.Network{
				*vmschema.DefaultPodNetwork(),
				{Name: "secondary", NetworkSource: vmschema.NetworkSource{Multus: &vmschema.MultusNetwork{NetworkName: "sec"}}},
			}
			ifaces := []vmschema.Interface{
				{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}},
				{Name: "secondary", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}},
			}

			ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}}

			expectedDomainIface := &domainschema.Interface{
				Alias:       domainschema.NewUserDefinedAlias("default"),
				Type:        ifaceTypeVhostUser,
				Source:      domainschema.InterfaceSource{Device: "eth0"},
				Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
				PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
				Model:       &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewPasstNetworkConfigurator(
				ifaces,
				networks,
				ifaceStatuses,
				domain.NetworkConfiguratorOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			existingIface := &domainschema.Interface{Alias: domainschema.NewUserDefinedAlias("existing-iface")}
			testDomSpec := &domainschema.DomainSpec{
				Devices: domainschema.Devices{
					Interfaces: []domainschema.Interface{*existingIface},
				},
			}

			mutatedDomSpec, err := testMutator.Mutate(testDomSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*existingIface, *expectedDomainIface}))
		})

		It("should set domain interface correctly when executed more than once", func() {
			networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}}}
			ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}}

			expectedDomainIface := &domainschema.Interface{
				Alias:       domainschema.NewUserDefinedAlias("default"),
				Type:        ifaceTypeVhostUser,
				Source:      domainschema.InterfaceSource{Device: "eth0"},
				Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
				PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
				Model:       &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewPasstNetworkConfigurator(
				ifaces,
				networks,
				ifaceStatuses,
				domain.NetworkConfiguratorOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			testDomSpec := &domainschema.DomainSpec{}

			mutatedDomSpec, err := testMutator.Mutate(testDomSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))

			Expect(testMutator.Mutate(mutatedDomSpec)).To(Equal(mutatedDomSpec))
		})
		It("should set domain interface source link to the optional one if exists", func() {
			networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}}}
			ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: "ovn-udn1"}}

			expectedDomainIface := &domainschema.Interface{
				Alias:       domainschema.NewUserDefinedAlias("default"),
				Type:        ifaceTypeVhostUser,
				Source:      domainschema.InterfaceSource{Device: "ovn-udn1"},
				Backend:     &domainschema.InterfaceBackend{Type: "passt", LogFile: domain.PasstLogFilePath},
				PortForward: []domainschema.InterfacePortForward{{Proto: "tcp"}, {Proto: "udp"}},
				Model:       &domainschema.Model{Type: "virtio-non-transitional"},
			}
			testMutator, err := domain.NewPasstNetworkConfigurator(
				ifaces,
				networks,
				ifaceStatuses,
				domain.NetworkConfiguratorOptions{},
			)
			Expect(err).ToNot(HaveOccurred())

			testDomSpec := &domainschema.DomainSpec{}

			mutatedDomSpec, err := testMutator.Mutate(testDomSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))

			Expect(testMutator.Mutate(mutatedDomSpec)).To(Equal(mutatedDomSpec))
		})
	})
	Context("should define memoryBacking for vhost-user", func() {
		var testMutator *domain.PasstNetworkConfigurator
		BeforeEach(func() {
			networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "passt"}}}
			ifaceStatuses := []vmschema.VirtualMachineInstanceNetworkInterface{{Name: "default", PodInterfaceName: defaultPrimaryPodIfaceName}}

			var err error
			testMutator, err = domain.NewPasstNetworkConfigurator(
				ifaces,
				networks,
				ifaceStatuses,
				domain.NetworkConfiguratorOptions{},
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should set shared memfd when clean domain", func() {
			expectedMemoryBacking := &domainschema.MemoryBacking{
				Access: &domainschema.MemoryBackingAccess{
					Mode: "shared",
				},
				Source: &domainschema.MemoryBackingSource{
					Type: "memfd",
				},
			}
			mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.MemoryBacking).To(Equal(expectedMemoryBacking))
		})

		It("should fail when private memory is predefined", func() {
			domainWithPrivateMem := &domainschema.DomainSpec{
				MemoryBacking: &domainschema.MemoryBacking{
					Access: &domainschema.MemoryBackingAccess{Mode: "private"},
				},
			}
			_, err := testMutator.Mutate(domainWithPrivateMem)
			Expect(err).To(HaveOccurred())
		})

		It("should use other configs of backing memory as long as they are shared", func() {
			domainWithOtherSharedMem := &domainschema.DomainSpec{
				MemoryBacking: &domainschema.MemoryBacking{
					Access: &domainschema.MemoryBackingAccess{Mode: "shared"},
					Source: &domainschema.MemoryBackingSource{Type: "file"},
				},
			}
			mutatedDomSpec, err := testMutator.Mutate(domainWithOtherSharedMem)
			Expect(err).NotTo(HaveOccurred())
			Expect(mutatedDomSpec.MemoryBacking).To(Equal(domainWithOtherSharedMem.MemoryBacking))
		})
	})
})

type option func(p *vmschema.Interface)

func newInterface(name string, options ...option) *vmschema.Interface {
	newIface := &vmschema.Interface{
		Name: name,
	}

	for _, option := range options {
		option(newIface)
	}

	return newIface
}

func withPasstBindingPlugin() option {
	return func(iface *vmschema.Interface) {
		iface.Binding = &vmschema.PluginBinding{
			Name: "passt",
		}
	}
}

func withOpenPort(protocol string, portNumber int32) option {
	return func(iface *vmschema.Interface) {
		iface.Ports = append(iface.Ports, vmschema.Port{
			Protocol: protocol,
			Port:     portNumber,
		})
	}
}
