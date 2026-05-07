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

package cgroup

import (
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing/fstest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/runc/libcontainer/devices"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"
	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("generateMacvtapDeviceRules", func() {
	var (
		ctrl         *gomock.Controller
		vmi          *virtv1.VirtualMachineInstance
		mockSafePath *MockSafePath
		mockDevPath  *MockSafePath
		mockTapPath  *MockSafePath
		mapFS        fstest.MapFS
		fsys         fs.ReadDirFS
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		vmi = &virtv1.VirtualMachineInstance{}

		mockSafePath = NewMockSafePath(ctrl)
		mockDevPath = NewMockSafePath(ctrl)
		mockTapPath = NewMockSafePath(ctrl)

		mapFS = fstest.MapFS{
			strings.TrimPrefix(procDevicesPath, "/"): &fstest.MapFile{Data: []byte("244 macvtap\n")}, // macvtap canonical major number is 237, but intentionally use 244 here to ensure the code dynamically gets macvtap's major number instead of hardcoding
			"tmp/dev/tap0":                           &fstest.MapFile{Data: []byte("")},
		}
		fsys = stripSlashFS{mapFS}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return specific rules when macvtap devices are found", func() {
		mockSafePath.EXPECT().JoinNoFollow("dev").Return(mockDevPath, nil)
		mockDevPath.EXPECT().ExecuteNoFollow(gomock.Any()).DoAndReturn(func(f func(string) error) error {
			return f("tmp/dev")
		})
		mockDevPath.EXPECT().JoinNoFollow("tap0").Return(mockTapPath, nil)

		mockTapPath.EXPECT().StatAtNoFollow().Return(mockFileInfo{mode: os.ModeDevice | os.ModeCharDevice, rdev: unix.Mkdev(244, 0)}, nil)

		rules, err := generateMacvtapDeviceRules(vmi, mockSafePath, fsys)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rules).To(HaveLen(1))
		Expect(rules[0].Type).To(Equal(devices.CharDevice))
		Expect(rules[0].Major).To(Equal(int64(244)))
		Expect(rules[0].Minor).To(Equal(int64(0)))
	})

	It("should return specific rules when 2 macvtap devices are found", func() {
		mapFS["tmp/dev/tap1"] = &fstest.MapFile{Data: []byte("")}

		mockSafePath.EXPECT().JoinNoFollow("dev").Return(mockDevPath, nil)
		mockDevPath.EXPECT().ExecuteNoFollow(gomock.Any()).DoAndReturn(func(f func(string) error) error {
			return f("tmp/dev")
		})

		mockTapPath2 := NewMockSafePath(ctrl)
		mockDevPath.EXPECT().JoinNoFollow("tap0").Return(mockTapPath, nil)
		mockDevPath.EXPECT().JoinNoFollow("tap1").Return(mockTapPath2, nil)

		mockTapPath.EXPECT().StatAtNoFollow().Return(mockFileInfo{mode: os.ModeDevice | os.ModeCharDevice, rdev: unix.Mkdev(244, 0)}, nil)
		mockTapPath2.EXPECT().StatAtNoFollow().Return(mockFileInfo{mode: os.ModeDevice | os.ModeCharDevice, rdev: unix.Mkdev(244, 1)}, nil)

		rules, err := generateMacvtapDeviceRules(vmi, mockSafePath, fsys)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rules).To(HaveLen(2))
		Expect(rules[0].Major).To(Equal(int64(244)))
		Expect(rules[1].Major).To(Equal(int64(244)))
	})

	It("should return nil if no macvtap devices are found in /dev", func() {
		delete(mapFS, "tmp/dev/tap0")
		mapFS["tmp/dev/not-a-tap"] = &fstest.MapFile{Data: []byte("")}

		mockSafePath.EXPECT().JoinNoFollow("dev").Return(mockDevPath, nil)
		mockDevPath.EXPECT().ExecuteNoFollow(gomock.Any()).DoAndReturn(func(f func(string) error) error {
			return f("tmp/dev")
		})

		rules, err := generateMacvtapDeviceRules(vmi, mockSafePath, fsys)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rules).To(BeNil())
	})

	It("should return error if getDeviceMajor fails", func() {
		delete(mapFS, "proc/devices")
		_, err := generateMacvtapDeviceRules(vmi, mockSafePath, fsys)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("macvtap device major number not found"))
	})

	It("should return nil if macvtap device is not found in proc/devices", func() {
		mapFS["proc/devices"] = &fstest.MapFile{Data: []byte(" 1 memory\n")}
		rules, err := generateMacvtapDeviceRules(vmi, mockSafePath, fsys)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rules).To(BeNil())
	})

	It("should return error if ReadDir fails", func() {
		mockSafePath.EXPECT().JoinNoFollow("dev").Return(mockDevPath, nil)
		mockDevPath.EXPECT().ExecuteNoFollow(gomock.Any()).DoAndReturn(func(f func(string) error) error {
			return f("tmp/nonexistent")
		})

		_, err := generateMacvtapDeviceRules(vmi, mockSafePath, fsys)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to list files in directory"))
	})

})

var _ = Describe("getDeviceMajor", func() {
	var (
		mapFS fstest.MapFS
		fsys  fs.ReadDirFS
	)

	BeforeEach(func() {
		mapFS = fstest.MapFS{
			strings.TrimPrefix(procDevicesPath, "/"): &fstest.MapFile{Data: []byte("244 macvtap\n")},
		}
		fsys = stripSlashFS{mapFS}
	})

	It("should return major number when device is found", func() {
		major, err := getDeviceMajor("macvtap", fsys)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(major).To(Equal(244))
	})

	It("should return error when device is not found", func() {
		_, err := getDeviceMajor("nonexistent", fsys)
		Expect(err).Should(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&errDeviceNotFound{}))
	})

	It("should return error when file cannot be opened", func() {
		delete(mapFS, strings.TrimPrefix(procDevicesPath, "/"))
		_, err := getDeviceMajor("macvtap", fsys)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("file does not exist"))
	})

	It("should return error when major number is not an integer", func() {
		mapFS[strings.TrimPrefix(procDevicesPath, "/")] = &fstest.MapFile{Data: []byte("abc macvtap\n")}
		_, err := getDeviceMajor("macvtap", fsys)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to parse major number"))
	})
})

// stripSlashFS wraps around MapFS to support absolute path by stripping the leading slash before passing the name to MapFS.
type stripSlashFS struct {
	fstest.MapFS
}

func (s stripSlashFS) Open(name string) (fs.File, error) {
	return s.MapFS.Open(strings.TrimPrefix(name, "/"))
}

func (s stripSlashFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return s.MapFS.ReadDir(strings.TrimPrefix(name, "/"))
}

type mockFileInfo struct {
	mode os.FileMode
	rdev uint64
}

func (m mockFileInfo) Name() string       { return "" }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.mode.IsDir() }
func (m mockFileInfo) Sys() interface{}   { return &syscall.Stat_t{Rdev: m.rdev} }
