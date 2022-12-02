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

package namescheme_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/namescheme"
)

var _ = Describe("Network Name Scheme", func() {
	Context("CreateNetworkNameScheme", func() {
		DescribeTable("should return the expected NetworkNameSchemeMap",
			func(networkList []virtv1.Network, expectedNetworkNameSchemeMap map[string]string) {
				podIfacesNameScheme := namescheme.CreateNetworkNameScheme(networkList)

				Expect(podIfacesNameScheme).To(Equal(expectedNetworkNameSchemeMap))
			},
			Entry("when network list is nil", nil, map[string]string{}),
			Entry("when no multus networks exist",
				[]virtv1.Network{
					newPodNetwork("default"),
				},
				map[string]string{
					"default": namescheme.PrimaryPodInterfaceName,
				}),
			Entry("when default multus networks exist",
				[]virtv1.Network{
					createMultusDefaultNetwork("network0", "default/nad0"),
					createMultusSecondaryNetwork("network1", "default/nad1"),
					createMultusSecondaryNetwork("network2", "default/nad2"),
				},
				map[string]string{
					"network0": namescheme.PrimaryPodInterfaceName,
					"network1": "a7662f44d65",
					"network2": "27f4a77f94e",
				}),
		)
	})
})

func createMultusSecondaryNetwork(name, networkName string) virtv1.Network {
	return createMultusNetwork(name, networkName)
}

func createMultusDefaultNetwork(name, networkName string) virtv1.Network {
	multusNetwork := createMultusNetwork(name, networkName)
	multusNetwork.Multus.Default = true
	return multusNetwork
}

func createMultusNetwork(name, networkName string) virtv1.Network {
	return virtv1.Network{
		Name: name,
		NetworkSource: virtv1.NetworkSource{
			Multus: &virtv1.MultusNetwork{
				NetworkName: networkName,
			},
		},
	}
}

func newPodNetwork(name string) virtv1.Network {
	return virtv1.Network{
		Name: name,
		NetworkSource: virtv1.NetworkSource{
			Pod: &virtv1.PodNetwork{},
		},
	}
}
