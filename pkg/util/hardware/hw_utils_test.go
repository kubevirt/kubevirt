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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package hardware

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
)

type CPUTopologyData struct {
	cpus              []int
	getCPUsErr        error
	siblings          map[int][]int
	getCPUSiblingsErr error
	res               []int
	err               bool
}

func (t CPUTopologyData) GetCPUs() ([]int, error) {
	return t.cpus, t.getCPUsErr
}

func (t CPUTopologyData) GetCPUSiblings(cpu int) ([]int, error) {
	return t.siblings[cpu], t.getCPUSiblingsErr
}

var _ = Describe("Hardware utils test", func() {

	Context("cpuset parser", func() {
		It("shoud parse cpuset correctly", func() {
			expectedList := []int{0, 1, 2, 7, 12, 13, 14}
			cpusetLine := "0-2,7,12-14"
			lst, err := ParseCPUSetLine(cpusetLine)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(lst)).To(Equal(7))
			Expect(lst).To(Equal(expectedList))
		})
	})

	Context("CPU threads grouping", func() {
		It("shoud group CPU threads correctly", func() {
			cpus8 := []int{0, 1, 2, 3, 4, 5, 6, 7}
			cpus3 := []int{1, 2, 5}
			siblings00 := map[int][]int{
				0: {0}, 4: {4},
				1: {1}, 5: {5},
				2: {2}, 6: {6},
				3: {3}, 7: {7},
			}
			siblingsMix := map[int][]int{
				0: {0}, 4: {4},
				1: {1, 5}, 5: {1, 5},
				2: {2}, 6: {6},
				3: {3}, 7: {7},
			}
			siblings01 := map[int][]int{
				0: {0, 1}, 1: {0, 1},
				2: {2, 3}, 3: {2, 3},
				4: {4, 5}, 5: {4, 5},
				6: {6, 7}, 7: {6, 7},
			}
			siblings04 := map[int][]int{
				0: {0, 4}, 4: {0, 4},
				1: {1, 5}, 5: {1, 5},
				2: {2, 6}, 6: {2, 6},
				3: {3, 7}, 7: {3, 7},
			}
			siblings0246 := map[int][]int{
				0: {0, 2, 4, 6}, 2: {0, 2, 4, 6}, 4: {0, 2, 4, 6}, 6: {0, 2, 4, 6},
				1: {1, 3, 5, 7}, 3: {1, 3, 5, 7}, 5: {1, 3, 5, 7}, 7: {1, 3, 5, 7},
			}
			testData := []CPUTopologyData{
				// No siblings: expect res, nil
				{cpus: cpus8, siblings: siblings00, res: cpus8},
				{cpus: cpus3, siblings: siblings00, res: cpus3},
				// Mixed siblings/non-siblings 1,5: expect res, nil
				{cpus: cpus8, siblings: siblingsMix, res: []int{0, 1, 5, 2, 3, 4, 6, 7}},
				{cpus: cpus3, siblings: siblingsMix, res: []int{1, 5, 2}},
				// Siblings 0,1 ... 6,7: expect res, nil
				{cpus: cpus8, siblings: siblings01, res: cpus8},
				{cpus: cpus3, siblings: siblings01, res: cpus3},
				// Siblings 0,4 ... 3,7: expect res, nil
				{cpus: cpus8, siblings: siblings04, res: []int{0, 4, 1, 5, 2, 6, 3, 7}},
				{cpus: cpus3, siblings: siblings04, res: []int{1, 5, 2}},
				// Siblings 0,2,4,6 ... 1,2,5,7
				{cpus: cpus8, siblings: siblings0246, res: []int{0, 2, 4, 6, 1, 3, 5, 7}},
				{cpus: cpus3, siblings: siblings0246, res: []int{1, 5, 2}},
				// GetCPUs returns an error: expect nil, err
				{getCPUsErr: errors.New(""), err: true},
				// GetCPUSiblings returns an error: expect res, nil
				{cpus: cpus8, getCPUSiblingsErr: errors.New(""), res: cpus8},
			}
			for _, t := range testData {
				res, err := GroupCPUThreads(t)
				Expect(res).To(Equal(t.res))
				if t.err {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})
	})

	Context("count vCPUs", func() {
		It("shoud count vCPUs correctly", func() {
			vCPUs := GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
				Cores:   2,
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(8)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
			})
			Expect(vCPUs).To(Equal(int64(2)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Cores: 2,
			})
			Expect(vCPUs).To(Equal(int64(2)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(2)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(4)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Sockets: 2,
				Cores:   2,
			})
			Expect(vCPUs).To(Equal(int64(4)), "Expect vCPUs")

			vCPUs = GetNumberOfVCPUs(&v1.CPU{
				Cores:   2,
				Threads: 2,
			})
			Expect(vCPUs).To(Equal(int64(4)), "Expect vCPUs")
		})
	})

	Context("parse PCI address", func() {
		It("shoud return an array of PCI DBSF fields (domain, bus, slot, function) or an error for malformed address", func() {
			testData := []struct {
				addr        string
				expectation []string
			}{
				{"05EA:Fc:1d.6", []string{"05EA", "Fc", "1d", "6"}},
				{"", nil},
				{"invalid address", nil},
				{" 05EA:Fc:1d.6", nil}, // leading symbol
				{"05EA:Fc:1d.6 ", nil}, // trailing symbol
				{"00Z0:00:1d.6", nil},  // invalid digit in domain
				{"0000:z0:1d.6", nil},  // invalid digit in bus
				{"0000:00:Zd.6", nil},  // invalid digit in slot
				{"05EA:Fc:1d:6", nil},  // colon ':' instead of dot '.' after slot
				{"0000:00:1d.9", nil},  // invalid function
			}

			for _, t := range testData {
				res, err := ParsePciAddress(t.addr)
				Expect(res).To(Equal(t.expectation))
				if t.expectation == nil {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})
	})
})
