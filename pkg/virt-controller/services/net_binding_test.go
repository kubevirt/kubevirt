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

package services_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("Network Binding", func() {
	const (
		testNetworkName1  = "net1"
		testBindingName1  = "binding1"
		testSidecarImage1 = "image1"
		testNetworkName2  = "net2"
		testBindingName2  = "binding2"
		testSidecarImage2 = "image2"
		testNetworkName3  = "net1"

		shouldSucceed = true
	)

	Context("binding plugin sidecar list", func() {
		DescribeTable("should create the correct sidecars", func(vmi *v1.VirtualMachineInstance, bindings map[string]v1.InterfaceBindingPlugin, expectedSidecars hooks.HookSidecarList) {
			config := &v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{
					Binding: bindings,
				},
			}
			sidecars, err := services.NetBindingPluginSidecarList(vmi, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(sidecars).To(ConsistOf(expectedSidecars))
		},
			Entry("VMI has binding plugin but config image is empty",
				libvmi.New(libvmi.WithInterface(v1.Interface{Name: testNetworkName1, Binding: &v1.PluginBinding{Name: testBindingName1}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				),
				map[string]v1.InterfaceBindingPlugin{testBindingName1: {}},
				nil),
			Entry("VMI has binding plugin and config image",
				libvmi.New(libvmi.WithInterface(v1.Interface{Name: testNetworkName1, Binding: &v1.PluginBinding{Name: testBindingName1}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				),
				map[string]v1.InterfaceBindingPlugin{testBindingName1: {SidecarImage: testSidecarImage1}},
				hooks.HookSidecarList{{Image: testSidecarImage1}}),
			Entry("VMI has multiple plugin bindings",
				libvmi.New(libvmi.WithInterface(v1.Interface{Name: testNetworkName1, Binding: &v1.PluginBinding{Name: testBindingName1}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
					libvmi.WithInterface(v1.Interface{Name: testNetworkName2, Binding: &v1.PluginBinding{Name: testBindingName2}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
					libvmi.WithInterface(v1.Interface{Name: testNetworkName3, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName3}),
				),
				map[string]v1.InterfaceBindingPlugin{testBindingName1: {SidecarImage: testSidecarImage1}, testBindingName2: {SidecarImage: testSidecarImage2}},
				hooks.HookSidecarList{{Image: testSidecarImage1}, {Image: testSidecarImage2}}),
			Entry("VMI has no plugin bindings",
				libvmi.New(libvmi.WithInterface(v1.Interface{Name: testNetworkName1, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				),
				nil,
				nil),
			Entry("VMI has two interfaces with the same plugin sidecar",
				libvmi.New(libvmi.WithInterface(v1.Interface{Name: testNetworkName1, Binding: &v1.PluginBinding{Name: testBindingName1}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
					libvmi.WithInterface(v1.Interface{Name: testNetworkName2, Binding: &v1.PluginBinding{Name: testBindingName1}}),
					libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
				),
				map[string]v1.InterfaceBindingPlugin{testBindingName1: {SidecarImage: testSidecarImage1}},
				hooks.HookSidecarList{{Image: testSidecarImage1}}),
		)

		It("should retrun an error when VMI has binding plugin but config doesn't exist", func() {
			vmi := libvmi.New(libvmi.WithInterface(v1.Interface{Name: testNetworkName1, Binding: &v1.PluginBinding{Name: testBindingName1}}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			)
			config := &v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{},
			}
			_, err := services.NetBindingPluginSidecarList(vmi, config)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("binding plugin configuration", func() {
		It("should read registered network binding plugins successfully", func() {
			const testPluginName = "test"
			testPlugin := v1.InterfaceBindingPlugin{SidecarImage: "kubevirt/network-slirp-binding"}

			config := &v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{
					Binding: map[string]v1.InterfaceBindingPlugin{
						"plugin1":      {SidecarImage: "org1/plugin1"},
						testPluginName: testPlugin,
						"plugin2":      {SidecarImage: "org2/plugin2"},
					},
				},
			}

			Expect(services.ReadNetBindingPluginConfiguration(config, testPluginName)).To(Equal(&testPlugin))
		})

		DescribeTable("should return no network binding plugins configuration, when config", func(config *v1.KubeVirtConfiguration) {
			Expect(services.ReadNetBindingPluginConfiguration(config, "myplugin")).To(BeNil())
		},
			Entry("is nil", nil),
			Entry("has no network configuration set", &v1.KubeVirtConfiguration{}),
			Entry("has no network binding plugin set",
				&v1.KubeVirtConfiguration{
					NetworkConfiguration: &v1.NetworkConfiguration{},
				},
			),
			Entry("has no network binding plugin registered",
				&v1.KubeVirtConfiguration{
					NetworkConfiguration: &v1.NetworkConfiguration{
						Binding: map[string]v1.InterfaceBindingPlugin{},
					},
				},
			),
			Entry("does not have the image registered",
				&v1.KubeVirtConfiguration{
					NetworkConfiguration: &v1.NetworkConfiguration{
						Binding: map[string]v1.InterfaceBindingPlugin{
							"anotherPlugin": {SidecarImage: "anotherOrg/anotherPlugin"},
						},
					},
				},
			),
		)

		It("should return default image when no Slirp network binding plugin image is registered", func() {
			expectedIfaceBindingPlugin := &v1.InterfaceBindingPlugin{SidecarImage: "quay.io/kubevirt/network-slirp-binding:20230830_638c60fc8"}
			Expect(services.ReadNetBindingPluginConfiguration(&v1.KubeVirtConfiguration{}, "slirp")).To(Equal(expectedIfaceBindingPlugin))
		})
	})
})
