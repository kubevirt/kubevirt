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

package controllers_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/controllers"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Network interface hot{un}plug", func() {
	DescribeTable("spec interfaces",
		func(specIfaces []v1.Interface, statusIfaces []v1.VirtualMachineInstanceNetworkInterface,
			expectedInterfaces []v1.Interface, expectedNetworks []v1.Network,
		) {
			var testNetworks []v1.Network
			for _, iface := range specIfaces {
				testNetworks = append(testNetworks, v1.Network{Name: iface.Name})
			}
			testStatusIfaces := vmispec.IndexInterfaceStatusByName(statusIfaces,
				func(i v1.VirtualMachineInstanceNetworkInterface) bool { return true })

			ifaces, networks := controllers.ClearDetachedInterfaces(specIfaces, testNetworks, testStatusIfaces)

			Expect(ifaces).To(Equal(expectedInterfaces))
			Expect(networks).To(Equal(expectedNetworks))
		},
		Entry("should remain, given non-absent interfaces, and no associated status ifaces (i.e.: plug pending)",
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.VirtualMachineInstanceNetworkInterface{},
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.Network{{Name: "blue"}, {Name: "red"}},
		),
		Entry("should remain, given non-absent interfaces, and associated status ifaces (i.e.: plugged iface)",
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: "blue"}, {Name: "red"}},
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.Network{{Name: "blue"}, {Name: "red"}},
		),
		Entry("should remain, given status iface and no associated iface in spec",
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: "RED"}},
			[]v1.Interface{{Name: "blue"}},
			[]v1.Network{{Name: "blue"}},
		),
	)
})
