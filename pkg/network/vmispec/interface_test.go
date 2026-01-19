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

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("VMI network spec", func() {
	Context("SR-IOV", func() {
		It("finds no SR-IOV interfaces in list", func() {
			ifaces := []v1.Interface{
				{
					Name:                   "net0",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
				{
					Name:                   "net1",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
				},
			}

			Expect(netvmispec.FilterSRIOVInterfaces(ifaces)).To(BeEmpty())
			Expect(netvmispec.SRIOVInterfaceExist(ifaces)).To(BeFalse())
		})

		It("finds two SR-IOV interfaces in list", func() {
			sriovNet1 := v1.Interface{
				Name:                   "sriov-net1",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
			}
			sriovNet2 := v1.Interface{
				Name:                   "sriov-net2",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
			}

			ifaces := []v1.Interface{
				{
					Name:                   "masq-net0",
					InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
				},
				sriovNet1,
				sriovNet2,
			}

			Expect(netvmispec.FilterSRIOVInterfaces(ifaces)).To(Equal([]v1.Interface{sriovNet1, sriovNet2}))
			Expect(netvmispec.SRIOVInterfaceExist(ifaces)).To(BeTrue())
		})
	})

	Context("migratable", func() {
		const (
			migratablePlugin    = "mig"
			nonMigratablePlugin = "non_mig"
			podNet0             = "default"
		)

		bindingPlugins := map[string]v1.InterfaceBindingPlugin{
			migratablePlugin:    {Migration: &v1.InterfaceBindingMigration{}},
			nonMigratablePlugin: {},
		}

		DescribeTable("should allow migration", func(vmi *v1.VirtualMachineInstance) {
			Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).To(Succeed())
		},
			Entry("when the VMI uses masquerade to connect to the pod network",
				libvmi.New(
					libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
			Entry("when the VMI uses bridge to connect to the pod network and has AllowLiveMigrationBridgePodNetwork annotation",
				libvmi.New(
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithAnnotation(v1.AllowPodBridgeNetworkLiveMigrationAnnotation, ""),
				),
			),
			Entry("when the VMI uses migratable binding plugin to connect to the pod network",
				libvmi.New(
					libvmi.WithInterface(interfaceWithBindingPlugin(podNet0, migratablePlugin)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
			Entry("when the VMI uses passtBinding to connect to the pod network",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithPasstBinding(podNet0)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
		)

		DescribeTable("shouldn't allow migration", func(vmi *v1.VirtualMachineInstance) {
			Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).ToNot(Succeed())
		},
			Entry("when the VMI uses bridge binding to connect to the pod network",
				libvmi.New(
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
			Entry("when the VMI uses non-migratable binding plugin to connect to the pod network",
				libvmi.New(
					libvmi.WithInterface(interfaceWithBindingPlugin(podNet0, nonMigratablePlugin)),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				),
			),
		)
	})

	const (
		deviceInfoPlugin    = "deviceinfo"
		nonDeviceInfoPlugin = "non_deviceinfo"
	)

	bindingPlugins := map[string]v1.InterfaceBindingPlugin{
		deviceInfoPlugin:    {DownwardAPI: v1.DeviceInfo},
		nonDeviceInfoPlugin: {},
	}

	Context("binding plugin network with device info", func() {
		It("returns false given non binding-plugin interface", func() {
			Expect(netvmispec.HasBindingPluginDeviceInfo(
				libvmi.InterfaceDeviceWithBridgeBinding("net1"),
				bindingPlugins,
			)).To(BeFalse())
		})
		It("returns false when interface binding is not plugin with device-info", func() {
			Expect(netvmispec.HasBindingPluginDeviceInfo(
				interfaceWithBindingPlugin("net1", nonDeviceInfoPlugin),
				bindingPlugins,
			)).To(BeFalse())
		})
		It("returns true when interface binding is plugin with device-info", func() {
			Expect(netvmispec.HasBindingPluginDeviceInfo(
				interfaceWithBindingPlugin("net2", deviceInfoPlugin),
				bindingPlugins,
			)).To(BeTrue())
		})
	})
	Context("binding plugin network with device info exist", func() {
		It("returns false when there is no network with device info plugin", func() {
			ifaces := []v1.Interface{interfaceWithBindingPlugin("net1", nonDeviceInfoPlugin)}
			Expect(netvmispec.BindingPluginNetworkWithDeviceInfoExist(ifaces, bindingPlugins)).To(BeFalse())
		})
		It("returns true when there is at least one network with device-info plugin", func() {
			ifaces := []v1.Interface{
				interfaceWithBindingPlugin("net1", nonDeviceInfoPlugin),
				interfaceWithBindingPlugin("net2", deviceInfoPlugin),
			}
			Expect(netvmispec.BindingPluginNetworkWithDeviceInfoExist(ifaces, bindingPlugins)).To(BeTrue())
		})
	})
})

func interfaceWithBindingPlugin(name, pluginName string) v1.Interface {
	return v1.Interface{
		Name:    name,
		Binding: &v1.PluginBinding{Name: pluginName},
	}
}
