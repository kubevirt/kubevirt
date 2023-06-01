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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Network", func() {
	podNetwork := createPodNetwork("default")
	multusDefaultNetwork := createMultusDefaultNetwork("network0", "default/nad0")
	multusSecondaryNetwork1 := createMultusSecondaryNetwork("network1", "default/nad1")
	multusSecondaryNetwork2 := createMultusSecondaryNetwork("network2", "default/nad2")
	multusSecondaryNetwork3 := createMultusSecondaryNetwork("network3", "default/nad3")
	DescribeTable("should return only Multus non-default networks", func(inputNetworks, expectFilteredNetworks []v1.Network) {
		filteredNetworks := vmispec.FilterMultusNonDefaultNetworks(inputNetworks)
		genericFilteredNetworks := vmispec.FilterNetworksSpec(inputNetworks, vmispec.IsSecondaryMultusNetwork)

		Expect(filteredNetworks).To(Equal(genericFilteredNetworks))
		Expect(filteredNetworks).To(Equal(expectFilteredNetworks))
	},
		Entry("when there are no networks", []v1.Network{}, nil),
		Entry("when there is only the pod network", []v1.Network{podNetwork}, nil),
		Entry("when there is only a default Multus network", []v1.Network{multusDefaultNetwork}, nil),
		Entry("when there are default and non-default Multus networks",
			[]v1.Network{
				multusDefaultNetwork,
				multusSecondaryNetwork1,
				multusSecondaryNetwork2,
				multusSecondaryNetwork3,
			},
			[]v1.Network{
				multusSecondaryNetwork1,
				multusSecondaryNetwork2,
				multusSecondaryNetwork3,
			}),
	)

	DescribeTable("should fail to return the default network", func(inputNetworks []v1.Network) {
		Expect(vmispec.LookUpDefaultNetwork(inputNetworks)).To(BeNil())
	},
		Entry("when there are no networks", []v1.Network{}),
		Entry("when there are no default networks", []v1.Network{multusSecondaryNetwork1, multusSecondaryNetwork2}),
	)
	DescribeTable("should succeed to return the default network", func(inputNetworks []v1.Network, expectNetwork *v1.Network) {
		Expect(vmispec.LookUpDefaultNetwork(inputNetworks)).To(Equal(expectNetwork))
	},
		Entry("when there is a default pod network",
			[]v1.Network{
				podNetwork,
				multusSecondaryNetwork1,
				multusSecondaryNetwork2,
			},
			&podNetwork,
		),
		Entry("when there is a multus default network",
			[]v1.Network{
				multusDefaultNetwork,
				multusSecondaryNetwork1,
				multusSecondaryNetwork2,
			},
			&multusDefaultNetwork,
		),
	)
})

func createMultusSecondaryNetwork(name, networkName string) v1.Network {
	return createMultusNetwork(name, networkName)
}

func createMultusDefaultNetwork(name, networkName string) v1.Network {
	multusNetwork := createMultusNetwork(name, networkName)
	multusNetwork.Multus.Default = true
	return multusNetwork
}

func createMultusNetwork(name, networkName string) v1.Network {
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: networkName,
			},
		},
	}
}

func createPodNetwork(name string) v1.Network {
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Pod: &v1.PodNetwork{},
		},
	}
}
