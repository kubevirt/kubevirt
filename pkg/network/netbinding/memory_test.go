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

package netbinding_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8scorev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/netbinding"
)

var _ = Describe("Network Binding plugin compute resource overhead", func() {
	const (
		iface1name  = "net1"
		iface2name  = "net2"
		plugin1name = "plugin1"
		plugin2name = "plugin2"
	)

	DescribeTable("Memory overhead should be zero",
		func(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) {
			memoryCalculator := netbinding.MemoryCalculator{}

			actualResult := memoryCalculator.Calculate(vmi, registeredPlugins)
			Expect(actualResult.Value()).To(BeZero())
		},
		Entry("when the VMI does not have NICs and there aren't any registered plugins", libvmi.New(), nil),
		Entry("when no binding plugin is used on the VMI and there aren't any registered plugins",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			nil,
		),
		Entry("when no binding plugin is used on the VMI and there are registered plugins",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			map[string]v1.InterfaceBindingPlugin{plugin1name: newPlugin(nil)},
		),
		Entry("when binding plugin is used on the VMI, but it does not require compute overhead",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(nil),
			},
		),
		Entry("when binding plugin is used on the VMI, but it does not require memory overhead",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceCPU: resource.MustParse("100m"),
						},
					},
				),
			},
		),
	)

	DescribeTable("It should calculate memory overhead",
		func(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin, expectedValue resource.Quantity) {
			memoryCalculator := netbinding.MemoryCalculator{}

			actualResult := memoryCalculator.Calculate(vmi, registeredPlugins)
			Expect(actualResult.Value()).To(Equal(expectedValue.Value()))
		},
		Entry("when there is a single interface using a binding plugin",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("500Mi"),
						},
					},
				),
			},
			resource.MustParse("500Mi"),
		),
		Entry("when there are two interfaces using the same binding plugin",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
				libvmi.WithInterface(v1.Interface{Name: iface2name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface2name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("500Mi"),
						},
					},
				),
			},
			resource.MustParse("500Mi"),
		),
		Entry("when there are two interfaces using different binding plugins",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
				libvmi.WithInterface(v1.Interface{Name: iface2name, Binding: &v1.PluginBinding{Name: plugin2name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface2name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("500Mi"),
						},
					},
				),
				plugin2name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("600Mi"),
						},
					},
				),
			},
			resource.MustParse("1100Mi"),
		),
		Entry("when there are two interfaces and just one is using a binding plugin",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
				libvmi.WithInterface(v1.Interface{Name: iface2name, Binding: &v1.PluginBinding{Name: plugin2name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface2name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("500Mi"),
						},
					},
				),
				plugin2name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("600Mi"),
						},
					},
				),
			},
			resource.MustParse("600Mi"),
		),
		Entry("when there is a non-registered plugin, it should not be taken into account",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: iface1name, Binding: &v1.PluginBinding{Name: plugin1name}}),
				libvmi.WithNetwork(&v1.Network{Name: iface1name}),
				libvmi.WithInterface(v1.Interface{Name: iface2name, Binding: &v1.PluginBinding{Name: "non existent"}}),
				libvmi.WithNetwork(&v1.Network{Name: iface2name}),
			),
			map[string]v1.InterfaceBindingPlugin{
				plugin1name: newPlugin(
					&v1.ResourceRequirementsWithoutClaims{
						Requests: map[k8scorev1.ResourceName]resource.Quantity{
							k8scorev1.ResourceMemory: resource.MustParse("500Mi"),
						},
					},
				),
			},
			resource.MustParse("500Mi"),
		),
	)
})

func newPlugin(computeResourceOverhead *v1.ResourceRequirementsWithoutClaims) v1.InterfaceBindingPlugin {
	return v1.InterfaceBindingPlugin{ComputeResourceOverhead: computeResourceOverhead}
}
