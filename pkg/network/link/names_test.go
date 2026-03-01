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

package link_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

var _ = Describe("Common Methods", func() {
	const maxInterfaceNameLength = 15

	Context("GenerateTapDeviceName function", func() {
		DescribeTable("Should return tap0 for the primary network", func(network v1.Network) {
			Expect(virtnetlink.GenerateTapDeviceName("eth0", network)).To(Equal("tap0"))
		},
			Entry("When connected to pod network",
				v1.Network{Name: "somenet", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}},
			),
			Entry("When connected to default Multus network",
				v1.Network{Name: "somenet", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{Default: true}}},
			),
		)
		It("Should return an ordinal name when using ordinal naming scheme", func() {
			secondaryNet := v1.Network{Name: "secondary", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}
			Expect(virtnetlink.GenerateTapDeviceName("net1", secondaryNet)).To(Equal("tap1"))
		})
		It("Should return hashed name when using hanshed naming scheme", func() {
			secondaryNet := v1.Network{Name: "secondary", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}
			hashedIfaceName := virtnetlink.GenerateTapDeviceName("pod16477688c0e", secondaryNet)
			Expect(len(hashedIfaceName)).To(BeNumerically("<=", maxInterfaceNameLength))
			Expect(hashedIfaceName).To(Equal("tap16477688c0e"))
		})
	})
	Context("GenerateNewBridgedVmiInterfaceName function", func() {
		It("Should return the new bridge interface name", func() {
			name, err := virtnetlink.GenerateNewBridgedVmiInterfaceName("eth0")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("eth0-nic"))
		})
		It("Should return another new bridge interface name", func() {
			name, err := virtnetlink.GenerateNewBridgedVmiInterfaceName("net12")
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal("net12-nic"))
		})
		It("Should return hash network name bridge interface name", func() {
			hashedIfaceName, err := virtnetlink.GenerateNewBridgedVmiInterfaceName("pod16477688c0e")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(hashedIfaceName)).To(BeNumerically("<=", maxInterfaceNameLength))
			Expect(hashedIfaceName).To(Equal("16477688c0e-nic"))
		})
		It("Should return an error when the generated name exceeds 15 characters", func() {
			// "toolongprefix" (13 chars) + "-nic" (4 chars) = 17 chars > 15
			_, err := virtnetlink.GenerateNewBridgedVmiInterfaceName("toolongprefix")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds the maximum allowed length"))
		})
	})
	Context("GenerateBridgeName function", func() {
		It("Should return the new bridge interface name", func() {
			Expect(virtnetlink.GenerateBridgeName("eth0")).To(Equal("k6t-eth0"))
		})
		It("Should return another new bridge interface name", func() {
			Expect(virtnetlink.GenerateBridgeName("net12")).To(Equal("k6t-net12"))
		})
		It("Should return hash network name bridge interface name", func() {
			hashedIfaceName := virtnetlink.GenerateBridgeName("pod16477688c0e")
			Expect(len(hashedIfaceName)).To(BeNumerically("<=", maxInterfaceNameLength))
			Expect(hashedIfaceName).To(Equal("k6t-16477688c0e"))
		})
	})
})
