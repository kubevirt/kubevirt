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

package sriov_test

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	sriovhostdev "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/sriov"
)

var _ = Describe("SRIOV PCI address pool with network-pci-map", func() {
	var emptyFileBytes []byte
	emptyNetworkPCIMapBytes := []byte("{}")
	networkPCIMapWithThreeNetworks := []byte(`{
		"interfaces": [
			{"network":"network1", "deviceInfo":{
				"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:04:02.5"}}},
			{"network":"network2", "deviceInfo":{
				"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:04:02.7"}}},
			{"network":"network3", "deviceInfo":{
				"type":"pci","version":"1.0.0","pci":{"pci-address":"0000:04:02.8"}}}
		]
	}`)
	It("should fail to create the pool when network-pci-map file is empty", func() {
		pool, err := sriovhostdev.NewPCIAddressPoolWithNetworkStatus(emptyFileBytes)

		var expectedTypeError *json.SyntaxError
		Expect(errors.As(err, &expectedTypeError)).To(BeTrue())
		Expect(pool).To(BeNil())
	})

	It("should create a pool with zero length when network-pci-map file holds empty map", func() {
		pool, err := sriovhostdev.NewPCIAddressPoolWithNetworkStatus(emptyNetworkPCIMapBytes)

		Expect(err).ToNot(HaveOccurred())
		Expect(pool.Len()).To(BeZero())
	})

	It("should fail to pop a pci-address from the pool when network-pci-map file has valid data but requested network is not in pool", func() {
		pool, err := sriovhostdev.NewPCIAddressPoolWithNetworkStatus(networkPCIMapWithThreeNetworks)
		Expect(err).ToNot(HaveOccurred())

		_, err = pool.Pop("foo")
		Expect(err).To(HaveOccurred())
	})

	It("should succeed to pop a pci-address from the pool when network-pci map is valid", func() {
		pool, err := sriovhostdev.NewPCIAddressPoolWithNetworkStatus(networkPCIMapWithThreeNetworks)
		Expect(err).ToNot(HaveOccurred())

		expectedNetworkToPCIMap := map[string]string{
			"network1": "0000:04:02.5",
			"network2": "0000:04:02.7",
			"network3": "0000:04:02.8",
		}
		for requestedNetwork, expectedPciAddress := range expectedNetworkToPCIMap {
			By("check pop succeeds to retrieve the PCI Address")
			Expect(pool.Pop(requestedNetwork)).To(Equal(expectedPciAddress))

			By("check pop fails to retrieve from the network after it was already popped")
			_, err := pool.Pop(requestedNetwork)
			Expect(err).To(HaveOccurred())
		}
	})

	DescribeTable("should return empty pool given network-info annotation with",
		func(netInfo string) {
			pool, err := sriovhostdev.NewPCIAddressPoolWithNetworkStatus([]byte(netInfo))
			Expect(err).ToNot(HaveOccurred())
			Expect(pool.Len()).To(Equal(0))
		},
		Entry("element who has no device-info",
			`{"interfaces":[{"network":"network1"}]}`,
		),
		Entry("element with device-info but no PCI info",
			`{"interfaces":[{"network":"network1","deviceInfo":{"version":"1.0.0","type":"pci"}}]}`,
		),
		Entry("element with device-info PCI info but no PCI address",
			`{"interfaces":[{"network":"network1","deviceInfo":{"version":"1.0.0","type":"pci","pci":{"pci-address":""}}}]}`,
		),
	)
})
