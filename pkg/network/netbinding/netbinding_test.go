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

package netbinding_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/network/netbinding"

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
	)

	Context("binding plugin sidecar list", func() {
		DescribeTable("should create the correct sidecars", func(vmi *v1.VirtualMachineInstance, bindings map[string]v1.InterfaceBindingPlugin, expectedSidecars hooks.HookSidecarList) {
			config := &v1.KubeVirtConfiguration{
				NetworkConfiguration: &v1.NetworkConfiguration{
					Binding: bindings,
				},
			}
			sidecars, err := netbinding.NetBindingPluginSidecarList(vmi, config, nil)
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
			_, err := netbinding.NetBindingPluginSidecarList(vmi, config, record.NewFakeRecorder(1))
			Expect(err).To(HaveOccurred())
		})
	})

	Context("slirp binding", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = v1.NewVMI("test", "1234")
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{
				Name:                   "testnet",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &v1.InterfaceSlirp{}},
			}}
		})

		It("should create Slirp hook sidecar with registered image (specified in Kubevirt config)", func() {
			fakeRecorder := record.NewFakeRecorder(1)
			config := &v1.KubeVirtConfiguration{
				ImagePullPolicy: k8scorev1.PullIfNotPresent,
				NetworkConfiguration: &v1.NetworkConfiguration{
					Binding: map[string]v1.InterfaceBindingPlugin{
						"slirp": {SidecarImage: "kubevirt/network-slirp-plugin"},
					},
				},
			}

			Expect(netbinding.NetBindingPluginSidecarList(vmi, config, fakeRecorder)).To(ConsistOf(hooks.HookSidecarList{{
				ImagePullPolicy: k8scorev1.PullIfNotPresent,
				Image:           "kubevirt/network-slirp-plugin",
			}}))
		})

		It("should create a single slirp hook sidecar, even if multiple slirp interfaces are defined", func() {
			fakeRecorder := record.NewFakeRecorder(1)
			config := &v1.KubeVirtConfiguration{
				ImagePullPolicy: k8scorev1.PullIfNotPresent,
				NetworkConfiguration: &v1.NetworkConfiguration{
					Binding: map[string]v1.InterfaceBindingPlugin{
						"slirp": {SidecarImage: "kubevirt/network-slirp-plugin"},
					},
				},
			}

			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, v1.Interface{
				Name:                   testNetworkName2,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &v1.InterfaceSlirp{}},
			})
			Expect(netbinding.NetBindingPluginSidecarList(vmi, config, fakeRecorder)).To(ConsistOf(hooks.HookSidecarList{{
				ImagePullPolicy: k8scorev1.PullIfNotPresent,
				Image:           "kubevirt/network-slirp-plugin",
			}}))
		})

		It("should create a slirp and a custom hook sidecar", func() {
			fakeRecorder := record.NewFakeRecorder(1)
			config := &v1.KubeVirtConfiguration{
				ImagePullPolicy: k8scorev1.PullIfNotPresent,
				NetworkConfiguration: &v1.NetworkConfiguration{
					Binding: map[string]v1.InterfaceBindingPlugin{
						testBindingName1: {SidecarImage: testSidecarImage1},
						"slirp":          {SidecarImage: "kubevirt/network-slirp-plugin"},
					},
				},
			}

			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, v1.Interface{
				Name:    testNetworkName1,
				Binding: &v1.PluginBinding{Name: testBindingName1},
			})
			Expect(netbinding.NetBindingPluginSidecarList(vmi, config, fakeRecorder)).To(ConsistOf(hooks.HookSidecarList{
				{
					ImagePullPolicy: k8scorev1.PullIfNotPresent,
					Image:           "kubevirt/network-slirp-plugin",
				},
				{
					ImagePullPolicy: k8scorev1.PullIfNotPresent,
					Image:           testSidecarImage1,
				},
			}))
		})

		DescribeTable("should create Slirp hook sidecar with default image, when Kubevirt config",
			func(config *v1.KubeVirtConfiguration) {
				fakeRecorder := record.NewFakeRecorder(1)

				Expect(netbinding.NetBindingPluginSidecarList(vmi, config, fakeRecorder)).To(ConsistOf(hooks.HookSidecarList{{
					Image: netbinding.DefaultSlirpPluginImage,
				}}))
			},
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

		It("should raise UnregisteredNetworkBindingPlugin warning event when no slirp binding plugin image is registered (specified in Kubevirt config)", func() {
			fakeRecorder := record.NewFakeRecorder(1)

			Expect(netbinding.NetBindingPluginSidecarList(vmi, &v1.KubeVirtConfiguration{}, fakeRecorder)).To(ConsistOf(hooks.HookSidecarList{{
				Image: netbinding.DefaultSlirpPluginImage,
			}}))

			Expect(fakeRecorder.Events).To(HaveLen(1))
			event := <-fakeRecorder.Events
			Expect(event).To(Equal(
				fmt.Sprintf("Warning %s no Slirp network binding plugin image is set in Kubevirt config, using '%s' sidecar image for Slirp network binding configuration",
					netbinding.UnregisteredNetworkBindingPluginReason, netbinding.DefaultSlirpPluginImage),
			))
		})
	})
})
