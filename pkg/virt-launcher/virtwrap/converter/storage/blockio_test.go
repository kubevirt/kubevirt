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

package storage_test

import (
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
)

const (
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

var _ = Describe("Convert_v1_BlockSize_To_api_BlockIO", func() {
	DescribeTable("Should handle custom block sizes correctly per architecture", func(arch string, logical, physical uint, shouldSucceed bool) {
		kubevirtDisk := &v1.Disk{
			BlockSize: &v1.BlockSize{
				Custom: &v1.CustomBlockSize{
					Logical:            logical,
					Physical:           physical,
					DiscardGranularity: pointer.P(physical),
				},
			},
		}
		libvirtDisk := &api.Disk{}
		err := storage.Convert_v1_BlockSize_To_api_BlockIO(kubevirtDisk, libvirtDisk, arch)
		if shouldSucceed {
			Expect(err).ToNot(HaveOccurred())
			expectedXML := fmt.Sprintf(`<Disk device="" type="">
  <source></source>
  <target></target>
  <blockio logical_block_size="%d" physical_block_size="%d" discard_granularity="%d"></blockio>
</Disk>`, logical, physical, physical)
			data, xmlErr := xml.MarshalIndent(libvirtDisk, "", "  ")
			Expect(xmlErr).ToNot(HaveOccurred())
			Expect(string(data)).To(Equal(expectedXML))
		} else {
			Expect(err).To(MatchError(ContainSubstring("exceeds the maximum supported size")))
		}
	},
		Entry("valid 1234 on amd64", amd64, uint(1234), uint(1234), true),
		Entry("valid 1234 on arm64", arm64, uint(1234), uint(1234), true),
		Entry("valid 1234 on s390x", s390x, uint(1234), uint(1234), true),
		Entry("4096 on s390x", s390x, uint(4096), uint(4096), true),
		Entry("2048 on s390x", s390x, uint(2048), uint(2048), true),
		Entry("1024 on s390x", s390x, uint(1024), uint(1024), true),
		Entry("8192 on s390x", s390x, uint(8192), uint(8192), false),
		Entry("65536 on s390x", s390x, uint(65536), uint(65536), false),
		Entry("1 MiB on s390x", s390x, uint(1048576), uint(1048576), false),
	)

	It("Should detect disk block sizes for a file DiskSource", func() {
		v1Disk := v1.Disk{
			Name: "test",
			BlockSize: &v1.BlockSize{
				MatchVolume: &v1.FeatureState{Enabled: pointer.P(true)},
			},
		}
		apiDisk := api.Disk{Source: api.DiskSource{File: "/"}}
		Expect(storage.Convert_v1_BlockSize_To_api_BlockIO(&v1Disk, &apiDisk, amd64)).To(Succeed())

		blockIO := apiDisk.BlockIO
		Expect(blockIO.LogicalBlockSize).To(Equal(blockIO.PhysicalBlockSize))
		Expect(blockIO.LogicalBlockSize).ToNot(BeZero())
		Expect(blockIO.DiscardGranularity).ToNot(BeNil())
		Expect(*blockIO.DiscardGranularity).To(Equal(blockIO.LogicalBlockSize))
	})

	It("Should fail for non-file or non-block devices", func() {
		const blockIoConfigErrorMessage = "failed to configure disk with block size detection enabled"
		v1Disk := v1.Disk{
			Name: "test",
			BlockSize: &v1.BlockSize{
				MatchVolume: &v1.FeatureState{Enabled: pointer.P(true)},
			},
		}
		apiDisk := api.Disk{Source: api.DiskSource{}}
		Expect(storage.Convert_v1_BlockSize_To_api_BlockIO(&v1Disk, &apiDisk, amd64)).To(MatchError(ContainSubstring(blockIoConfigErrorMessage)))
	})

	It("Should fail block size detection for a nil domain disk", func() {
		const nilDiskErrorMessage = "disk is nil"
		v1Disk := v1.Disk{
			Name: "test",
			BlockSize: &v1.BlockSize{
				MatchVolume: &v1.FeatureState{Enabled: pointer.P(true)},
			},
		}
		Expect(storage.Convert_v1_BlockSize_To_api_BlockIO(&v1Disk, nil, amd64)).To(MatchError(ContainSubstring(nilDiskErrorMessage)))
	})
})
