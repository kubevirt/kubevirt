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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package link_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
)

var _ = Describe("Common Methods", func() {
	const maxInterfaceNameLength = 15

	Context("GenerateTapDeviceName function", func() {
		It("Should return a tap device name with one digit suffix", func() {
			Expect(virtnetlink.GenerateTapDeviceName("eth0")).To(Equal("tap0"))
		})
		It("Should return another tap device name with one digits suffix", func() {
			Expect(virtnetlink.GenerateTapDeviceName("net1")).To(Equal("tap1"))
		})
		It("Should return a tap device name with three digits suffix", func() {
			Expect(virtnetlink.GenerateTapDeviceName("eth123")).To(Equal("tap123"))
		})
		It("Should return hash network name tap device name", func() {
			hashedIfaceName := virtnetlink.GenerateTapDeviceName("pod16477688c0e")
			Expect(len(hashedIfaceName)).To(BeNumerically("<=", maxInterfaceNameLength))
			Expect(hashedIfaceName).To(Equal("tap16477688c0e"))
		})
	})
	Context("GenerateNewBridgedVmiInterfaceName function", func() {
		It("Should return the new bridge interface name", func() {
			Expect(virtnetlink.GenerateNewBridgedVmiInterfaceName("eth0")).To(Equal("eth0-nic"))
		})
		It("Should return another new bridge interface name", func() {
			Expect(virtnetlink.GenerateNewBridgedVmiInterfaceName("net12")).To(Equal("net12-nic"))
		})
		It("Should return hash network name bridge interface name", func() {
			hashedIfaceName := virtnetlink.GenerateNewBridgedVmiInterfaceName("pod16477688c0e")
			Expect(len(hashedIfaceName)).To(BeNumerically("<=", maxInterfaceNameLength))
			Expect(hashedIfaceName).To(Equal("16477688c0e-nic"))
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
