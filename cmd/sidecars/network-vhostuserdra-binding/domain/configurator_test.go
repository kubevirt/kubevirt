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

package domain_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/cmd/sidecars/network-vhostuserdra-binding/domain"

	domainschema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	ifaceTypeVhostUser = "vhostuser"

	bindingPluginName = "vhostuserdra"
	testSocketDir     = "/var/run/kubevirt/dra/hostpath"
)

var testSocketPath = filepath.Join(testSocketDir, domain.DefaultSocketFileName)

func draNetwork(name string) vmschema.Network {
	return vmschema.Network{
		Name: name,
		NetworkSource: vmschema.NetworkSource{
			ResourceClaim: &vmschema.ClaimRequest{ClaimName: "claim", RequestName: "req"},
		},
	}
}

var _ = Describe("vhostuser network configurator", func() {
	Context("generate domain spec interface", func() {
		DescribeTable("should fail to create configurator given",
			func(ifaces []vmschema.Interface, networks []vmschema.Network) {
				GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

				_, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
				Expect(err).To(HaveOccurred())
			},
			Entry("no DRA network",
				[]vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}},
				[]vmschema.Network{*vmschema.DefaultPodNetwork()},
			),
			Entry("no corresponding iface",
				[]vmschema.Interface{{Name: "not-default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}},
				[]vmschema.Network{draNetwork("default")},
			),
			Entry("interface with no binding plugin",
				[]vmschema.Interface{{Name: "default", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}}},
				[]vmschema.Network{draNetwork("default")},
			),
			Entry("interface with a different binding plugin",
				[]vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: "not-vhostuserdra"}}},
				[]vmschema.Network{draNetwork("default")},
			),
		)

		It("should fail when the DRA mountpoint env var is not set", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, "")

			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}
			networks := []vmschema.Network{draNetwork("default")}

			_, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when an invalid socket mode is requested", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)
			GinkgoT().Setenv(domain.SocketModeEnvVar, "bogus")

			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}
			networks := []vmschema.Network{draNetwork("default")}

			_, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).To(HaveOccurred())
		})

		It("should honor a custom socket file name", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)
			GinkgoT().Setenv(domain.SocketFileNameEnvVar, "custom.sock")

			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			expectedDomainIface := &domainschema.Interface{
				Alias:  domainschema.NewUserDefinedAlias("default"),
				Type:   ifaceTypeVhostUser,
				Source: domainschema.InterfaceSource{Type: "unix", Path: filepath.Join(testSocketDir, "custom.sock"), Mode: "client"},
				Model:  &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
		})

		It("should honor a custom socket mode", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)
			GinkgoT().Setenv(domain.SocketModeEnvVar, "server")

			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			expectedDomainIface := &domainschema.Interface{
				Alias:  domainschema.NewUserDefinedAlias("default"),
				Type:   ifaceTypeVhostUser,
				Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "server"},
				Model:  &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
		})

		DescribeTable("should add interface to domain spec given iface with",
			func(iface *vmschema.Interface, useVirtioTransitional bool, expectedDomainIface *domainschema.Interface) {
				GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

				ifaces := []vmschema.Interface{*iface}
				networks := []vmschema.Network{draNetwork("default")}

				testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, useVirtioTransitional)
				Expect(err).ToNot(HaveOccurred())

				mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))
			},
			Entry("vhostuser binding plugin",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}},
				false,
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   ifaceTypeVhostUser,
					Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
				},
			),
			Entry("virtio transitional enabled",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}},
				true,
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   ifaceTypeVhostUser,
					Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
					Model:  &domainschema.Model{Type: "virtio-transitional"},
				},
			),
			Entry("PCI address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					PciAddress: "0000:02:02.0",
				},
				false,
				&domainschema.Interface{
					Alias:   domainschema.NewUserDefinedAlias("default"),
					Type:    ifaceTypeVhostUser,
					Source:  domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
					Model:   &domainschema.Model{Type: "virtio-non-transitional"},
					Address: &domainschema.Address{Type: "pci", Domain: "0x0000", Bus: "0x02", Slot: "0x02", Function: "0x0"},
				},
			),
			Entry("MAC address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					MacAddress: "02:02:02:02:02:02",
				},
				false,
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   ifaceTypeVhostUser,
					Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
					MAC:    &domainschema.MAC{MAC: "02:02:02:02:02:02"},
				},
			),
			Entry("ACPI index",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					ACPIIndex: 2,
				},
				false,
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   ifaceTypeVhostUser,
					Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
					Model:  &domainschema.Model{Type: "virtio-non-transitional"},
					ACPI:   &domainschema.ACPI{Index: uint(2)},
				},
			),
			Entry("non virtio model",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					Model: "e1000",
				},
				false,
				&domainschema.Interface{
					Alias:  domainschema.NewUserDefinedAlias("default"),
					Type:   ifaceTypeVhostUser,
					Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
					Model:  &domainschema.Model{Type: "e1000"},
				},
			),
		)

		It("should not override other interfaces", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

			networks := []vmschema.Network{
				draNetwork("default"),
				{Name: "secondary", NetworkSource: vmschema.NetworkSource{Multus: &vmschema.MultusNetwork{NetworkName: "sec"}}},
			}
			ifaces := []vmschema.Interface{
				{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}},
				{Name: "secondary", InterfaceBindingMethod: vmschema.InterfaceBindingMethod{Bridge: &vmschema.InterfaceBridge{}}},
			}

			expectedDomainIface := &domainschema.Interface{
				Alias:  domainschema.NewUserDefinedAlias("default"),
				Type:   ifaceTypeVhostUser,
				Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
				Model:  &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
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
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			expectedDomainIface := &domainschema.Interface{
				Alias:  domainschema.NewUserDefinedAlias("default"),
				Type:   ifaceTypeVhostUser,
				Source: domainschema.InterfaceSource{Type: "unix", Path: testSocketPath, Mode: "client"},
				Model:  &domainschema.Model{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomSpec, err := testMutator.Mutate(&domainschema.DomainSpec{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomSpec.Devices.Interfaces).To(Equal([]domainschema.Interface{*expectedDomainIface}))

			Expect(testMutator.Mutate(mutatedDomSpec)).To(Equal(mutatedDomSpec))
		})
	})

	Context("should define memoryBacking for vhost-user", func() {
		var testMutator *domain.VhostUserNetworkConfigurator
		BeforeEach(func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			var err error
			testMutator, err = domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
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
