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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
)

var _ = Describe("Driver Cache and IO Settings", func() {
	var ctrl *gomock.Controller
	var mockDirectIOChecker *storage.MockDirectIOChecker

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockDirectIOChecker = storage.NewMockDirectIOChecker(ctrl)
	})

	expectCheckTrue := func() {
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(true, nil)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(true, nil)
	}

	expectCheckFalse := func() {
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(false, nil)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(false, nil)
	}

	expectCheckError := func() {
		checkerError := fmt.Errorf("DirectIOChecker error")
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(false, checkerError)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(false, checkerError)
	}

	DescribeTable("should correctly set driver cache mode", func(cache, expectedCache string, setExpectations func()) {
		disk := &api.Disk{
			Driver: &api.DiskDriver{
				Cache: cache,
			},
			Source: api.DiskSource{
				File: "file",
			},
		}
		setExpectations()
		err := storage.SetDriverCacheMode(disk, mockDirectIOChecker)
		if expectedCache == "" {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(disk.Driver.Cache).To(Equal(expectedCache))
		}
	},
		Entry("detect 'none' with direct io", "", string(v1.CacheNone), expectCheckTrue),
		Entry("detect 'writethrough' without direct io", "", string(v1.CacheWriteThrough), expectCheckFalse),
		Entry("fallback to 'writethrough' on error", "", string(v1.CacheWriteThrough), expectCheckError),
		Entry("keep 'none' with direct io", string(v1.CacheNone), string(v1.CacheNone), expectCheckTrue),
		Entry("return error without direct io", string(v1.CacheNone), "", expectCheckFalse),
		Entry("return error on error", string(v1.CacheNone), "", expectCheckError),
		Entry("'writethrough' with direct io", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), expectCheckTrue),
		Entry("'writethrough' without direct io", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), expectCheckFalse),
		Entry("'writethrough' on error", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), expectCheckError),
	)

	It("should fail to set appropriate driver cache mode for a nil disk", func() {
		Expect(storage.SetDriverCacheMode(nil, nil)).To(MatchError("unable to set a driver cache mode, disk is nil"))
	})

	It("should check block device paths correctly", func() {
		disk := &api.Disk{
			Source: api.DiskSource{Dev: "/dev/vda"},
			Driver: &api.DiskDriver{},
		}
		mockDirectIOChecker.EXPECT().CheckBlockDevice("/dev/vda").Return(true, nil)

		Expect(storage.SetDriverCacheMode(disk, mockDirectIOChecker)).To(Succeed())
		Expect(disk.Driver.Cache).To(Equal(string(v1.CacheNone)))
	})

	It("should resolve datastore block dev over frontend file source", func() {
		disk := &api.Disk{
			Source: api.DiskSource{
				File: "/test/overlay.qcow2",
				DataStore: &api.DataStore{
					Source: &api.DiskSource{Dev: "/dev/vda"},
				},
			},
			Driver: &api.DiskDriver{},
		}
		mockDirectIOChecker.EXPECT().CheckBlockDevice("/dev/vda").Return(true, nil)

		Expect(storage.SetDriverCacheMode(disk, mockDirectIOChecker)).To(Succeed())
	})
	DescribeTable("should set appropriate IO modes", func(disk *api.Disk, expectedIO v1.DriverIO, isPreAllocated bool) {
		storage.SetOptimalIOMode(disk, func(path string) bool { return isPreAllocated })
		Expect(disk.Driver.IO).To(Equal(expectedIO))
	},
		Entry("user-specified IO",
			&api.Disk{Driver: &api.DiskDriver{IO: v1.IOThreads}},
			v1.IOThreads, false,
		),
		Entry("sparse image",
			&api.Disk{Source: api.DiskSource{File: "test.img"}, Driver: &api.DiskDriver{}},
			v1.DriverIO(""), false,
		),
		Entry("pre-allocated image with O_DIRECT",
			&api.Disk{Source: api.DiskSource{File: "test.img"}, Driver: &api.DiskDriver{Cache: string(v1.CacheNone)}},
			v1.IONative, true,
		),
		Entry("pre-allocated image without O_DIRECT",
			&api.Disk{Source: api.DiskSource{File: "test.img"}, Driver: &api.DiskDriver{Cache: string(v1.CacheWriteThrough)}},
			v1.DriverIO(""), true,
		),
		Entry("block device with O_DIRECT",
			&api.Disk{Source: api.DiskSource{Dev: "/dev/test"}, Driver: &api.DiskDriver{Cache: string(v1.CacheNone)}},
			v1.IONative, true,
		),
		Entry("datastore block device with O_DIRECT",
			&api.Disk{
				Source: api.DiskSource{
					File: "/test/overlay.qcow2",
					DataStore: &api.DataStore{
						Type:   "block",
						Source: &api.DiskSource{Dev: "/dev/vda"},
					},
				},
				Driver: &api.DiskDriver{Cache: string(v1.CacheNone)},
			},
			v1.IONative, false,
		),
		Entry("datastore block device without O_DIRECT",
			&api.Disk{
				Source: api.DiskSource{
					File: "/test/overlay.qcow2",
					DataStore: &api.DataStore{
						Type:   "block",
						Source: &api.DiskSource{Dev: "/dev/vda"},
					},
				},
				Driver: &api.DiskDriver{Cache: string(v1.CacheWriteThrough)},
			},
			v1.DriverIO(""), false,
		),
		Entry("datastore file backend pre-allocated with O_DIRECT",
			&api.Disk{
				Source: api.DiskSource{
					File: "/test/overlay.qcow2",
					DataStore: &api.DataStore{
						Type:   "file",
						Source: &api.DiskSource{File: "/disks/disk.img"},
					},
				},
				Driver: &api.DiskDriver{Cache: string(v1.CacheNone)},
			},
			v1.IONative, true,
		),
		Entry("datastore file backend sparse with O_DIRECT",
			&api.Disk{
				Source: api.DiskSource{
					File: "/test/overlay.qcow2",
					DataStore: &api.DataStore{
						Type:   "file",
						Source: &api.DiskSource{File: "/disks/disk.img"},
					},
				},
				Driver: &api.DiskDriver{Cache: string(v1.CacheNone)},
			},
			v1.DriverIO(""), false,
		),
	)
})
