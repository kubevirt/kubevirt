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
	"fmt"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	archconverter "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

var _ = Describe("Volume conversion", func() {
	type ConverterFunc = func(name string, disk *api.Disk, c *convertertypes.ConverterContext) error

	var c *convertertypes.ConverterContext

	BeforeEach(func() {
		c = &convertertypes.ConverterContext{
			Architecture:   archconverter.NewConverter(runtime.GOARCH),
			AllowEmulation: true,
			IsBlockPVC: map[string]bool{
				"test-block-pvc": true,
			},
			IsBlockDV: map[string]bool{
				"test-block-dv": true,
			},
			VolumesDiscardIgnore: []string{
				"test-discard-ignore",
			},
		}
	})

	DescribeTable("should convert hotplug volume",
		func(converterFunc ConverterFunc, volumeName string, isBlockMode, ignoreDiscard bool) {
			expectedDisk := &api.Disk{}
			expectedDisk.Driver = &api.DiskDriver{}
			expectedDisk.Driver.Type = "raw"
			expectedDisk.Driver.ErrorPolicy = "stop"
			if isBlockMode {
				expectedDisk.Type = "block"
				expectedDisk.Source.Dev = filepath.Join(v1.HotplugDiskDir, volumeName)
			} else {
				expectedDisk.Type = "file"
				expectedDisk.Source.File = fmt.Sprintf("%s.img", filepath.Join(v1.HotplugDiskDir, volumeName))
			}
			if !ignoreDiscard {
				expectedDisk.Driver.Discard = "unmap"
			}

			disk := &api.Disk{
				Driver: &api.DiskDriver{},
			}
			Expect(converterFunc(volumeName, disk, c)).To(Succeed())
			Expect(disk).To(Equal(expectedDisk))
		},
		Entry("filesystem PVC", storage.ConvertV1HotplugPersistentVolumeClaimToAPIDisk, "test-fs-pvc", false, false),
		Entry("block mode PVC", storage.ConvertV1HotplugPersistentVolumeClaimToAPIDisk, "test-block-pvc", true, false),
		Entry("'discard ignore' PVC", storage.ConvertV1HotplugPersistentVolumeClaimToAPIDisk, "test-discard-ignore", false, true),
		Entry("filesystem DV", storage.ConvertV1HotplugDataVolumeToAPIDisk, "test-fs-dv", false, false),
		Entry("block mode DV", storage.ConvertV1HotplugDataVolumeToAPIDisk, "test-block-dv", true, false),
		Entry("'discard ignore' DV", storage.ConvertV1HotplugDataVolumeToAPIDisk, "test-discard-ignore", false, true),
	)
})
