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

package disksource_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
)

var _ = Describe("DiskTopology", func() {
	DescribeTable("should resolve topology correctly",
		func(disk api.Disk, expectedSource, expectedBackend string, expectedIsBlock, expectedHasOverlay bool) {
			ds := disksource.Resolve(disk)
			Expect(ds.SourcePath()).To(Equal(expectedSource))
			Expect(ds.BackendPath()).To(Equal(expectedBackend))
			Expect(ds.BackendIsBlock()).To(Equal(expectedIsBlock))
			Expect(ds.HasOverlay()).To(Equal(expectedHasOverlay))
		},
		Entry("plain file disk",
			api.Disk{
				Source: api.DiskSource{
					File: "/test/disk.img",
				},
			},
			"/test/disk.img",
			"/test/disk.img",
			false,
			false,
		),
		Entry("plain block device",
			api.Disk{
				Source: api.DiskSource{
					Dev: "/dev/vda",
				},
			},
			"/dev/vda",
			"/dev/vda",
			true,
			false,
		),
		Entry("qcow2 overlay on block device via datastore",
			api.Disk{
				Source: api.DiskSource{
					File: "/test/overlay.qcow2",
					DataStore: &api.DataStore{
						Type:   "block",
						Source: &api.DiskSource{Dev: "/dev/vda"},
					},
				},
			},
			"/test/overlay.qcow2",
			"/dev/vda",
			true,
			true,
		),
		Entry("qcow2 overlay on file via datastore",
			api.Disk{
				Source: api.DiskSource{
					File: "/test/overlay.qcow2",
					DataStore: &api.DataStore{
						Type:   "file",
						Source: &api.DiskSource{File: "/test/disk.img"},
					},
				},
			},
			"/test/overlay.qcow2",
			"/test/disk.img",
			false,
			true,
		),
		Entry("empty disk",
			api.Disk{},
			"",
			"",
			false,
			false,
		),
		Entry("datastore with nil source",
			api.Disk{
				Source: api.DiskSource{
					File:      "/test/overlay.qcow2",
					DataStore: &api.DataStore{Type: "block"},
				},
			},
			"/test/overlay.qcow2",
			"/test/overlay.qcow2",
			false,
			false,
		),
	)

	DescribeTable("should detect hotplug disks correctly",
		func(disk api.Disk, expectedHotplug, expectedHotplugOrEmpty bool) {
			ds := disksource.Resolve(disk)
			Expect(ds.IsHotplugDisk()).To(Equal(expectedHotplug))
			Expect(ds.IsHotplugOrEmpty()).To(Equal(expectedHotplugOrEmpty))
		},
		Entry("hotplug disk",
			api.Disk{
				Source: api.DiskSource{
					File: filepath.Join(v1.HotplugDiskDir, "disk.img"),
				},
			},
			true,
			true,
		),
		Entry("non-hotplug disk",
			api.Disk{
				Source: api.DiskSource{
					File: "/test/disk.img",
				},
			},
			false,
			false,
		),
		Entry("empty disk",
			api.Disk{},
			false,
			true,
		),
	)
})
