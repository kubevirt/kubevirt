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
	"path"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	cgroups "github.com/opencontainers/cgroups"
	devices "github.com/opencontainers/cgroups/devices/config"
	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/safepath"
)

var _ = Describe("cgroup manager", func() {

	var (
		ctrl               *gomock.Controller
		rulesDefined       []*devices.Rule
		v2DirPath          string
		cgroupPathsDefined []string
	)

	newMockManager := func() (Manager, error) {
		mockCgroupsManager := NewMockcgroupsManager(ctrl)
		mockCgroupsManager.EXPECT().GetPaths().DoAndReturn(func() map[string]string {
			return map[string]string{"": v2DirPath}
		}).AnyTimes()

		execVirtChrootFunc := func(r *cgroups.Resources, cgroupPaths []string, rootless bool) error {
			rulesDefined = r.Devices
			cgroupPathsDefined = cgroupPaths
			return nil
		}

		return newCustomizedV2Manager(mockCgroupsManager, false, nil, execVirtChrootFunc)
	}

	newResourcesWithRule := func(rule *devices.Rule) *cgroups.Resources {
		return &cgroups.Resources{
			Devices: []*devices.Rule{
				rule,
			},
		}
	}

	newDeviceRule := func(UID int64) *devices.Rule {
		return &devices.Rule{
			Type:        'z',
			Major:       UID,
			Minor:       UID,
			Permissions: "fakePermissions",
			Allow:       true,
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		rulesDefined = make([]*devices.Rule, 0)
		v2DirPath = "/sys/fs/cgroup/"
	})

	AfterEach(func() {
		v2DirPath = ""
	})

	It("should add default rules", func() {
		manager, err := newMockManager()
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule := newDeviceRule(123)

		err = manager.Set(newResourcesWithRule(fakeRule))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(rulesDefined).To(ContainElement(fakeRule), "defined rule is expected to exist")

		defaultDeviceRules := GenerateDefaultDeviceRules()
		for _, defaultRule := range defaultDeviceRules {
			Expect(rulesDefined).To(ContainElement(defaultRule), "default rules are expected to be defined")
		}
		Expect(rulesDefined).To(HaveLen(len(defaultDeviceRules) + 1))
	})

	It("should not override past rules", func() {
		manager, err := newMockManager()
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule1 := newDeviceRule(123)
		fakeRule2 := newDeviceRule(456)

		err = manager.Set(newResourcesWithRule(fakeRule1))
		Expect(err).ShouldNot(HaveOccurred())

		err = manager.Set(newResourcesWithRule(fakeRule2))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(rulesDefined).To(ContainElement(fakeRule1), "previous rule is expected to not be overridden")
	})

	It("should override past rules if explicitly set", func() {
		manager, err := newMockManager()
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule := newDeviceRule(123)
		fakeRule.Permissions = "fake-permissions-123"

		err = manager.Set(newResourcesWithRule(fakeRule))
		Expect(err).ShouldNot(HaveOccurred())
		Expect(rulesDefined).To(ContainElement(fakeRule), "defined rule is expected to exist")

		fakeRule.Permissions = "fake-permissions-456"
		Expect(rulesDefined).To(ContainElement(fakeRule), "rule needs to be overridden since explicitly re-set")
	})

	DescribeTable("ensure that correct set of cgroups is configured", func(dirPath string, expectedPaths []string) {
		v2DirPath = dirPath
		manager, err := newMockManager()
		Expect(err).ShouldNot(HaveOccurred())

		fakeRule := newDeviceRule(123)

		err = manager.Set(newResourcesWithRule(fakeRule))
		Expect(err).ShouldNot(HaveOccurred())

		Expect(rulesDefined).To(ContainElement(fakeRule), "defined rule is expected to exist")

		defaultDeviceRules := GenerateDefaultDeviceRules()
		for _, defaultRule := range defaultDeviceRules {
			Expect(rulesDefined).To(ContainElement(defaultRule), "default rules are expected to be defined")
		}
		Expect(rulesDefined).To(HaveLen(len(defaultDeviceRules) + 1))
		Expect(cgroupPathsDefined).To(ConsistOf(expectedPaths))
	},
		Entry("for crun installation",
			"/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/crio-456.scope/container",
			[]string{
				"/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/crio-456.scope/container",
				"/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/crio-456.scope",
			},
		),
		Entry("for runc installation",
			"/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/crio-456.scope",
			[]string{
				"/sys/fs/cgroup/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod123.slice/crio-456.scope",
			},
		),
	)
})

var _ = Describe("GetMiscCapacity", func() {
	var originalMiscCapacityPath string
	var tempDir string

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		originalMiscCapacityPath = miscCapacityPath
		miscCapacityPath = path.Join(tempDir, "misc.capacity")
	})

	AfterEach(func() {
		miscCapacityPath = originalMiscCapacityPath
	})

	DescribeTable("should return correct capacity",
		func(fileContent string, key string, expectedCapacity int, expectError bool) {
			if fileContent != "" {
				err := os.WriteFile(path.Join(tempDir, "misc.capacity"), []byte(fileContent), 0644)
				Expect(err).ToNot(HaveOccurred())
			}
			capacity, err := GetMiscCapacity(key)
			Expect(capacity).To(Equal(expectedCapacity))
			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("returns capacity for matching key",
			"tdx 10\nsev 5\n", "tdx", 10, false,
		),
		Entry("produces error when key not found",
			"tdx 10\nsev 5\n", "nonexistent", 0, true,
		),
		Entry("produces error for malformed line",
			"tdx\n", "tdx", 0, true,
		),
		Entry("produces error for non-numeric capacity",
			"tdx abc\n", "tdx", 0, true,
		),
	)
})

var _ = Describe("generateDeviceRulesForVMI", func() {
	var (
		origStatDevice func(*safepath.Path, string) (os.FileInfo, error)
		origReadDevDir func(*safepath.Path, string) ([]os.DirEntry, error)
	)

	noDevices := func(_ *safepath.Path, _ string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	noDirs := func(_ *safepath.Path, _ string) ([]os.DirEntry, error) {
		return nil, os.ErrNotExist
	}

	BeforeEach(func() {
		origStatDevice = statDevice
		origReadDevDir = readDeviceDir
	})

	AfterEach(func() {
		statDevice = origStatDevice
		readDeviceDir = origReadDevDir
	})

	It("should skip hypervisor device rule when emulation is allowed and device is missing", func() {
		statDevice = noDevices
		readDeviceDir = noDirs

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})

	It("should fail when hypervisor device is missing and emulation is not allowed", func() {
		statDevice = noDevices
		readDeviceDir = noDirs

		_, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", false)
		Expect(err).To(HaveOccurred())
	})

	It("should create a rule for the hypervisor device", func() {
		statDevice = func(_ *safepath.Path, relPath string) (os.FileInfo, error) {
			if relPath == "/dev/kvm" {
				return charDeviceInfo(10, 232), nil
			}
			return nil, os.ErrNotExist
		}
		readDeviceDir = noDirs

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(10)), "Minor": Equal(int64(232)),
			})),
		))
	})

	It("should discover VFIO device nodes", func() {
		statDevice = func(_ *safepath.Path, relPath string) (os.FileInfo, error) {
			switch relPath {
			case "/dev/vfio/vfio":
				return charDeviceInfo(10, 196), nil
			case "/dev/vfio/42":
				return charDeviceInfo(243, 0), nil
			default:
				return nil, os.ErrNotExist
			}
		}
		readDeviceDir = func(_ *safepath.Path, relPath string) ([]os.DirEntry, error) {
			if relPath == "/dev/vfio" {
				return []os.DirEntry{
					&fakeDirEntry{name: "vfio"},
					&fakeDirEntry{name: "42"},
				}, nil
			}
			return nil, os.ErrNotExist
		}

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(10)), "Minor": Equal(int64(196)),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(243)), "Minor": Equal(int64(0)),
			})),
		))
	})

	It("should discover USB device nodes in nested directories", func() {
		statDevice = func(_ *safepath.Path, relPath string) (os.FileInfo, error) {
			switch relPath {
			case "/dev/bus/usb/001/001":
				return charDeviceInfo(189, 0), nil
			case "/dev/bus/usb/001/002":
				return charDeviceInfo(189, 1), nil
			default:
				return nil, os.ErrNotExist
			}
		}
		readDeviceDir = func(_ *safepath.Path, relPath string) ([]os.DirEntry, error) {
			switch relPath {
			case "/dev/bus/usb":
				return []os.DirEntry{&fakeDirEntry{name: "001", isDir: true}}, nil
			case "/dev/bus/usb/001":
				return []os.DirEntry{
					&fakeDirEntry{name: "001"},
					&fakeDirEntry{name: "002"},
				}, nil
			default:
				return nil, os.ErrNotExist
			}
		}

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(189)), "Minor": Equal(int64(0)),
			})),
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(189)), "Minor": Equal(int64(1)),
			})),
		))
	})

	It("should discover devices from both VFIO and USB", func() {
		statDevice = func(_ *safepath.Path, relPath string) (os.FileInfo, error) {
			switch relPath {
			case "/dev/vfio/vfio":
				return charDeviceInfo(10, 196), nil
			case "/dev/vfio/0":
				return charDeviceInfo(243, 0), nil
			case "/dev/bus/usb/001/001":
				return charDeviceInfo(189, 0), nil
			case "/dev/bus/usb/002/001":
				return charDeviceInfo(189, 128), nil
			default:
				return nil, os.ErrNotExist
			}
		}
		readDeviceDir = func(_ *safepath.Path, relPath string) ([]os.DirEntry, error) {
			switch relPath {
			case "/dev/vfio":
				return []os.DirEntry{
					&fakeDirEntry{name: "vfio"},
					&fakeDirEntry{name: "0"},
				}, nil
			case "/dev/bus/usb":
				return []os.DirEntry{
					&fakeDirEntry{name: "001", isDir: true},
					&fakeDirEntry{name: "002", isDir: true},
				}, nil
			case "/dev/bus/usb/001":
				return []os.DirEntry{&fakeDirEntry{name: "001"}}, nil
			case "/dev/bus/usb/002":
				return []os.DirEntry{&fakeDirEntry{name: "001"}}, nil
			default:
				return nil, os.ErrNotExist
			}
		}

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(HaveLen(4))
	})

	It("should create a rule for urandom when RNG is enabled", func() {
		statDevice = func(_ *safepath.Path, relPath string) (os.FileInfo, error) {
			if relPath == "/dev/urandom" {
				return charDeviceInfo(1, 9), nil
			}
			return nil, os.ErrNotExist
		}
		readDeviceDir = noDirs

		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

		rules, err := generateDeviceRulesForVMI(vmi, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(1)), "Minor": Equal(int64(9)),
			})),
		))
	})

	It("should create a rule for vhost-vsock when AutoattachVSOCK is enabled", func() {
		statDevice = func(_ *safepath.Path, relPath string) (os.FileInfo, error) {
			if relPath == "/dev/vhost-vsock" {
				return charDeviceInfo(10, 241), nil
			}
			return nil, os.ErrNotExist
		}
		readDeviceDir = noDirs

		autoAttach := true
		vmi := &v1.VirtualMachineInstance{}
		vmi.Spec.Domain.Devices.AutoattachVSOCK = &autoAttach

		rules, err := generateDeviceRulesForVMI(vmi, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(ConsistOf(
			PointTo(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(devices.CharDevice), "Major": Equal(int64(10)), "Minor": Equal(int64(241)),
			})),
		))
	})

	It("should not fail when /dev/vfio does not exist", func() {
		statDevice = noDevices
		readDeviceDir = noDirs

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})

	It("should not fail when /dev/vfio exists but is empty", func() {
		statDevice = noDevices
		readDeviceDir = func(_ *safepath.Path, relPath string) ([]os.DirEntry, error) {
			if relPath == "/dev/vfio" {
				return nil, nil
			}
			return nil, os.ErrNotExist
		}

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})

	It("should not fail when /dev/bus/usb exists but is empty", func() {
		statDevice = noDevices
		readDeviceDir = func(_ *safepath.Path, relPath string) ([]os.DirEntry, error) {
			if relPath == "/dev/bus/usb" {
				return nil, nil
			}
			return nil, os.ErrNotExist
		}

		rules, err := generateDeviceRulesForVMI(&v1.VirtualMachineInstance{}, nil, "", "kvm", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(BeEmpty())
	})
})

func charDeviceInfo(major, minor uint32) os.FileInfo {
	return &fakeFileInfo{
		mode: os.ModeDevice | os.ModeCharDevice,
		rdev: unix.Mkdev(major, minor),
	}
}

type fakeFileInfo struct {
	name string
	mode os.FileMode
	rdev uint64
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return 0 }
func (f *fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f *fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f *fakeFileInfo) IsDir() bool        { return f.mode.IsDir() }
func (f *fakeFileInfo) Sys() interface{}   { return &syscall.Stat_t{Rdev: f.rdev} }

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (e *fakeDirEntry) Name() string { return e.name }
func (e *fakeDirEntry) IsDir() bool  { return e.isDir }
func (e *fakeDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}
func (e *fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }
