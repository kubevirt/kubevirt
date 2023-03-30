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

package virtwrap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("nic hotplug on virt-launcher", func() {
	const (
		networkName = "n1"
	)

	DescribeTable("networksToHotplugWhoseInterfacesAreNotInTheDomain", func(vmiStatusIfaces []v1.VirtualMachineInstanceNetworkInterface, domainIfaces []api.Interface, expectedNetworks []string) {
		Expect(
			networksToHotplugWhoseInterfacesAreNotInTheDomain(vmiStatusIfaces, domainIfaces),
		).To(ConsistOf(expectedNetworks))
	},
		Entry("no interfaces in vmi status, and no interfaces in the domain",
			[]v1.VirtualMachineInstanceNetworkInterface{},
			[]api.Interface{},
			nil,
		),
		Entry("1 interfaces in vmi status, and an associated interface in the domain",
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: networkName}},
			[]api.Interface{{Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("1 interfaces in vmi status (when the pod interface is *not* ready), with no interfaces in the domain",
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: networkName}},
			[]api.Interface{},
			nil,
		),
		Entry("1 interfaces in vmi status (when the pod interface *is* ready), but already present in the domain",
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: networkName, InfoSource: vmispec.InfoSourceMultusStatus}},
			[]api.Interface{{Alias: api.NewUserDefinedAlias(networkName)}},
			nil,
		),
		Entry("1 interfaces in vmi status (when the pod interface *is* ready), but no interfaces in the domain",
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: networkName, InfoSource: vmispec.InfoSourceMultusStatus}},
			[]api.Interface{},
			[]string{networkName},
		),
	)
})
