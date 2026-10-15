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

	DescribeTable("should allow migration", func(vmi *v1.VirtualMachineInstance) {
		Expect(netmigration.VerifyVMIMigratable(vmi, bindingPlugins)).To(Succeed())
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
				libvmi.WithInterface(libvmi.NewInterface(podNet0, libvmi.WithPasstBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		),
	)

	DescribeTable("shouldn't allow migration", func(vmi *v1.VirtualMachineInstance) {
		Expect(netmigration.VerifyVMIMigratable(vmi, bindingPlugins)).ToNot(Succeed())
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

func interfaceWithBindingPlugin(name, pluginName string) v1.Interface {
	return v1.Interface{
		Name:    name,
		Binding: &v1.PluginBinding{Name: pluginName},
	}
}
