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

package migration_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	netmigration "kubevirt.io/kubevirt/pkg/network/migration"
)

var _ = Describe("VerifyVMIMigratable", func() {
	const (
		migratablePlugin    = "mig"
		nonMigratablePlugin = "non_mig"
		podNet0             = "default"
	)

	bindingPlugins := map[string]v1.InterfaceBindingPlugin{
		migratablePlugin:    {Migration: &v1.InterfaceBindingMigration{}},
		nonMigratablePlugin: {},
	}

	const draGateEnabled = true

	DescribeTable("should allow migration", func(vmi *v1.VirtualMachineInstance, isDRAEnabled bool) {
		configProvider := stubConfigProvider{
			bindingPlugins: bindingPlugins,
			isDRAEnabled:   isDRAEnabled,
		}
		Expect(netmigration.VerifyVMIMigratable(vmi, configProvider)).To(Succeed())
	},
		Entry("when the VMI uses masquerade to connect to the pod network",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses bridge to connect to the pod network and has AllowLiveMigrationBridgePodNetwork annotation",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithAnnotation(v1.AllowPodBridgeNetworkLiveMigrationAnnotation, ""),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses migratable binding plugin to connect to the pod network",
			libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(podNet0, libvmi.WithBindingPlugin(v1.PluginBinding{Name: migratablePlugin}))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses passtBinding to connect to the pod network",
			libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(podNet0, libvmi.WithPasstBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses a DRA network backed by a ResourceClaimTemplate",
			newDRAVMI(v1.VirtualMachineInstanceResourceClaim{
				Name:                      "claim0",
				ResourceClaimTemplateName: new("claim-template"),
			}),
			draGateEnabled,
		),
		Entry("when the VMI uses a DRA network backed by a ResourceClaim but the feature gate is disabled",
			newDRAVMI(v1.VirtualMachineInstanceResourceClaim{
				Name:              "claim0",
				ResourceClaimName: new("existing-claim"),
			}),
			!draGateEnabled,
		),
		Entry("when the VMI uses a DRA network backed by a ResourceClaimTemplate alongside a masquerade pod network",
			newDRAVMI(v1.VirtualMachineInstanceResourceClaim{
				Name:                      "claim0",
				ResourceClaimTemplateName: new("claim-template"),
			},
				libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
	)

	DescribeTable("shouldn't allow migration", func(vmi *v1.VirtualMachineInstance, isDRAEnabled bool) {
		configProvider := stubConfigProvider{
			bindingPlugins: bindingPlugins,
			isDRAEnabled:   isDRAEnabled,
		}
		Expect(netmigration.VerifyVMIMigratable(vmi, configProvider)).ToNot(Succeed())
	},
		Entry("when the VMI uses bridge binding to connect to the pod network",
			libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses non-migratable binding plugin to connect to the pod network",
			libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(podNet0, libvmi.WithBindingPlugin(v1.PluginBinding{Name: nonMigratablePlugin}))),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses a DRA network backed by a ResourceClaim",
			newDRAVMI(v1.VirtualMachineInstanceResourceClaim{
				Name:              "claim0",
				ResourceClaimName: new("existing-claim"),
			}),
			draGateEnabled,
		),
		Entry("when the VMI uses a DRA network backed by a ResourceClaim alongside a masquerade pod network",
			newDRAVMI(v1.VirtualMachineInstanceResourceClaim{
				Name:              "claim0",
				ResourceClaimName: new("existing-claim"),
			},
				libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
		Entry("when the VMI uses a DRA network backed by a ResourceClaimTemplate alongside a non-migratable bridge pod network",
			newDRAVMI(v1.VirtualMachineInstanceResourceClaim{
				Name:                      "claim0",
				ResourceClaimTemplateName: new("claim-template"),
			},
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
			draGateEnabled,
		),
	)
})

func newDRAVMI(claim v1.VirtualMachineInstanceResourceClaim, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	const netName = "net1"

	return libvmi.New(slices.Concat([]libvmi.Option{
		libvmi.WithInterface(libvmi.NewInterface(netName, libvmi.WithBindingPlugin(v1.PluginBinding{Name: "draBinding"}))),
		libvmi.WithNetwork(libvmi.DRANetwork(netName, claim.Name, "req0")),
		libvmi.WithResourceClaim(claim),
	}, opts)...)
}

type stubConfigProvider struct {
	bindingPlugins map[string]v1.InterfaceBindingPlugin
	isDRAEnabled   bool
}

func (c stubConfigProvider) GetNetworkBindings() map[string]v1.InterfaceBindingPlugin {
	return c.bindingPlugins
}

func (c stubConfigProvider) NetworkDevicesWithDRAGateEnabled() bool {
	return c.isDRAEnabled
}
