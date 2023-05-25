package downwardmetrics

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hostmetrics", func() {

	It("should interpret the proc files as expected", func() {
		hostmetrics := &hostMetricsCollector{
			procCPUInfo: "testdata/cpuinfo",
			procStat:    "testdata/stat",
			procMemInfo: "testdata/meminfo",
			procVMStat:  "testdata/vmstat",
			pageSize:    4096,
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

	DescribeTable("should cope with failed reads on stats files and return what it can get for", func(cpuinfo, meminfo, stat, vmstat string, count int) {
		hostmetrics := &hostMetricsCollector{
			procCPUInfo: cpuinfo,
			procStat:    meminfo,
			procMemInfo: stat,
			procVMStat:  vmstat,
			pageSize:    4096,
		}

		metrics := hostmetrics.Collect()
		Expect(metrics).To(HaveLen(count))

	},
		Entry("cpuinfo", "nonexistent", "testdata/meminfo", "testdata/stat", "testdata/vmstat", 8),
		Entry("meminfo", "testdata/cpuinfo", "nonexistent", "testdata/stat", "testdata/vmstat", 8),
		Entry("stat", "testdata/cpuinfo", "testdata/meminfo", "nonexistent", "testdata/vmstat", 5),
		Entry("vmstat", "testdata/cpuinfo", "testdata/meminfo", "testdata/stat", "nonexistent", 7),
	)

})
