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
	"kubevirt.io/kubevirt/cmd/sidecars/network-bridge-binding/domain"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("pod network configurator", func() {
	Context("generate domain spec interface", func() {
		DescribeTable("should fail to create configurator given",
			func(ifaces []vmschema.Interface, networks []vmschema.Network) {
				_, err := domain.NewBridgeNetworkConfigurator(ifaces, networks, domain.NetworkConfiguratorOptions{})

				Expect(err).To(HaveOccurred())
			},
			Entry("no pod network",
				nil,
				[]vmschema.Network{{Name: "default", NetworkSource: vmschema.NetworkSource{Multus: &vmschema.MultusNetwork{}}}},
			),
			Entry("no corresponding iface",
				[]vmschema.Interface{{Name: "not-default", Binding: &vmschema.PluginBinding{Name: "bridge"}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
			),
			Entry("interface with no passt binding method",
				[]vmschema.Interface{{Name: "default", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
			),
			Entry("interface with no bridge binding plugin",
				[]vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "no-passt"}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
			),
		)

		It("should fail given interface with invalid PCI address", func() {
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"},
				PciAddress: "invalid-pci-address"}}
			networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}

			testMutator, err := domain.NewBridgeNetworkConfigurator(ifaces, networks, domain.NetworkConfiguratorOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("should add interface to domain spec given iface with",
			func(iface *vmschema.Interface, expectedDomainIface *domainschema.Interface) {
				ifaces := []vmschema.Interface{*iface}
				networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}

				testMutator, err := domain.NewBridgeNetworkConfigurator(ifaces, networks, domain.NetworkConfiguratorOptions{})
				Expect(err).ToNot(HaveOccurred())

				mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
			},
			Entry("bridge binding plugin",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"}},
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   "ethernet",
					Rom:    &domainschema.Rom{Enabled: "no"},
					MTU:    &domainschema.MTU{Size: "1480"},
					Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
				},
			),
			Entry("PCI address",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"},
					PciAddress: "0000:02:02.0"},
				&domainschema.Interface{
					Alias:   domainschema.NewUserDefinedAlias("default"),
					Type:    "ethernet",
					Rom:     &domainschema.Rom{Enabled: "no"},
					MTU:     &domainschema.MTU{Size: "1480"},
					Target:  &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:   &domainschema.Model{Type: "virtio-non-transitional"},
					Address: &domainschema.Address{Type: "pci", Domain: "0x0000", Bus: "0x02", Slot: "0x02", Function: "0x0"},
				},
			),
			Entry("MAC address",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"},
					MacAddress: "02:02:02:02:02:02"},
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   "ethernet",
					Rom:    &domainschema.Rom{Enabled: "no"},
					MTU:    &domainschema.MTU{Size: "1480"},
					Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
					MAC:    &domainschema.MAC{MAC: "02:02:02:02:02:02"},
				},
			),
			Entry("ACPI address",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"},
					ACPIIndex: 2},
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   "ethernet",
					Rom:    &domainschema.Rom{Enabled: "no"},
					MTU:    &domainschema.MTU{Size: "1480"},
					Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
					ACPI:   &domainschema.ACPI{Index: uint(2)},
				},
			),
			Entry("non virtio model",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"},
					Model: "e1000",
				},
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   "ethernet",
					Rom:    &domainschema.Rom{Enabled: "no"},
					MTU:    &domainschema.MTU{Size: "1480"},
					Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:  &domainschema.Model{Type: "e1000"},
				},
			),
		)

		DescribeTable("should add interface to domain spec given iface given the option",
			func(opts *domain.NetworkConfiguratorOptions, expectedDomainIface *domainschema.Interface) {
				ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"}}}
				networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}

				testMutator, err := domain.NewBridgeNetworkConfigurator(ifaces, networks, *opts)
				Expect(err).ToNot(HaveOccurred())

				mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
			},
			Entry("virtio transitional enabled",
				&domain.NetworkConfiguratorOptions{UseVirtioTransitional: true},
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   "ethernet",
					Rom:    &domainschema.Rom{Enabled: "no"},
					MTU:    &domainschema.MTU{Size: "1480"},
					Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:  &domainschema.Model{Type: "virtio-transitional"},
				},
			),
			Entry("mac address is specified",
				&domain.NetworkConfiguratorOptions{Mac: "52:54:00:00:00:01"},
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   "ethernet",
					MAC:    &domainschema.MAC{MAC: "52:54:00:00:00:01"},
					Rom:    &domainschema.Rom{Enabled: "no"},
					MTU:    &domainschema.MTU{Size: "1480"},
					Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
				},
			),
		)

		It("should not override other interfaces", func() {
			networks := []vmschema.Network{
				*vmschema.DefaultPodNetwork(),
				{Name: "secondary", NetworkSource: vmschema.NetworkSource{Multus: &vmschema.MultusNetwork{NetworkName: "sec"}}},
			}
			ifaces := []vmschema.Interface{
				{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"}},
				{Name: "secondary", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}},
			}

			expectedDomainIface := &domainschema.Interface{
				Alias:  domainschema.NewUserDefinedAlias("default"),
				Type:   "ethernet",
				Rom:    &domainschema.Rom{Enabled: "no"},
				MTU:    &domainschema.MTU{Size: "1480"},
				Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
				Model:  &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewBridgeNetworkConfigurator(ifaces, networks, domain.NetworkConfiguratorOptions{})
			Expect(err).ToNot(HaveOccurred())

			existingIface := &domainschema.Interface{Alias: domainschema.NewUserDefinedAlias("existing-iface")}
			testDomSpec := &domainschema.DomainSpec{
				Devices: domainschema.Devices{
					Interfaces: []domainschema.Interface{*existingIface}}}

			mutatedDomSpec, err := testMutator.Mutate(testDomSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*existingIface, *expectedDomainIface}))
		})

		It("should set domain interface correctly when executed more than once", func() {
			networks := []vmschema.Network{*vmschema.DefaultPodNetwork()}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "bridge"}}}

			expectedDomainIface := &domainschema.Interface{
				Alias:  domainschema.NewUserDefinedAlias("default"),
				Type:   "ethernet",
				Rom:    &domainschema.Rom{Enabled: "no"},
				MTU:    &domainschema.MTU{Size: "1480"},
				Target: &domainschema.InterfaceTarget{Device: "tap0", Managed: "no"},
				Model:  &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewBridgeNetworkConfigurator(ifaces, networks, domain.NetworkConfiguratorOptions{})
			Expect(err).ToNot(HaveOccurred())

			testDomSpec := &domainschema.DomainSpec{}

			mutatedDomSpec, err := testMutator.Mutate(testDomSpec)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))

			Expect(testMutator.Mutate(mutatedDomSpec)).To(Equal(mutatedDomSpec))
		})
	})
})
