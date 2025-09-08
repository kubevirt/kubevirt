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

package downwardmetrics

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hostmetrics", func() {

	var (
		tempSysDir string
	)

	BeforeEach(func() {
		tempSysDir = GinkgoT().TempDir()

		type topology struct {
			coreId             string
			physicalPackageId  string
			coreSiblingsList   string
			threadSiblingsList string
		}

		for i, cpuTopology := range []topology{{
			coreId:             "0",
			physicalPackageId:  "0",
			coreSiblingsList:   "0-5",
			threadSiblingsList: "0,4",
		}, {
			coreId:             "1",
			physicalPackageId:  "0",
			coreSiblingsList:   "0-5",
			threadSiblingsList: "1,5",
		}, {
			coreId:             "2",
			physicalPackageId:  "0",
			coreSiblingsList:   "0-5",
			threadSiblingsList: "2",
		}, {
			coreId:             "3",
			physicalPackageId:  "0",
			coreSiblingsList:   "0-5",
			threadSiblingsList: "3",
		}, {
			coreId:             "0",
			physicalPackageId:  "0",
			coreSiblingsList:   "0-5",
			threadSiblingsList: "1,4",
		}, {
			coreId:             "1",
			physicalPackageId:  "0",
			coreSiblingsList:   "0-5",
			threadSiblingsList: "2,5",
		}, {
			coreId:             "2",
			physicalPackageId:  "1",
			coreSiblingsList:   "6",
			threadSiblingsList: "6",
		}, {
			coreId:             "3",
			physicalPackageId:  "2",
			coreSiblingsList:   "7",
			threadSiblingsList: "7",
		}} {
			topologyDir := filepath.Join(tempSysDir, "devices", "system", "cpu",
				fmt.Sprintf("cpu%d", i), "topology")
			Expect(os.MkdirAll(topologyDir, os.ModePerm)).To(Succeed())

			Expect(os.WriteFile(filepath.Join(topologyDir, "core_id"), []byte(cpuTopology.coreId), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(topologyDir, "physical_package_id"), []byte(cpuTopology.physicalPackageId), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(topologyDir, "core_siblings_list"), []byte(cpuTopology.coreSiblingsList), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(topologyDir, "thread_siblings_list"), []byte(cpuTopology.threadSiblingsList), os.ModePerm)).To(Succeed())
		}
	})

	It("should interpret the proc and sys files as expected", func() {
		hostmetrics := &hostMetricsCollector{
			procPath: "testdata",
			sysPath:  tempSysDir,
			pageSize: 4096,
		}

		metrics := hostmetrics.Collect()

		Expect(metrics).To(HaveLen(9))
		Expect(metrics[0].Name).To(Equal("NumberOfPhysicalCPUs"))
		Expect(metrics[0].Unit).To(Equal(""))
		Expect(metrics[0].Value).To(Equal("3"))
		Expect(metrics[1].Name).To(Equal("TotalCPUTime"))
		Expect(metrics[1].Unit).To(Equal("s"))
		Expect(metrics[1].Value).To(Equal("267804.880000"))
		Expect(metrics[2].Name).To(Equal("FreePhysicalMemory"))
		Expect(metrics[2].Unit).To(Equal("KiB"))
		Expect(metrics[2].Value).To(Equal("2435476"))
		Expect(metrics[3].Name).To(Equal("FreeVirtualMemory"))
		Expect(metrics[3].Unit).To(Equal("KiB"))
		Expect(metrics[3].Value).To(Equal("19563768"))
		Expect(metrics[4].Name).To(Equal("MemoryAllocatedToVirtualServers"))
		Expect(metrics[4].Unit).To(Equal("KiB"))
		Expect(metrics[4].Value).To(Equal("8002064"))
		Expect(metrics[5].Name).To(Equal("UsedVirtualMemory"))
		Expect(metrics[5].Unit).To(Equal("KiB"))
		Expect(metrics[5].Value).To(Equal("30836704"))
		Expect(metrics[6].Name).To(Equal("PagedInMemory"))
		Expect(metrics[6].Unit).To(Equal("KiB"))
		Expect(metrics[6].Value).To(Equal("17254016"))
		Expect(metrics[7].Name).To(Equal("PagedOutMemory"))
		Expect(metrics[7].Unit).To(Equal("KiB"))
		Expect(metrics[7].Value).To(Equal("27252776"))
		Expect(metrics[8].Name).To(Equal("Time"))
		Expect(metrics[8].Unit).To(Equal("s"))
	})

	Context("with testdata copy", func() {
		var tempDir string

		const (
			memInfoFile = "meminfo"
			statFile    = "stat"
			vmStatFile  = "vmstat"
		)

		BeforeEach(func() {
			testBaseDir, err := filepath.Abs("testdata")
			Expect(err).ToNot(HaveOccurred())

			tempDir = GinkgoT().TempDir()
			Expect(os.Symlink(filepath.Join(testBaseDir, memInfoFile), filepath.Join(tempDir, memInfoFile))).To(Succeed())
			Expect(os.Symlink(filepath.Join(testBaseDir, statFile), filepath.Join(tempDir, statFile))).To(Succeed())
			Expect(os.Symlink(filepath.Join(testBaseDir, vmStatFile), filepath.Join(tempDir, vmStatFile))).To(Succeed())
		})

		DescribeTable("should cope with missing", func(fileToRemove string, count int) {
			Expect(os.Remove(filepath.Join(tempDir, fileToRemove))).To(Succeed())

			hostmetrics := &hostMetricsCollector{
				procPath: tempDir,
				sysPath:  tempSysDir,
				pageSize: 4096,
			}
			metrics := hostmetrics.Collect()
			Expect(metrics).To(HaveLen(count))
		},
			Entry("meminfo", memInfoFile, 5),
			Entry("stat", statFile, 8),
			Entry("vmstat", vmStatFile, 7),
		)

		It("should cope with missing sys directory", func() {
			Expect(os.RemoveAll(tempSysDir)).To(Succeed())

			hostmetrics := &hostMetricsCollector{
				procPath: tempDir,
				sysPath:  tempSysDir,
				pageSize: 4096,
			}
			metrics := hostmetrics.Collect()
			Expect(metrics).To(HaveLen(8))
		})
	})

	It("should parse vmstat correctly", func() {
		vmstat, err := readVMStat("testdata/vmstat")
		Expect(err).ToNot(HaveOccurred())

		Expect(vmstat.pswpin).To(Equal(uint64(4313504)), "pswpin not loaded correctly")
		Expect(vmstat.pswpout).To(Equal(uint64(6813194)), "pswpout not loaded correctly")
	})
})
