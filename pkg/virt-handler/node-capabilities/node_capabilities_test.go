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
 * Copyright the KubeVirt Authors.
 *
 */

package nodecapabilities_test

import (
	_ "embed"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/pointer"
	nodecapabilities "kubevirt.io/kubevirt/pkg/virt-handler/node-capabilities"
)

//go:embed testdata/capabilities.xml
var hostCapabilitiesXML string

//go:embed testdata/capabilities_with_numa.xml
var hostCapablitiesWithNUMAXML string

var _ = Describe("node-capabilities", func() {

	Context("Host capabilities", func() {
		It("should be able to read the TSC timer frequency from the host", func() {
			capabilities, err := nodecapabilities.ParseHostCapabilities(hostCapabilitiesXML)
			Expect(err).ToNot(HaveOccurred())
			Expect(capabilities.Host.CPU.Counter.Name).To(Equal("tsc"))
			Expect(capabilities.Host.CPU.Counter.Frequency).To(BeEquivalentTo(4008012000))
			Expect(capabilities.Host.CPU.Counter.Scaling).ToNot(Equal("yes"))
		})

		It("should properly read cpu siblings", func() {
			capabilities, err := nodecapabilities.ParseHostCapabilities(hostCapabilitiesXML)
			Expect(err).ToNot(HaveOccurred())
			Expect(capabilities.Host.NUMA.Cells.Cells).To(HaveLen(1))
			Expect(capabilities.Host.NUMA.Cells.Cells[0].CPUS.CPUs).To(HaveLen(8))
			Expect(capabilities.Host.NUMA.Cells.Cells[0].CPUS.CPUs[0].Siblings).To(Equal("0,4"))
		})

		It("should read the NUMA topology from the host", func() {
			expectedCell := libvirtxml.CapsHostNUMACell{
				ID:     0,
				Memory: &libvirtxml.CapsHostNUMAMemory{Size: 1289144, Unit: "KiB"},
				PageInfo: []libvirtxml.CapsHostNUMAPageInfo{
					{Count: 314094, Unit: "KiB", Size: 4},
					{Count: 16, Unit: "KiB", Size: 2048},
					{Count: 0, Unit: "KiB", Size: 1048576},
				},
				Distances: &libvirtxml.CapsHostNUMADistances{
					Siblings: []libvirtxml.CapsHostNUMASibling{
						{ID: 0, Value: 10},
						{ID: 1, Value: 10},
						{ID: 2, Value: 10},
						{ID: 3, Value: 10},
					},
				},
				CPUS: &libvirtxml.CapsHostNUMACPUs{
					Num: 6,
					CPUs: []libvirtxml.CapsHostNUMACPU{
						{ID: 0, SocketID: pointer.P(0), DieID: pointer.P(0), CoreID: pointer.P(0), Siblings: "0"},
						{ID: 1, SocketID: pointer.P(1), DieID: pointer.P(0), CoreID: pointer.P(0), Siblings: "1"},
						{ID: 2, SocketID: pointer.P(2), DieID: pointer.P(0), CoreID: pointer.P(0), Siblings: "2"},
						{ID: 3, SocketID: pointer.P(3), DieID: pointer.P(0), CoreID: pointer.P(0), Siblings: "3"},
						{ID: 4, SocketID: pointer.P(4), DieID: pointer.P(0), CoreID: pointer.P(0), Siblings: "4"},
						{ID: 5, SocketID: pointer.P(5), DieID: pointer.P(0), CoreID: pointer.P(0), Siblings: "5"},
					},
				},
			}

			capabilities, err := nodecapabilities.ParseHostCapabilities(hostCapablitiesWithNUMAXML)
			Expect(err).ToNot(HaveOccurred())
			Expect(capabilities.Host.NUMA.Cells.Num).To(BeEquivalentTo(4))
			Expect(capabilities.Host.NUMA.Cells.Cells).To(HaveLen(4))
			Expect(capabilities.Host.NUMA.Cells.Cells[0]).To(Equal(expectedCell))
		})
	})
})
