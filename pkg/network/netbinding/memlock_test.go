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

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/netbinding"
)

var _ = Describe("Handling network binding plugin memlock limit requirements", func() {

	DescribeTable("Find out if network binding plugin requires other memory lock limits",
		func(memlockReqs *v1.MemoryLockLimitRequirements, expected bool) {
			Expect(netbinding.NetBindingHasMemoryLockRequirements(&v1.InterfaceBindingPlugin{MemoryLockLimits: memlockReqs})).To(Equal(expected))
		},
		Entry("Is empty",
			&v1.MemoryLockLimitRequirements{},
			false,
		),
		Entry("Offset is zero",
			&v1.MemoryLockLimitRequirements{Offset: resource.NewScaledQuantity(0, resource.Kilo)},
			false,
		),
		Entry("LockGuestMemory is true",
			&v1.MemoryLockLimitRequirements{LockGuestMemory: true},
			true,
		),
		Entry("Contains an offset",
			&v1.MemoryLockLimitRequirements{Offset: resource.NewScaledQuantity(10, resource.Kilo)},
			true,
		),
		Entry("Contains a both offset and ratio",
			&v1.MemoryLockLimitRequirements{
				LockGuestMemory: true,
				Offset:          resource.NewScaledQuantity(10, resource.Kilo)},
			true,
		),
	)

	const (
		pluginName1 = "plugin1"
		pluginName2 = "plugin2"
	)

	DescribeTable("Apply memory lock requirements",
		// These tests assume/simulate that the guest memory is 1 kB
		func(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin, expected *resource.Quantity) {
			Expect(netbinding.ApplyNetBindingMemlockRequirements(
				resource.NewScaledQuantity(1, resource.Kilo),
				vmi,
				registeredPlugins,
			)).To(Equal(expected))
		},
		Entry("No interfaces at all",
			libvmi.New(),
			nil,
			resource.NewScaledQuantity(1, resource.Kilo),
		),
		Entry("Interfaces are not network binding plugins",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			map[string]v1.InterfaceBindingPlugin{pluginName1: pluginFromMemLockReqs(true, nil)},
			resource.NewScaledQuantity(1, resource.Kilo),
		),
		Entry("Interfaces are network binding plugins without memlock requirements",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: "foo", Binding: &v1.PluginBinding{Name: pluginName1}}),
				libvmi.WithNetwork(&v1.Network{Name: "foo"}),
			),
			map[string]v1.InterfaceBindingPlugin{pluginName1: pluginFromMemLockReqs(true, nil)},
			resource.NewScaledQuantity(1, resource.Kilo),
		),
		Entry("Interfaces are network binding plugins one with memlock requirements other without them",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: "foo", Binding: &v1.PluginBinding{Name: pluginName1}}),
				libvmi.WithInterface(v1.Interface{Name: "bar", Binding: &v1.PluginBinding{Name: pluginName2}}),
				libvmi.WithNetwork(&v1.Network{Name: "foo"}),
				libvmi.WithNetwork(&v1.Network{Name: "bar"}),
			),
			map[string]v1.InterfaceBindingPlugin{
				pluginName1: pluginFromMemLockReqs(true, nil),
				pluginName2: pluginFromMemLockReqs(false, nil)},
			resource.NewScaledQuantity(1, resource.Kilo),
		),
		Entry("Interfaces are network binding plugins with different memlock requirements",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: "foo", Binding: &v1.PluginBinding{Name: pluginName1}}),
				libvmi.WithInterface(v1.Interface{Name: "bar", Binding: &v1.PluginBinding{Name: pluginName2}}),
				libvmi.WithNetwork(&v1.Network{Name: "foo"}),
				libvmi.WithNetwork(&v1.Network{Name: "bar"}),
			),
			map[string]v1.InterfaceBindingPlugin{
				pluginName1: pluginFromMemLockReqs(true, resource.NewScaledQuantity(1, resource.Kilo)),
				pluginName2: pluginFromMemLockReqs(false, resource.NewScaledQuantity(23, resource.Kilo))},
			resource.NewScaledQuantity(25, resource.Kilo),
		),
		Entry("Interfaces are network binding plugins with same memlock requirements",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: "foo", Binding: &v1.PluginBinding{Name: pluginName1}}),
				libvmi.WithInterface(v1.Interface{Name: "bar", Binding: &v1.PluginBinding{Name: pluginName1}}),
				libvmi.WithNetwork(&v1.Network{Name: "foo"}),
				libvmi.WithNetwork(&v1.Network{Name: "bar"}),
			),
			map[string]v1.InterfaceBindingPlugin{
				pluginName1: pluginFromMemLockReqs(true, resource.NewScaledQuantity(1, resource.Kilo)),
			},
			resource.NewScaledQuantity(3, resource.Kilo),
		),
		Entry("Interface are network binding plugins with memlock requirements with different offset scale",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: "foo", Binding: &v1.PluginBinding{Name: pluginName1}}),
				libvmi.WithNetwork(&v1.Network{Name: "foo"}),
			),
			map[string]v1.InterfaceBindingPlugin{
				pluginName1: pluginFromMemLockReqs(true, resource.NewScaledQuantity(1, resource.Giga)),
			},
			resource.NewScaledQuantity(1000001, resource.Kilo),
		),
	)

})

func pluginFromMemLockReqs(lock bool, offset *resource.Quantity) v1.InterfaceBindingPlugin {
	return v1.InterfaceBindingPlugin{MemoryLockLimits: &v1.MemoryLockLimitRequirements{LockGuestMemory: lock, Offset: offset}}
}
