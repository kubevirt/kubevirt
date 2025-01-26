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
 * Copyright The KubeVirt Authors
 *
 */

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	network "kubevirt.io/kubevirt/pkg/network/setup"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Network setup filters", func() {
	Context("FilterNetsForVMStartup", func() {
		It("Should return a list non-absent networks", func() {
			const absentNetName = "absent-net"
			absentIface := libvmi.InterfaceDeviceWithBridgeBinding(absentNetName)
			absentIface.State = v1.InterfaceStateAbsent

			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(absentIface),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(absentNetName, "somenad")),
			)

			Expect(network.FilterNetsForVMStartup(vmi)).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))
		})
	})
})
