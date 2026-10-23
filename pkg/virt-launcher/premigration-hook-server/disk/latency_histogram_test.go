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
 */

package disk_test

import (
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LatencyHistogramHook", func() {
	It("should return without changes when the domain has no devices", func() {
		domain := &libvirtxml.Domain{}

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())
		Expect(domain.Devices).To(BeNil())
	})

	It("should skip disks without a driver", func() {
		domain := &libvirtxml.Domain{
			Devices: &libvirtxml.DomainDeviceList{
				Disks: []libvirtxml.DomainDisk{
					{},
				},
			},
		}

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].Driver).To(BeNil())
	})

	It("should add default latency histograms to an unconfigured disk", func() {
		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Name: "qemu",
		})

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())

		driver := domain.Devices.Disks[0].Driver
		Expect(driver.Statistics).ToNot(BeNil())

		histograms := driver.Statistics.LatencyHistogram
		Expect(histograms).To(HaveLen(3))
		Expect(histogramTypes(histograms)).To(Equal(
			[]string{"read", "write", "flush"},
		))

		expectedStarts := []uint{
			0,
			10_000_000,
			50_000_000,
			100_000_000,
			500_000_000,
			1_000_000_000,
		}

		for _, histogram := range histograms {
			Expect(histogramBinStarts(histogram)).To(Equal(expectedStarts))
		}

		domainXML, err := domain.Marshal()
		Expect(err).ToNot(HaveOccurred())
		Expect(domainXML).To(ContainSubstring(
			`<latency-histogram type="read">`,
		))
		Expect(domainXML).To(ContainSubstring(
			`<latency-histogram type="write">`,
		))
		Expect(domainXML).To(ContainSubstring(
			`<latency-histogram type="flush">`,
		))
	})

	It("should be idempotent", func() {
		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Name: "qemu",
		})

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())
		configuredXML, err := domain.Marshal()
		Expect(err).ToNot(HaveOccurred())

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())
		updatedXML, err := domain.Marshal()
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedXML).To(Equal(configuredXML))

		histograms := domain.Devices.Disks[0].
			Driver.Statistics.LatencyHistogram

		Expect(histograms).To(HaveLen(3))
		Expect(histogramTypes(histograms)).To(Equal(
			[]string{"read", "write", "flush"},
		))
	})

	It("should replace mismatched bins and add missing operations without changing other statistics", func() {
		existingRead := libvirtxml.DomainDiskLatencyHistogram{
			Type: "read",
			Bin: []libvirtxml.DomainDiskLatencyHistogramBin{
				{Start: 0},
				{Start: 42},
			},
		}

		existingStatistic := libvirtxml.DomainDiskStatistic{
			Interval: 10,
		}

		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Name: "qemu",
			Statistics: &libvirtxml.DomainDiskStatistics{
				Statistic: []libvirtxml.DomainDiskStatistic{
					existingStatistic,
				},
				LatencyHistogram: []libvirtxml.DomainDiskLatencyHistogram{
					existingRead,
				},
			},
		})

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())

		statistics := domain.Devices.Disks[0].Driver.Statistics
		Expect(statistics.Statistic).To(Equal(
			[]libvirtxml.DomainDiskStatistic{existingStatistic},
		))

		histograms := statistics.LatencyHistogram
		Expect(histograms).To(HaveLen(3))
		Expect(histograms[0].Type).To(Equal(existingRead.Type))
		for _, histogram := range histograms {
			Expect(histogram.Bin).To(Equal(expectedLatencyHistogramBins()))
		}
		Expect(histogramTypes(histograms)).To(Equal(
			[]string{"read", "write", "flush"},
		))
	})

	It("should normalize an untyped histogram without adding redundant operation-specific histograms", func() {
		allOperationsHistogram := libvirtxml.DomainDiskLatencyHistogram{
			Bin: []libvirtxml.DomainDiskLatencyHistogramBin{
				{Start: 0},
				{Start: 1_000_000},
			},
		}

		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Name: "qemu",
			Statistics: &libvirtxml.DomainDiskStatistics{
				LatencyHistogram: []libvirtxml.DomainDiskLatencyHistogram{
					allOperationsHistogram,
				},
			},
		})

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())

		histograms := domain.Devices.Disks[0].
			Driver.Statistics.LatencyHistogram

		Expect(histograms).To(HaveLen(1))
		Expect(histograms[0].Type).To(BeEmpty())
		Expect(histograms[0].Bin).To(Equal(expectedLatencyHistogramBins()))
	})

	DescribeTable("should normalize typed overrides regardless of the untyped histogram's position", func(untypedFirst bool) {
		untyped := libvirtxml.DomainDiskLatencyHistogram{
			Bin: expectedLatencyHistogramBins(),
		}
		read := libvirtxml.DomainDiskLatencyHistogram{
			Type: "read",
			Bin: []libvirtxml.DomainDiskLatencyHistogramBin{
				{Start: 0},
				{Start: 42},
			},
		}
		histograms := []libvirtxml.DomainDiskLatencyHistogram{read, untyped}
		if untypedFirst {
			histograms[0], histograms[1] = histograms[1], histograms[0]
		}

		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Statistics: &libvirtxml.DomainDiskStatistics{
				LatencyHistogram: histograms,
			},
		})

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())

		histograms = domain.Devices.Disks[0].Driver.Statistics.LatencyHistogram
		Expect(histograms).To(HaveLen(2))
		for _, histogram := range histograms {
			Expect(histogram.Bin).To(Equal(expectedLatencyHistogramBins()))
		}
	},
		Entry("untyped histogram first", true),
		Entry("untyped histogram last", false),
	)

	It("should preserve histograms for operations outside the read/write/flush contract", func() {
		zone := libvirtxml.DomainDiskLatencyHistogram{
			Type: "zone",
			Bin: []libvirtxml.DomainDiskLatencyHistogramBin{
				{Start: 0},
				{Start: 42},
			},
		}
		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Statistics: &libvirtxml.DomainDiskStatistics{
				LatencyHistogram: []libvirtxml.DomainDiskLatencyHistogram{zone},
			},
		})

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())

		histograms := domain.Devices.Disks[0].Driver.Statistics.LatencyHistogram
		Expect(histograms).To(HaveLen(4))
		Expect(histograms[0]).To(Equal(zone))
	})

	It("should configure an empty CD-ROM on the migration target", func() {
		domain := domainWithDriver(&libvirtxml.DomainDiskDriver{
			Name: "qemu",
			Type: "raw",
		})
		domain.Devices.Disks[0].Device = "cdrom"

		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())
		Expect(domain.Devices.Disks[0].Source).To(BeNil())

		histograms := domain.Devices.Disks[0].Driver.Statistics.LatencyHistogram
		Expect(histogramTypes(histograms)).To(Equal(
			[]string{"read", "write", "flush"},
		))
		for _, histogram := range histograms {
			Expect(histogram.Bin).To(Equal(expectedLatencyHistogramBins()))
		}
	})
})

func expectedLatencyHistogramBins() []libvirtxml.DomainDiskLatencyHistogramBin {
	return []libvirtxml.DomainDiskLatencyHistogramBin{
		{Start: 0},
		{Start: 10_000_000},
		{Start: 50_000_000},
		{Start: 100_000_000},
		{Start: 500_000_000},
		{Start: 1_000_000_000},
	}
}

func domainWithDriver(
	driver *libvirtxml.DomainDiskDriver,
) *libvirtxml.Domain {
	return &libvirtxml.Domain{
		Devices: &libvirtxml.DomainDeviceList{
			Disks: []libvirtxml.DomainDisk{
				{
					Driver: driver,
				},
			},
		},
	}
}

func histogramTypes(
	histograms []libvirtxml.DomainDiskLatencyHistogram,
) []string {
	types := make([]string, len(histograms))

	for i, histogram := range histograms {
		types[i] = histogram.Type
	}

	return types
}

func histogramBinStarts(
	histogram libvirtxml.DomainDiskLatencyHistogram,
) []uint {
	starts := make([]uint, len(histogram.Bin))

	for i, bin := range histogram.Bin {
		starts[i] = bin.Start
	}

	return starts
}
