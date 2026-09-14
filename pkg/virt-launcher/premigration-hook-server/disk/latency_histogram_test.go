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
			1_000_000,
			10_000_000,
			50_000_000,
			100_000_000,
			500_000_000,
			1_000_000_000,
			2_000_000_000,
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
		Expect(disk.LatencyHistogramHook(nil, nil, domain)).To(Succeed())

		histograms := domain.Devices.Disks[0].
			Driver.Statistics.LatencyHistogram

		Expect(histograms).To(HaveLen(3))
		Expect(histogramTypes(histograms)).To(Equal(
			[]string{"read", "write", "flush"},
		))
	})

	It("should preserve existing histograms and add missing operations", func() {
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
		Expect(histograms[0]).To(Equal(existingRead))
		Expect(histogramTypes(histograms)).To(Equal(
			[]string{"read", "write", "flush"},
		))
	})

	It("should not add operation-specific histograms when an untyped histogram exists", func() {
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

		Expect(histograms).To(Equal(
			[]libvirtxml.DomainDiskLatencyHistogram{
				allOperationsHistogram,
			},
		))
	})
})

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
