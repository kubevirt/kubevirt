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
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Default pod network", func() {
	DescribeTable("It should not automatically add the pod network to the VMI", func(vmi *v1.VirtualMachineInstance) {
		origSpec := vmi.Spec.DeepCopy()

		Expect(vmispec.SetDefaultNetworkInterface(stubClusterConfig{}, &vmi.Spec)).To(Succeed())

		Expect(vmi.Spec).To(Equal(*origSpec))
	},
		Entry("when AutoattachPodInterface is false", libvmi.New(libvmi.WithAutoAttachPodInterface(false))),
		Entry("when at least one network and one interface are specified",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: "mynet"}),
				libvmi.WithNetwork(&v1.Network{Name: "mynet"}),
			),
		),
	)

	DescribeTable("It should add the pod network using bridge binding", func(vmi *v1.VirtualMachineInstance) {
		config := stubClusterConfig{
			defaultNetworkInterface:              string(v1.BridgeInterface),
			isBridgeInterfaceEnabledOnPodNetwork: true,
		}

		Expect(vmispec.SetDefaultNetworkInterface(config, &vmi.Spec)).To(Succeed())

		Expect(vmi.Spec.Domain.Devices.Interfaces).To(Equal([]v1.Interface{*v1.DefaultBridgeNetworkInterface()}))
		Expect(vmi.Spec.Networks).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))
	},
		Entry("when AutoattachPodInterface is undefined", libvmi.New()),
		Entry("when AutoattachPodInterface is true", libvmi.New(libvmi.WithAutoAttachPodInterface(true))),
	)

	DescribeTable("It should add the pod network using masquerade binding", func(vmi *v1.VirtualMachineInstance) {
		config := stubClusterConfig{defaultNetworkInterface: string(v1.MasqueradeInterface)}

		Expect(vmispec.SetDefaultNetworkInterface(config, &vmi.Spec)).To(Succeed())

		Expect(vmi.Spec.Domain.Devices.Interfaces).To(Equal([]v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}))
		Expect(vmi.Spec.Networks).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))
	},
		Entry("when AutoattachPodInterface is undefined", libvmi.New()),
		Entry("when AutoattachPodInterface is true", libvmi.New(libvmi.WithAutoAttachPodInterface(true))),
	)

	DescribeTable("It should return an error", func(config stubClusterConfig, expectedErrMsg string) {
		Expect(vmispec.SetDefaultNetworkInterface(config, &v1.VirtualMachineInstanceSpec{})).To(MatchError(expectedErrMsg))
	},
		Entry("when bridge binding is the cluster-wide default, but it is disabled on pod network",
			stubClusterConfig{
				defaultNetworkInterface:              string(v1.BridgeInterface),
				isBridgeInterfaceEnabledOnPodNetwork: false,
			},
			"bridge interface is not enabled in kubevirt-config",
		),
		Entry("when the deprecated slirp binding is the cluster-wide default",
			stubClusterConfig{defaultNetworkInterface: string(v1.DeprecatedSlirpInterface)},
			"slirp interface is deprecated as of v1.3",
		),
	)
})

type stubClusterConfig struct {
	defaultNetworkInterface              string
	isBridgeInterfaceEnabledOnPodNetwork bool
}

func (scc stubClusterConfig) GetDefaultNetworkInterface() string {
	return scc.defaultNetworkInterface
}

func (scc stubClusterConfig) IsBridgeInterfaceOnPodNetworkEnabled() bool {
	return scc.isBridgeInterfaceEnabledOnPodNetwork
}
