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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirtxml"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/cmd/sidecars/test-network-vhostuser-dra-binding/domain"
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	bindingPluginName = "vhostuserdra"
	testSocketPath    = "/var/run/kubevirt/dra/vhostuser/vhost.sock"
)

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

var _ = Describe("vhostuser network configurator", func() {
	Context("generate domain spec interface", func() {
		DescribeTable("should fail to create configurator given",
			func(ifaces []vmschema.Interface, networks []vmschema.Network) {
				_, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, testSocketPath)
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

		It("should fail when the socket path is empty", func() {
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}
			networks := []vmschema.Network{draNetwork("default")}

			_, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, "")
			Expect(err).To(HaveOccurred())
		})

		It("should add interface to domain spec", func() {
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}
			networks := []vmschema.Network{draNetwork("default")}

			expectedDomainIface := libvirtxml.DomainInterface{
				Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
				Source: vhostUserSource(testSocketPath, "client"),
				Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, testSocketPath)
			Expect(err).ToNot(HaveOccurred())

			mutatedDomain, err := testMutator.Mutate(&libvirtxml.Domain{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mutatedDomain.Devices.Interfaces).To(Equal([]libvirtxml.DomainInterface{expectedDomainIface}))
		})

		It("should not override other interfaces", func() {
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

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, testSocketPath)
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
			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			expectedDomainIface := libvirtxml.DomainInterface{
				Alias:  &libvirtxml.DomainAlias{Name: api.UserAliasPrefix + "default"},
				Source: vhostUserSource(testSocketPath, "client"),
				Model:  &libvirtxml.DomainInterfaceModel{Type: "virtio-non-transitional"},
			}

			testMutator, err := domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, testSocketPath)
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
			networks := []vmschema.Network{draNetwork("default")}
			ifaces := []vmschema.Interface{{Name: "default", Binding: &vmschema.PluginBinding{Name: bindingPluginName}}}

			var err error
			testMutator, err = domain.NewVhostUserNetworkConfigurator(ifaces, networks, bindingPluginName, testSocketPath)
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
