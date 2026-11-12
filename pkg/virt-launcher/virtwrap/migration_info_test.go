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

package virtwrap_test

import (
	"bytes"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	gokitlog "github.com/go-kit/log"
	"libvirt.org/go/libvirt"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

var _ = Describe("Migration info logging", func() {
	var output bytes.Buffer
	var logger *log.FilteredLogger

	BeforeEach(func() {
		output.Reset()
		logger = log.MakeLogger(gokitlog.NewJSONLogger(&output))
		Expect(logger.SetVerbosityLevel(2)).To(Succeed())
	})

	message := func() string {
		var record struct {
			Message string `json:"msg"`
		}
		Expect(json.Unmarshal(output.Bytes(), &record)).To(Succeed())
		return record.Message
	}

	DescribeTable("should report raw counters without inferring a compression ratio",
		func(jobType libvirt.DomainJobType, normalBytes, processedBytes uint64, expectedNormal, expectedProcessed string) {
			virtwrap.LogMigrationInfo(logger, "test-migration", &libvirt.DomainJobInfo{
				Type:              jobType,
				MemNormalBytesSet: true,
				MemNormalBytes:    normalBytes,
				DataProcessedSet:  true,
				DataProcessed:     processedBytes,
				MemConstantSet:    true,
				MemConstant:       0,
			})

			msg := message()
			Expect(msg).To(ContainSubstring("Migration info for test-migration:"))
			Expect(msg).To(ContainSubstring(expectedNormal))
			Expect(msg).To(ContainSubstring(expectedProcessed))
			Expect(msg).To(ContainSubstring("ConstantPages:0"))
			Expect(msg).NotTo(ContainSubstring("CompressionRatio"))
		},
		Entry("with the first uncompressed multifd sample from issue #19736", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(4<<20), uint64(2<<20), "NormalData:4MiB", "DataProcessed:2MiB"),
		Entry("with the second uncompressed multifd sample", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(12<<20), uint64(10<<20), "NormalData:12MiB", "DataProcessed:10MiB"),
		Entry("with the third uncompressed multifd sample", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(16<<20), uint64(14<<20), "NormalData:16MiB", "DataProcessed:14MiB"),
		Entry("with completed migration counters", libvirt.DOMAIN_JOB_COMPLETED, uint64(12<<20), uint64(10<<20), "NormalData:12MiB", "DataProcessed:10MiB"),
		Entry("with equal counters", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(4<<20), uint64(4<<20), "NormalData:4MiB", "DataProcessed:4MiB"),
		Entry("with more processed data than normal bytes", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(4<<20), uint64(8<<20), "NormalData:4MiB", "DataProcessed:8MiB"),
		Entry("before any data is processed", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(4<<20), uint64(0), "NormalData:4MiB", "DataProcessed:0MiB"),
		Entry("before any normal pages are sent", libvirt.DOMAIN_JOB_UNBOUNDED, uint64(0), uint64(0), "NormalData:0MiB", "DataProcessed:0MiB"),
	)

	It("should preserve migration progress and expected downtime", func() {
		virtwrap.LogMigrationInfo(logger, "test-migration", &libvirt.DomainJobInfo{
			TimeElapsed: 4016, DataProcessed: 2 << 20, DataRemaining: 124 << 20, DataTotal: 131 << 20,
			MemProcessed: 0, MemRemaining: 124 << 20, MemTotal: 128 << 20, MemBps: 1000000,
			MemDirtyRate: 1000, MemPageSize: 4096, MemIteration: 1, MemPostcopyReqs: 2,
			MemConstant: 0, MemNormal: 1024, MemNormalBytes: 4 << 20, Downtime: 300, DiskBps: 2000000,
		})

		Expect(message()).To(Equal("Migration info for test-migration: TimeElapsed:4016ms DataProcessed:2MiB DataRemaining:124MiB DataTotal:131MiB " +
			"MemoryProcessed:0MiB MemoryRemaining:124MiB MemoryTotal:128MiB MemoryBandwidth:8Mbps DirtyRate:32Mbps " +
			"Iteration:1 PostcopyRequests:2 ConstantPages:0 NormalPages:1024 NormalData:4MiB ExpectedDowntime:300ms DiskMbps:16"))
	})

	It("should preserve the final downtime and network overhead for completed migrations", func() {
		virtwrap.LogMigrationInfo(logger, "test-migration", &libvirt.DomainJobInfo{
			Type: libvirt.DOMAIN_JOB_COMPLETED, Downtime: 20, DowntimeNetSet: true, DowntimeNet: 8,
		})

		Expect(message()).To(ContainSubstring("Downtime:20ms DowntimeNet:8ms DiskMbps:0"))
		Expect(message()).NotTo(ContainSubstring("ExpectedDowntime"))
	})
})
