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

	"libvirt.org/go/libvirtxml"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/cmd/sidecars/network-vhostuserdra-binding/domain"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
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

func vhostUserSource(path, mode string) *libvirtxml.DomainInterfaceSource {
	return &libvirtxml.DomainInterfaceSource{
		VHostUser: &libvirtxml.DomainInterfaceSourceVHostUser{
			Chardev: &libvirtxml.DomainChardevSource{
				UNIX: &libvirtxml.DomainChardevSourceUNIX{Path: path, Mode: mode},
			},
		},
	}
}

func pciAddress(dom, bus, slot, function uint) *libvirtxml.DomainAddress {
	return &libvirtxml.DomainAddress{
		PCI: &libvirtxml.DomainAddressPCI{Domain: &dom, Bus: &bus, Slot: &slot, Function: &function},
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

		It("should honor a custom socket file name", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)
			GinkgoT().Setenv(domain.SocketFileNameEnvVar, "custom.sock")

			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			expectedDomainIface := libvirtxml.DomainInterface{
				Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
				Source: vhostUserSource(filepath.Join(testSocketDir, "custom.sock"), "client"),
				Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomain, err := testMutator.Mutate(&libvirtxml.Domain{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomain.Devices.Interfaces).To(Equal([]libvirtxml.DomainInterface{expectedDomainIface}))
		})

		DescribeTable("socket mode",
			func(mode, expectedMode string, expectErr bool) {
				GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)
				if mode != "" {
					GinkgoT().Setenv(domain.SocketModeEnvVar, mode)
				}

				networks := []vmschema.Network{draNetwork("default")}
				ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

				testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
				if expectErr {
					Expect(err).To(HaveOccurred())
					return
				}
				Expect(err).ToNot(HaveOccurred())

				expectedDomainIface := libvirtxml.DomainInterface{
					Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source: vhostUserSource(testSocketPath, expectedMode),
					Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
				}

				mutatedDomain, err := testMutator.Mutate(&libvirtxml.Domain{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomain.Devices.Interfaces).To(Equal([]libvirtxml.DomainInterface{expectedDomainIface}))
			},
			Entry("defaults to client when unset", "", "client", false),
			Entry("honors server", "server", "server", false),
			Entry("rejects an invalid mode", "foo", "", true),
		)

		DescribeTable("should add interface to domain spec given iface with",
			func(iface *vmschema.Interface, useVirtioTransitional bool, expectedDomainIface libvirtxml.DomainInterface) {
				GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

				ifaces := []vmschema.Interface{*iface}
				networks := []vmschema.Network{draNetwork("default")}

				testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, useVirtioTransitional)
				Expect(err).ToNot(HaveOccurred())

				mutatedDomain, err := testMutator.Mutate(&libvirtxml.Domain{})
				Expect(err).ToNot(HaveOccurred())
				Expect(mutatedDomain.Devices.Interfaces).To(Equal([]libvirtxml.DomainInterface{expectedDomainIface}))
			},
			Entry("vhostuser binding plugin",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}},
				false,
				libvirtxml.DomainInterface{
					Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source: vhostUserSource(testSocketPath, "client"),
					Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
				},
			),
			Entry("virtio transitional enabled",
				&vmschema.Interface{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}},
				true,
				libvirtxml.DomainInterface{
					Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source: vhostUserSource(testSocketPath, "client"),
					Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-transitional"},
				},
			),
			Entry("PCI address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					PciAddress: "0000:02:02.0",
				},
				false,
				libvirtxml.DomainInterface{
					Alias:   &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source:  vhostUserSource(testSocketPath, "client"),
					Model:   &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
					Address: pciAddress(0x0000, 0x02, 0x02, 0x0),
				},
			),
			Entry("MAC address",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					MacAddress: "02:02:02:02:02:02",
				},
				false,
				libvirtxml.DomainInterface{
					Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source: vhostUserSource(testSocketPath, "client"),
					Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
					MAC:    &libvirtxml.DomainInterfaceMAC{Address: "02:02:02:02:02:02"},
				},
			),
			Entry("ACPI index",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					ACPIIndex: 2,
				},
				false,
				libvirtxml.DomainInterface{
					Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source: vhostUserSource(testSocketPath, "client"),
					Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
					ACPI:   &libvirtxml.DomainDeviceACPI{Index: uint(2)},
				},
			),
			Entry("non virtio model",
				&vmschema.Interface{
					Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName},
					Model: "e1000",
				},
				false,
				libvirtxml.DomainInterface{
					Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
					Source: vhostUserSource(testSocketPath, "client"),
					Model:  &libvirtxml.DomainInterfaceModel{Type: "e1000"},
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

			expectedDomainIface := libvirtxml.DomainInterface{
				Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
				Source: vhostUserSource(testSocketPath, "client"),
				Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).ToNot(HaveOccurred())

			existingIface := libvirtxml.DomainInterface{Alias: &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "existing-iface"}}
			testDomain := &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Interfaces: []libvirtxml.DomainInterface{existingIface},
				},
			}

			mutatedDomain, err := testMutator.Mutate(testDomain)
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomain.Devices.Interfaces).To(Equal([]libvirtxml.DomainInterface{existingIface, expectedDomainIface}))
		})

		It("should set domain interface correctly when executed more than once", func() {
			GinkgoT().Setenv(domain.SocketDirEnvVar, testSocketDir)

			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			expectedDomainIface := libvirtxml.DomainInterface{
				Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
				Source: vhostUserSource(testSocketPath, "client"),
				Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, false)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomain, err := testMutator.Mutate(&libvirtxml.Domain{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomain.Devices.Interfaces).To(Equal([]libvirtxml.DomainInterface{expectedDomainIface}))

			Expect(testMutator.Mutate(mutatedDomain)).To(Equal(mutatedDomain))
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
			expectedMemoryBacking := &libvirtxml.DomainMemoryBacking{
				MemoryAccess: &libvirtxml.DomainMemoryAccess{
					Mode: "shared",
				},
				MemorySource: &libvirtxml.DomainMemorySource{
					Type: "memfd",
				},
			}
			mutatedDomain, err := testMutator.Mutate(&libvirtxml.Domain{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomain.MemoryBacking).To(Equal(expectedMemoryBacking))
		})

		It("should fail when private memory is predefined", func() {
			domainWithPrivateMem := &libvirtxml.Domain{
				MemoryBacking: &libvirtxml.DomainMemoryBacking{
					MemoryAccess: &libvirtxml.DomainMemoryAccess{Mode: "private"},
				},
			}
			_, err := testMutator.Mutate(domainWithPrivateMem)
			Expect(err).To(HaveOccurred())
		})

		It("should use other configs of backing memory as long as they are shared", func() {
			domainWithOtherSharedMem := &libvirtxml.Domain{
				MemoryBacking: &libvirtxml.DomainMemoryBacking{
					MemoryAccess: &libvirtxml.DomainMemoryAccess{Mode: "shared"},
					MemorySource: &libvirtxml.DomainMemorySource{Type: "file"},
				},
			}
			mutatedDomain, err := testMutator.Mutate(domainWithOtherSharedMem)
			Expect(err).NotTo(HaveOccurred())
			Expect(mutatedDomain.MemoryBacking).To(Equal(domainWithOtherSharedMem.MemoryBacking))
		})
	})
})
