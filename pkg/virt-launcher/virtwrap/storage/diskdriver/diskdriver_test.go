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

package diskdriver

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type configurableIOChecker struct {
	supportDirectIO bool
	err             error
}

func (s *configurableIOChecker) CheckBlockDevice(_ string) (bool, error) {
	return s.supportDirectIO, s.err
}

func (s *configurableIOChecker) CheckFile(_ string) (bool, error) {
	return s.supportDirectIO, s.err
}

var _ = Describe("SetDriverCacheMode", func() {
	newConfigurator := func(supportDirectIO bool, err error) *Configurator {
		return &Configurator{ioChecker: &configurableIOChecker{supportDirectIO: supportDirectIO, err: err}}
	}

	DescribeTable("should correctly set driver cache mode", func(cache, expectedCache string, supportDirectIO bool, checkerErr error) {
		disk := &api.Disk{
			Driver: &api.DiskDriver{
				Cache: cache,
			},
			Source: api.DiskSource{
				File: "file",
			},
		}
		configurator := newConfigurator(supportDirectIO, checkerErr)
		err := configurator.SetDriverCacheMode(disk)
		if expectedCache == "" {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(disk.Driver.Cache).To(Equal(expectedCache))
		}
	},
		Entry("detect 'none' with direct io", "", string(v1.CacheNone), true, nil),
		Entry("detect 'writethrough' without direct io", "", string(v1.CacheWriteThrough), false, nil),
		Entry("fallback to 'writethrough' on error", "", string(v1.CacheWriteThrough), false, fmt.Errorf("DirectIOChecker error")),
		Entry("keep 'none' with direct io", string(v1.CacheNone), string(v1.CacheNone), true, nil),
		Entry("return error without direct io", string(v1.CacheNone), "", false, nil),
		Entry("return error on error", string(v1.CacheNone), "", false, fmt.Errorf("DirectIOChecker error")),
		Entry("'writethrough' with direct io", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), true, nil),
		Entry("'writethrough' without direct io", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), false, nil),
		Entry("'writethrough' on error", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), false, fmt.Errorf("DirectIOChecker error")),
	)

	It("should fail to set appropriate driver cache mode for a nil disk", func() {
		configurator := newConfigurator(true, nil)
		Expect(configurator.SetDriverCacheMode(nil)).To(MatchError("unable to set a driver cache mode, disk is nil"))
	})

	It("should check block device paths correctly", func() {
		disk := &api.Disk{
			Source: api.DiskSource{Dev: "/dev/vda"},
			Driver: &api.DiskDriver{},
		}
		configurator := newConfigurator(true, nil)
		Expect(configurator.SetDriverCacheMode(disk)).To(Succeed())
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
		configurator := newConfigurator(true, nil)
		Expect(configurator.SetDriverCacheMode(disk)).To(Succeed())
	})
})

var _ = Describe("SetOptimalIOMode", func() {
	var origIsPreAllocated func(string) bool

	BeforeEach(func() {
		origIsPreAllocated = IsPreAllocated
	})

	AfterEach(func() {
		IsPreAllocated = origIsPreAllocated
	})

	DescribeTable("should set appropriate IO modes", func(d *api.Disk, expectedIO v1.DriverIO, preAllocated bool) {
		IsPreAllocated = func(_ string) bool { return preAllocated }
		SetOptimalIOMode(d)
		Expect(d.Driver.IO).To(Equal(expectedIO))
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

var _ = Describe("directIOChecker", func() {
	var checker *directIOChecker
	var tmpDir string
	var existingFile string
	var nonExistingFile string
	var err error

	BeforeEach(func() {
		checker = &directIOChecker{}
		tmpDir, err = os.MkdirTemp("", "direct-io-checker")
		Expect(err).ToNot(HaveOccurred())
		existingFile = filepath.Join(tmpDir, "disk.img")
		Expect(os.WriteFile(existingFile, []byte("test"), 0644)).To(Succeed())
		nonExistingFile = filepath.Join(tmpDir, "non-existing-file")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should not fail when file/device exists", func() {
		_, err = checker.CheckFile(existingFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = checker.CheckBlockDevice(existingFile)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not fail when file does not exist", func() {
		_, err := checker.CheckFile(nonExistingFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(nonExistingFile)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})

	It("should fail when device does not exist", func() {
		_, err := checker.CheckBlockDevice(nonExistingFile)
		Expect(err).To(HaveOccurred())
		_, err = os.Stat(nonExistingFile)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})

	It("should fail when the path does not exist", func() {
		nonExistingPath := "/non/existing/path/disk.img"
		_, err = checker.CheckFile(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
		_, err = checker.CheckBlockDevice(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
		_, err = os.Stat(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})
})
