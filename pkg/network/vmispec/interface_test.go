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
	Context("pod network", func() {
		const podNet0 = "podnet0"

		networks := []v1.Network{podNetwork(podNet0)}

		It("does not exist", func() {
			ifaces := []v1.Interface{interfaceWithBridgeBinding(podNet0)}
			Expect(netvmispec.IsPodNetworkWithMasqueradeBindingInterface([]v1.Network{}, ifaces)).To(BeTrue())
		})

		It("is used by a masquerade interface", func() {
			ifaces := []v1.Interface{interfaceWithMasqueradeBinding(podNet0)}
			Expect(netvmispec.IsPodNetworkWithMasqueradeBindingInterface(networks, ifaces)).To(BeTrue())
		})

		It("used by a non-masquerade interface", func() {
			ifaces := []v1.Interface{interfaceWithBridgeBinding(podNet0)}
			Expect(netvmispec.IsPodNetworkWithMasqueradeBindingInterface(networks, ifaces)).To(BeFalse())
		})
	})

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

		Context("pod network with migratable binding plugin", func() {
			It("returns false when there is no pod network", func() {
				const nonPodNet = "nonPodNet"
				networks := []v1.Network{
					{Name: nonPodNet, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				}
				ifaces := []v1.Interface{interfaceWithBridgeBinding(nonPodNet)}
				Expect(netvmispec.IsPodNetworkWithMigratableBindingPlugin(networks, ifaces, bindingPlugins)).To(BeFalse())
			})
			It("returns false when the binding is not a plugin", func() {
				networks := []v1.Network{podNetwork(podNet0)}
				ifaces := []v1.Interface{interfaceWithBridgeBinding(podNet0)}
				Expect(netvmispec.IsPodNetworkWithMigratableBindingPlugin(networks, ifaces, bindingPlugins)).To(BeFalse())
			})

			It("returns false when the plugin is not migratable", func() {
				networks := []v1.Network{podNetwork(podNet0)}
				ifaces := []v1.Interface{interfaceWithBindingPlugin(podNet0, nonMigratablePlugin)}
				Expect(netvmispec.IsPodNetworkWithMigratableBindingPlugin(networks, ifaces, bindingPlugins)).To(BeFalse())
			})

			It("returns false when non pod network is migratable", func() {
				const nonPodNetName = "nonPod"
				nonPodNetwork := v1.Network{
					Name:          nonPodNetName,
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}},
				}
				networks := []v1.Network{nonPodNetwork}
				ifaces := []v1.Interface{interfaceWithBindingPlugin(nonPodNetName, migratablePlugin)}
				Expect(netvmispec.IsPodNetworkWithMigratableBindingPlugin(networks, ifaces, bindingPlugins)).To(BeFalse())
			})

			It("returns true when the plugin is migratable", func() {
				networks := []v1.Network{podNetwork(podNet0)}
				ifaces := []v1.Interface{interfaceWithBindingPlugin(podNet0, migratablePlugin)}
				Expect(netvmispec.IsPodNetworkWithMigratableBindingPlugin(networks, ifaces, bindingPlugins)).To(BeTrue())
			})
			It("returns true when the secondary interface has is migratable", func() {
				networks := []v1.Network{podNetwork(podNet0)}
				ifaces := []v1.Interface{interfaceWithBindingPlugin(podNet0, migratablePlugin)}
				Expect(netvmispec.IsPodNetworkWithMigratableBindingPlugin(networks, ifaces, bindingPlugins)).To(BeTrue())
			})
		})

		Context("vmi", func() {
			It("shouldn't allow migration if the VMI use non-migratable binding plugin to connect to the pod network", func() {
				network := podNetwork(podNet0)
				vmi := libvmi.New(
					libvmi.WithInterface(interfaceWithBindingPlugin(podNet0, nonMigratablePlugin)),
					libvmi.WithNetwork(&network),
				)
				Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).ToNot(Succeed())
			})
			It("shouldn't allow migration if the VMI uses bridge binding to connect to the pod network", func() {
				network := podNetwork(podNet0)
				vmi := libvmi.New(
					libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
					libvmi.WithNetwork(&network),
				)
				Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).ToNot(Succeed())
			})
			It("should allow migration if the VMI uses masquerade to connect to the pod network", func() {
				network := podNetwork(podNet0)
				vmi := libvmi.New(
					libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
					libvmi.WithNetwork(&network),
				)
				Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).To(Succeed())
			})
			It("should allow migration if the VMI use bridge to connect to the pod network and has AllowLiveMigrationBridgePodNetwork annotation",
				func() {
					network := podNetwork(podNet0)
					vmi := libvmi.New(
						libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
						libvmi.WithNetwork(&network),
						libvmi.WithAnnotation(v1.AllowPodBridgeNetworkLiveMigrationAnnotation, ""),
					)
					Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).To(Succeed())
				})
			It("should allow migration if the VMI use migratable binding plugin to connect to the pod network", func() {
				network := podNetwork(podNet0)
				vmi := libvmi.New(
					libvmi.WithInterface(interfaceWithBindingPlugin(podNet0, migratablePlugin)),
					libvmi.WithNetwork(&network),
				)
				Expect(netvmispec.VerifyVMIMigratable(vmi, bindingPlugins)).To(Succeed())
			})
		})
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

func podNetwork(name string) v1.Network {
	return v1.Network{
		Name:          name,
		NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
	}
}

func interfaceWithBridgeBinding(name string) v1.Interface {
	return v1.Interface{
		Name:                   name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}},
	}
}

func interfaceWithMasqueradeBinding(name string) v1.Interface {
	return v1.Interface{
		Name:                   name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}},
	}
}

func interfaceWithBindingPlugin(name, pluginName string) v1.Interface {
	return v1.Interface{
		Name:    name,
		Binding: &v1.PluginBinding{Name: pluginName},
	}
}
