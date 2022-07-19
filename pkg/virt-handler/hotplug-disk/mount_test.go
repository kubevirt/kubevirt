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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package hotplug_volume

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/prometheus/procfs"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var (
	tempDir              string
	tmpDirSafe           *safepath.Path
	orgIsoDetector       = isolationDetector
	orgProcMounts        = procMounts
	orgDeviceBasePath    = deviceBasePath
	orgStatSourceCommand = statSourceDevice
	orgStatCommand       = statDevice
	orgCgroupsBasePath   = cgroupsBasePath
	orgMknodCommand      = mknodCommand
	orgSourcePodBasePath = sourcePodBasePath
	orgMountCommand      = mountCommand
	orgUnMountCommand    = unmountCommand
	orgIsMounted         = isMounted
	orgIsBlockDevice     = isBlockDevice
)

var _ = Describe("HotplugVolume mount target records", func() {
	var (
		m      *volumeMounter
		err    error
		vmi    *v1.VirtualMachineInstance
		record *vmiMountTargetRecord
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"

		m = &volumeMounter{
			podIsolationDetector: &mockIsolationDetector{},
			mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:        tempDir,
		}
		record = &vmiMountTargetRecord{
			MountTargetEntries: []vmiMountTargetEntry{
				{
					TargetFile: filepath.Join(tempDir, "test"),
				},
			},
		}
		err := m.setMountTargetRecord(vmi, record)
		Expect(err).ToNot(HaveOccurred())
		expectedBytes, err := json.Marshal(record)
		Expect(err).ToNot(HaveOccurred())
		bytes, err := ioutil.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
		Expect(err).ToNot(HaveOccurred())
		Expect(bytes).To(Equal(expectedBytes))
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		statSourceDevice = orgStatSourceCommand
		cgroupsBasePath = orgCgroupsBasePath
		mknodCommand = orgMknodCommand
	})

	It("setMountTargetRecord should fail if vmi.UID is empty", func() {
		vmi.UID = ""
		record := &vmiMountTargetRecord{
			MountTargetEntries: []vmiMountTargetEntry{
				{
					TargetFile: filepath.Join(tempDir, "test"),
				},
			},
		}
		err := m.setMountTargetRecord(vmi, record)
		Expect(err).To(HaveOccurred())
	})

	It("getMountTargetRecord should get record from file if not in cache", func() {
		res, err := m.getMountTargetRecord(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(reflect.DeepEqual(*res, *record)).To(BeTrue())
	})

	It("getMountTargetRecord should get record from cache if in cache", func() {
		cacheRecord := &vmiMountTargetRecord{
			MountTargetEntries: []vmiMountTargetEntry{
				{
					TargetFile: "test2",
				},
			},
		}
		m.mountRecords[vmi.UID] = cacheRecord
		res, err := m.getMountTargetRecord(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(reflect.DeepEqual(*res, *cacheRecord)).To(BeTrue())
	})

	It("getMountTargetRecord should error if vmi UID is empty", func() {
		vmi.UID = ""
		_, err := m.getMountTargetRecord(vmi)
		Expect(err).To(HaveOccurred())
	})

	It("getMountTargetRecord should return nil not in cache and nothing stored in file", func() {
		err := m.deleteMountTargetRecord(vmi)
		Expect(err).ToNot(HaveOccurred())
		res, err := m.getMountTargetRecord(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(&vmiMountTargetRecord{UsesSafePaths: true}))
	})

	It("deleteMountTargetRecord should remove both record file and entry file", func() {
		err := ioutil.WriteFile(filepath.Join(tempDir, "test"), []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		err = m.deleteMountTargetRecord(vmi)
		Expect(err).ToNot(HaveOccurred())
		recordFile := filepath.Join(tempDir, string(vmi.UID))
		_, err = os.Stat(recordFile)
		Expect(err).To(HaveOccurred())
		_, err = os.Stat(filepath.Join(tempDir, "test"))
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("HotplugVolume block devices", func() {
	var (
		m      *volumeMounter
		err    error
		vmi    *v1.VirtualMachineInstance
		record *vmiMountTargetRecord
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		tmpDirSafe, err = safepath.JoinAndResolveWithRelativeRoot(tempDir)
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"
		activePods := make(map[types.UID]string, 0)
		activePods["abcd"] = "host"
		vmi.Status.ActivePods = activePods

		m = &volumeMounter{
			podIsolationDetector: &mockIsolationDetector{},
			mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:        tempDir,
			skipSafetyCheck:      true,
		}
		record = &vmiMountTargetRecord{}

		deviceBasePath = func(sourceUID types.UID) (*safepath.Path, error) {
			return safepath.JoinAndResolveWithRelativeRoot(tempDir, string(sourceUID))
		}
		statSourceDevice = func(*safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0777, 123456), nil
		}
		cgroupsBasePath = func() (*safepath.Path, error) {
			return tmpDirSafe, nil
		}

	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		statSourceDevice = orgStatSourceCommand
		statDevice = orgStatCommand
		cgroupsBasePath = orgCgroupsBasePath
		mknodCommand = orgMknodCommand
		isBlockDevice = orgIsBlockDevice
	})

	It("isBlockVolume should determine if we have a block volume", func() {
		err = os.RemoveAll(filepath.Join(tempDir, string(vmi.UID), "volumeDevices"))
		Expect(err).ToNot(HaveOccurred())
		By("Passing empty UID, should return false")
		res := m.isBlockVolume("")
		Expect(res).To(BeFalse())
		By("Not having the volume directory, should return false")
		res = m.isBlockVolume(vmi.UID)
		Expect(res).To(BeFalse(), filepath.Join(tempDir, string(vmi.UID), "volumeDevices"))
		By("Creating the volume directory, should return true")
		err = os.MkdirAll(filepath.Join(tempDir, string(vmi.UID), "volumeDevices"), 0755)
		Expect(err).ToNot(HaveOccurred())
		isBlockDevice = func(path *safepath.Path) (bool, error) {
			return strings.Contains(unsafepath.UnsafeAbsolute(path.Raw()), string(vmi.UID)), nil
		}
		res = m.isBlockVolume(vmi.UID)
		Expect(res).To(BeTrue())
	})

	It("findVirtlauncherUID should find the right UID", func() {
		res := m.findVirtlauncherUID(vmi)
		Expect(res).To(BeEquivalentTo("abcd"))
		vmi.Status.ActivePods["abcde"] = "host"
		res = m.findVirtlauncherUID(vmi)
		Expect(res).To(BeEquivalentTo(""))
	})

	It("mountBlockHotplugVolume and unmountBlockHotplugVolumes should make appropriate calls", func() {
		blockSourcePodUID := types.UID("fghij")
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		sourcePodPath := filepath.Join(tempDir, string(blockSourcePodUID), "volumes/kubernetes.io~empty-dir/hotplug-disks/testvolume")
		err = os.MkdirAll(sourcePodPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		deviceFile, err := newFile(tempDir, string(blockSourcePodUID), "volumes", "testvolume", "file")
		Expect(err).ToNot(HaveOccurred())
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(filepath.Join(tempDir, slicePath), 0755)
		Expect(err).ToNot(HaveOccurred())
		devicesFile := filepath.Join(tempDir, slicePath, "devices.list")
		allowFile := filepath.Join(tempDir, slicePath, "devices.allow")
		denyFile := filepath.Join(tempDir, slicePath, "devices.deny")
		_, err = os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(devicesFile)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(unsafepath.UnsafeAbsolute(deviceFile.Raw()), []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0666, 123456), nil
		}
		statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0666, 123456), nil
		}
		isBlockDevice = func(fileName *safepath.Path) (bool, error) {
			return true, nil
		}
		err = m.mountBlockHotplugVolume(vmi, "testvolume", blockSourcePodUID, record)
		Expect(err).ToNot(HaveOccurred())
		By("Verifying the block file exists")
		_, err = os.Stat(filepath.Join(targetPodPath, "testvolume"))
		Expect(err).ToNot(HaveOccurred())
		By("Verifying the correct values are written to the allow file")
		content, err := ioutil.ReadFile(allowFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 482:64 rwm").To(Equal(string(content)))

		By("Unmounting, we verify the reverse process happens")
		path, err := safepath.JoinAndResolveWithRelativeRoot(targetPodPath, "testvolume")
		Expect(err).ToNot(HaveOccurred())
		err = m.unmountBlockHotplugVolumes(path, vmi)
		Expect(err).ToNot(HaveOccurred())
		content, err = ioutil.ReadFile(denyFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 482:64 rwm").To(Equal(string(content)))
		_, err = os.Stat(filepath.Join(targetPodPath, "testvolume"))
		Expect(err).To(HaveOccurred())
	})

	It("getSourceMajorMinor should return an error if no uid", func() {
		vmi.UID = ""
		_, _, err := m.getSourceMajorMinor("fghij", "")
		Expect(err).To(HaveOccurred())
	})

	It("getSourceMajorMinor should return file if exists", func() {
		isBlockDevice = func(fileName *safepath.Path) (bool, error) {
			return true, nil
		}
		deviceFile := filepath.Join(tempDir, "fghij", "volumes", "file")
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(deviceFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		numbers, perm, err := m.getSourceMajorMinor("fghij", "file")
		Expect(err).ToNot(HaveOccurred())
		Expect(unix.Major(numbers)).To(Equal(uint32(482)))
		Expect(unix.Minor(numbers)).To(Equal(uint32(64)))
		Expect(perm & 0777).To(Equal(os.FileMode(0777)))
	})

	It("getSourceMajorMinor should return error if file doesn't exists", func() {
		deviceFile := filepath.Join(tempDir, "fghij", "volumes", "file")
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		_, _, err := m.getSourceMajorMinor("fghij", "file")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no such file or directory"))
	})

	table.DescribeTable("Should return proper values", func(stat func(safePath *safepath.Path) (os.FileInfo, error), major, minor uint32, perm os.FileMode, expectErr bool) {
		statSourceDevice = stat
		testFileName, err := newFile(tempDir, "test-file")
		Expect(err).ToNot(HaveOccurred())
		numbers, permRes, err := m.getBlockFileMajorMinor(testFileName, statSourceDevice)
		if expectErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(unix.Major(numbers)).To(Equal(major))
		Expect(unix.Minor(numbers)).To(Equal(minor))
		Expect(perm).To(Equal(permRes & 0777))
	},
		table.Entry("Should return values if stat command successful", func(safePath *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), nil
		}, uint32(482), uint32(64), os.FileMode(0664), false),
		table.Entry("Should not return error if stat command errors", func(safePath *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), fmt.Errorf("Error")
		}, uint32(0), uint32(0), os.FileMode(0), true),
	)

	It("getTargetCgroupPath should return cgroup path", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(expectedCgroupPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		path, err := m.getTargetCgroupPath(vmi)
		Expect(err).ToNot(HaveOccurred())
		expectedPath, err := safepath.JoinNoFollow(tmpDirSafe, slicePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(path).To(Equal(expectedPath))
	})

	It("getTargetCgroupPath should return error if detect returns error", func() {
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
			err:   fmt.Errorf("dectection error"),
		}
		_, err := m.getTargetCgroupPath(vmi)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("dectection error"))
	})

	It("getTargetCgroupPath should return error if path doesn't exist", func() {
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		_, err := m.getTargetCgroupPath(vmi)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no such file or directory"))
	})

	It("getTargetCgroupPath should return cgroup path", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		_, err = os.Create(expectedCgroupPath)
		Expect(err).ToNot(HaveOccurred())
		_, err := m.getTargetCgroupPath(vmi)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("but it is not a directory"))
	})

	It("should write properly to allow/deny files if able", func() {
		allowFile := filepath.Join(tempDir, "devices.allow")
		listFile := filepath.Join(tempDir, "devices.list")
		denyFile := filepath.Join(tempDir, "devices.deny")
		_, err := os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		err = m.allowBlockMajorMinor(unix.Mkdev(482, 64), tmpDirSafe)
		Expect(err).ToNot(HaveOccurred())
		content, err := ioutil.ReadFile(allowFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 482:64 rwm").To(Equal(string(content)))

		err = m.removeBlockMajorMinor(unix.Mkdev(482, 64), tmpDirSafe)
		Expect(err).ToNot(HaveOccurred())
		content, err = ioutil.ReadFile(denyFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 482:64 rwm").To(Equal(string(content)))
	})

	It("Should error if allow/deny cannot be found", func() {
		err = m.allowBlockMajorMinor(unix.Mkdev(482, 64), tmpDirSafe)
		Expect(err).To(HaveOccurred())

		err = m.removeBlockMajorMinor(unix.Mkdev(482, 64), tmpDirSafe)
		Expect(err).To(HaveOccurred())
	})

	It("Should attempt to create a block device file if it doesn't exist", func() {
		testMajor := uint32(482)
		testMinor := uint32(64)
		testPerm := os.FileMode(0664)
		mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
			Expect(basePath).To(Equal(tmpDirSafe))
			Expect(deviceName).To(Equal("testfile"))
			Expect(unix.Major(dev)).To(Equal(testMajor))
			Expect(unix.Minor(dev)).To(Equal(testMinor))
			Expect(blockDevicePermissions).To(Equal(testPerm))
			return nil
		}
		err := m.createBlockDeviceFile(tmpDirSafe, "testfile", 123456, testPerm)
		Expect(err).ToNot(HaveOccurred())

		mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
			Expect(basePath).To(Equal(tmpDirSafe))
			Expect(deviceName).To(Equal("testfile"))
			Expect(unix.Major(dev)).To(Equal(testMajor))
			Expect(unix.Minor(dev)).To(Equal(testMinor))
			Expect(blockDevicePermissions).To(Equal(testPerm))
			return fmt.Errorf("Error creating block file")
		}
		err = m.createBlockDeviceFile(tmpDirSafe, "testfile", 123456, testPerm)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Error creating block file"))
	})

	It("Should not attempt to create a block device file if it exists", func() {
		testFile := filepath.Join(tempDir, "testfile")
		testPerm := os.FileMode(0664)
		_, err = os.Create(testFile)
		Expect(err).ToNot(HaveOccurred())
		mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
			Fail("Should not get called")
			return nil
		}
		err := m.createBlockDeviceFile(tmpDirSafe, "testfile", 123456, testPerm)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should remove the block device and permissions on unmount", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(expectedCgroupPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), nil
		}
		statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), nil
		}
		isBlockDevice = func(fileName *safepath.Path) (bool, error) {
			return true, nil
		}
		deviceFile, err := newFile(tempDir, "devicefile")
		Expect(err).ToNot(HaveOccurred())
		denyFile := filepath.Join(expectedCgroupPath, "devices.deny")
		listFile := filepath.Join(expectedCgroupPath, "devices.list")
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		err = m.unmountBlockHotplugVolumes(deviceFile, vmi)
		Expect(err).ToNot(HaveOccurred())
		content, err := ioutil.ReadFile(denyFile)
		Expect(err).ToNot(HaveOccurred())
		// Since stat returns values in hex, we need to get hex value as int.
		Expect("b 482:64 rwm").To(Equal(string(content)))
	})

	It("Should return error if deviceFile doesn' exist", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(expectedCgroupPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), nil
		}
		statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), nil
		}
		deviceFile, err := newFile(tempDir, "devicefile")
		Expect(err).ToNot(HaveOccurred())
		err = m.unmountBlockHotplugVolumes(deviceFile, vmi)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no such file or directory"))
	})

	It("Should return error if detect fails", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
			err:   fmt.Errorf("Error detecting"),
		}
		err = os.MkdirAll(expectedCgroupPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0644, 123456), nil
		}
		statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0664, 123456), nil
		}
		deviceFile, err := newFile(tempDir, "devicefile")
		Expect(err).ToNot(HaveOccurred())
		err = m.unmountBlockHotplugVolumes(deviceFile, vmi)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Error detecting"))
	})
})

var _ = Describe("HotplugVolume filesystem volumes", func() {
	var (
		m      *volumeMounter
		err    error
		vmi    *v1.VirtualMachineInstance
		record *vmiMountTargetRecord
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"
		activePods := make(map[types.UID]string, 0)
		activePods["abcd"] = "host"
		vmi.Status.ActivePods = activePods

		record = &vmiMountTargetRecord{}

		m = &volumeMounter{
			podIsolationDetector: &mockIsolationDetector{},
			mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:        tempDir,
		}

		deviceBasePath = func(sourceUID types.UID) (*safepath.Path, error) {
			return newDir(tempDir, string(sourceUID))
		}

	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		sourcePodBasePath = orgSourcePodBasePath
		mountCommand = orgMountCommand
		unmountCommand = orgUnMountCommand
		isMounted = orgIsMounted
		procMounts = orgProcMounts
		isolationDetector = orgIsoDetector
	})

	It("getSourcePodFile should find the disk.img file, if it exists", func() {
		path, err := newDir(tempDir, "ghfjk", "volumes")
		Expect(err).ToNot(HaveOccurred())
		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return path, nil
		}
		_, err = newFile(unsafepath.UnsafeAbsolute(path.Raw()), "disk.img")
		Expect(err).ToNot(HaveOccurred())
		file, err := m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeRelative(file.Raw())).To(Equal(unsafepath.UnsafeAbsolute(path.Raw())))
	})

	It("getSourcePodFile should return error if no UID", func() {
		_, err := m.getSourcePodFilePath("", vmi, "")
		Expect(err).To(HaveOccurred())
	})

	It("getSourcePodFile should return error if disk.img doesn't exist", func() {
		path, err := newDir(tempDir, "ghfjk", "volumes")
		Expect(err).ToNot(HaveOccurred())
		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return path, nil
		}
		_, err = m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).To(HaveOccurred())
	})

	It("getSourcePodFile should return error if iso detection returns error", func() {
		expectedPath, err := newDir(tempDir, "ghfjk", "volumes")
		Expect(err).ToNot(HaveOccurred())
		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return expectedPath, nil
		}
		isolationDetector = func(path string) isolation.PodIsolationDetector {
			return &mockIsolationDetector{
				pid: 9999,
			}
		}

		_, err = m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("isolation error"))
	})

	It("getSourcePodFile should return error if proc mounts returns error", func() {
		expectedPath, err := newDir(tempDir, "ghfjk", "volumes")
		Expect(err).ToNot(HaveOccurred())
		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return expectedPath, nil
		}
		isolationDetector = func(path string) isolation.PodIsolationDetector {
			return &mockIsolationDetector{
				pid: 1,
			}
		}
		procMounts = func(pid int) ([]*procfs.MountInfo, error) {
			Expect(pid).To(Equal(1))
			res := make([]*procfs.MountInfo, 0)
			return res, fmt.Errorf("mount detection error")
		}

		_, err = m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mount detection error"))
	})

	It("getSourcePodFile should return the mountinfo value", func() {
		expectedPath, err := newDir(tempDir, "ghfjk", "volumes")
		Expect(err).ToNot(HaveOccurred())
		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return expectedPath, nil
		}
		isolationDetector = func(path string) isolation.PodIsolationDetector {
			return &mockIsolationDetector{
				pid: 1,
			}
		}
		os.MkdirAll("/test/result", 0755)
		procMounts = func(pid int) ([]*procfs.MountInfo, error) {
			Expect(pid).To(Equal(1))
			res := make([]*procfs.MountInfo, 0)
			res = append(res, &procfs.MountInfo{
				Root:       "/test/result",
				MountPoint: "/pvc",
			})
			return res, nil
		}

		res, err := m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeRelative(res.Raw())).To(Equal("/test/result"))
	})

	It("should properly mount and unmount filesystem", func() {
		sourcePodUID := "ghfjk"
		path, err := newDir(tempDir, sourcePodUID, "volumes")
		Expect(err).ToNot(HaveOccurred())
		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return path, nil
		}
		safepath.TouchAtNoFollow(path, "disk.img", os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		targetPodPath, err := newDir(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		Expect(err).ToNot(HaveOccurred())
		targetFilePath, err := newFile(unsafepath.UnsafeAbsolute(targetPodPath.Raw()), "testvolume")
		Expect(err).ToNot(HaveOccurred())
		mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
			Expect(unsafepath.UnsafeRelative(sourcePath.Raw())).To(Equal(unsafepath.UnsafeAbsolute(path.Raw())))
			Expect(targetPath).To(Equal(targetFilePath))
			return []byte("Success"), nil
		}
		isMounted = func(diskPath *safepath.Path) (bool, error) {
			Expect(targetFilePath).To(Equal(diskPath))
			return false, nil
		}

		err = m.mountFileSystemHotplugVolume(vmi, "testvolume", types.UID(sourcePodUID), record)
		Expect(err).ToNot(HaveOccurred())
		Expect(record.MountTargetEntries[0].TargetFile).To(Equal(unsafepath.UnsafeAbsolute(targetFilePath.Raw())))

		unmountCommand = func(diskPath *safepath.Path) ([]byte, error) {
			Expect(targetFilePath).To(Equal(diskPath))
			return []byte("Success"), nil
		}

		isMounted = func(diskPath *safepath.Path) (bool, error) {
			Expect(targetFilePath).To(Equal(diskPath))
			return true, nil
		}

		err = m.unmountFileSystemHotplugVolumes(targetFilePath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("unmountFileSystemHotplugVolumes should return error if isMounted returns error", func() {
		testPath, err := newFile(tempDir, "test")
		Expect(err).ToNot(HaveOccurred())
		isMounted = func(path *safepath.Path) (bool, error) {
			Expect(testPath).To(Equal(path))
			return false, fmt.Errorf("isMounted error")
		}

		err = m.unmountFileSystemHotplugVolumes(testPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("isMounted error"))
	})

	It("unmountFileSystemHotplugVolumes should return nil if isMounted returns false", func() {
		testPath, err := newFile(tempDir, "test")
		Expect(err).ToNot(HaveOccurred())
		isMounted = func(diskPath *safepath.Path) (bool, error) {
			Expect(testPath).To(Equal(diskPath))
			return false, nil
		}

		err = m.unmountFileSystemHotplugVolumes(testPath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("unmountFileSystemHotplugVolumes should return error if unmountCommand returns error", func() {
		testPath, err := newFile(tempDir, "test")
		Expect(err).ToNot(HaveOccurred())
		isMounted = func(diskPath *safepath.Path) (bool, error) {
			Expect(testPath).To(Equal(diskPath))
			return true, nil
		}
		unmountCommand = func(diskPath *safepath.Path) ([]byte, error) {
			return []byte("Failure"), fmt.Errorf("unmount error")
		}

		err = m.unmountFileSystemHotplugVolumes(testPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unmount error"))
	})
})

var _ = Describe("HotplugVolume volumes", func() {
	var (
		m   *volumeMounter
		err error
		vmi *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		tmpDirSafe, err = safepath.JoinAndResolveWithRelativeRoot(tempDir)
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"
		activePods := make(map[types.UID]string, 0)
		activePods["abcd"] = "host"
		vmi.Status.ActivePods = activePods

		m = &volumeMounter{
			podIsolationDetector: &mockIsolationDetector{},
			mountRecords:         make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:        tempDir,
			skipSafetyCheck:      true,
		}

		deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return newDir(tempDir, string(podUID))
		}
		statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0777, 123456), nil
		}
		statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
			return fakeStat(true, 0777, 123456), nil
		}
		cgroupsBasePath = func() (*safepath.Path, error) {
			return tmpDirSafe, nil
		}
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		sourcePodBasePath = orgSourcePodBasePath
		mountCommand = orgMountCommand
		unmountCommand = orgUnMountCommand
		isMounted = orgIsMounted
		statSourceDevice = orgStatSourceCommand
		statDevice = orgStatCommand
		cgroupsBasePath = orgCgroupsBasePath
		mknodCommand = orgMknodCommand
		isBlockDevice = orgIsBlockDevice
	})

	It("mount and umount should work for filesystem volumes", func() {
		blockSourcePodUID := types.UID("klmno")
		fsSourcePodUID := types.UID("fghij")
		volumeStatuses := make([]v1.VolumeStatus, 0)
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "permanent",
		})
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "filesystemvolume",
			HotplugVolume: &v1.HotplugVolumeStatus{
				AttachPodName: "fs-pod",
				AttachPodUID:  fsSourcePodUID,
			},
		})
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "blockvolume",
			HotplugVolume: &v1.HotplugVolumeStatus{
				AttachPodName: "block-pod",
				AttachPodUID:  blockSourcePodUID,
			},
		})
		isBlockDevice = func(path *safepath.Path) (bool, error) {
			return strings.Contains(unsafepath.UnsafeAbsolute(path.Raw()), "blockvolume"), nil
		}
		vmi.Status.VolumeStatus = volumeStatuses
		deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return safepath.JoinAndResolveWithRelativeRoot(tempDir, string(podUID))
		}
		blockDevicePath, err := newDir(tempDir, string(blockSourcePodUID))
		Expect(err).ToNot(HaveOccurred())
		fileSystemPath, err := newDir(tempDir, string(fsSourcePodUID), "volumes")
		Expect(err).ToNot(HaveOccurred())
		_, err = newDir(tempDir, string(blockSourcePodUID), "volumeDevices")
		Expect(err).ToNot(HaveOccurred())
		deviceFile, err := newFile(unsafepath.UnsafeAbsolute(blockDevicePath.Raw()), "volumes", "blockvolume")
		Expect(err).ToNot(HaveOccurred())
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(filepath.Join(tempDir, slicePath), 0755)
		Expect(err).ToNot(HaveOccurred())
		allowFile := filepath.Join(tempDir, slicePath, "devices.allow")
		listFile := filepath.Join(tempDir, slicePath, "devices.list")
		denyFile := filepath.Join(tempDir, slicePath, "devices.deny")
		_, err = os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(unsafepath.UnsafeAbsolute(deviceFile.Raw()), []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())

		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			if podUID == blockSourcePodUID {
				return blockDevicePath, nil
			}
			return fileSystemPath, nil
		}
		_, err = newFile(unsafepath.UnsafeAbsolute(fileSystemPath.Raw()), "disk.img")
		Expect(err).ToNot(HaveOccurred())
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		fileSystemVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/filesystemvolume")
		blockVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/blockvolume")
		targetFilePath := filepath.Join(targetPodPath, "filesystemvolume")
		isMounted = func(diskPath *safepath.Path) (bool, error) {
			Expect(targetFilePath).To(Equal(unsafepath.UnsafeAbsolute(diskPath.Raw())))
			return false, nil
		}
		mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
			Expect(unsafepath.UnsafeAbsolute(sourcePath.Raw())).To(Equal(unsafepath.UnsafeAbsolute(fileSystemPath.Raw())))
			Expect(unsafepath.UnsafeAbsolute(targetPath.Raw())).To(Equal(targetFilePath))
			return []byte("Success"), nil
		}
		err = m.Mount(vmi)
		Expect(err).ToNot(HaveOccurred())
		By("Verifying there are 2 records in tempDir/1234")
		record := &vmiMountTargetRecord{
			MountTargetEntries: []vmiMountTargetEntry{
				{
					TargetFile: fileSystemVolume,
				},
				{
					TargetFile: blockVolume,
				},
			},
			UsesSafePaths: true,
		}
		expectedBytes, err := json.Marshal(record)
		Expect(err).ToNot(HaveOccurred())
		bytes, err := ioutil.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
		Expect(err).ToNot(HaveOccurred())
		Expect(bytes).To(Equal(expectedBytes))
		_, err = os.Stat(fileSystemVolume)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(blockVolume)
		Expect(err).ToNot(HaveOccurred())

		volumeStatuses = make([]v1.VolumeStatus, 0)
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "permanent",
		})
		vmi.Status.VolumeStatus = volumeStatuses
		err = m.Unmount(vmi)
		Expect(err).ToNot(HaveOccurred())
		_, err = ioutil.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
		Expect(err).To(HaveOccurred(), "record file still exists %s", filepath.Join(tempDir, string(vmi.UID)))
		_, err = os.Stat(fileSystemVolume)
		Expect(err).To(HaveOccurred(), "filesystem volume file still exists %s", fileSystemVolume)
		_, err = os.Stat(blockVolume)
		Expect(err).To(HaveOccurred(), "block device volume still exists %s", blockVolume)
	})

	It("Should not do anything if vmi has no hotplug volumes", func() {
		volumeStatuses := make([]v1.VolumeStatus, 0)
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "permanent",
		})
		vmi.Status.VolumeStatus = volumeStatuses
		Expect(m.Mount(vmi)).To(BeNil())
	})

	It("unmountAll should cleanup regardless of vmi volumestatuses", func() {
		blockSourcePodUID := types.UID("klmno")
		fsSourcePodUID := types.UID("fghij")
		volumeStatuses := make([]v1.VolumeStatus, 0)
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "permanent",
		})
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "filesystemvolume",
			HotplugVolume: &v1.HotplugVolumeStatus{
				AttachPodName: "fs-pod",
				AttachPodUID:  fsSourcePodUID,
			},
		})
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "blockvolume",
			HotplugVolume: &v1.HotplugVolumeStatus{
				AttachPodName: "block-pod",
				AttachPodUID:  blockSourcePodUID,
			},
		})
		isBlockDevice = func(path *safepath.Path) (bool, error) {
			return strings.Contains(unsafepath.UnsafeAbsolute(path.Raw()), "blockvolume"), nil
		}
		isMounted = func(diskPath *safepath.Path) (bool, error) {
			Expect(unsafepath.UnsafeAbsolute(diskPath.Raw())).To(ContainSubstring("filesystemvolume"))
			return false, nil
		}

		vmi.Status.VolumeStatus = volumeStatuses
		deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
			return safepath.JoinAndResolveWithRelativeRoot(tempDir, string(podUID))
		}
		blockDevicePath, err := newDir(tempDir, string(blockSourcePodUID))
		Expect(err).ToNot(HaveOccurred())
		fileSystemPath, err := newDir(tempDir, string(fsSourcePodUID), "volumes")
		Expect(err).ToNot(HaveOccurred())
		_, err = newDir(tempDir, string(blockSourcePodUID), "volumeDevices")
		Expect(err).ToNot(HaveOccurred())
		deviceFile, err := newFile(unsafepath.UnsafeAbsolute(blockDevicePath.Raw()), "volumes", "blockvolume")
		Expect(err).ToNot(HaveOccurred())
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(filepath.Join(tempDir, slicePath), 0755)
		Expect(err).ToNot(HaveOccurred())
		allowFile := filepath.Join(tempDir, slicePath, "devices.allow")
		listFile := filepath.Join(tempDir, slicePath, "devices.list")
		denyFile := filepath.Join(tempDir, slicePath, "devices.deny")
		_, err = os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(unsafepath.UnsafeAbsolute(deviceFile.Raw()), []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())

		sourcePodBasePath = func(podUID types.UID) (*safepath.Path, error) {
			if podUID == blockSourcePodUID {
				return blockDevicePath, nil
			}
			return fileSystemPath, nil
		}

		_, err = newFile(unsafepath.UnsafeAbsolute(fileSystemPath.Raw()), "disk.img")
		Expect(err).ToNot(HaveOccurred())
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		fileSystemVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/filesystemvolume")
		blockVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/blockvolume")
		Expect(err).ToNot(HaveOccurred())
		targetFilePath := filepath.Join(targetPodPath, "filesystemvolume")
		mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
			Expect(unsafepath.UnsafeAbsolute(sourcePath.Raw())).To(Equal(unsafepath.UnsafeAbsolute(fileSystemPath.Raw())))
			Expect(unsafepath.UnsafeAbsolute(targetPath.Raw())).To(Equal(targetFilePath))
			return []byte("Success"), nil
		}
		err = m.Mount(vmi)
		Expect(err).ToNot(HaveOccurred())
		By("Verifying there are 2 records in tempDir/1234")
		record := &vmiMountTargetRecord{
			MountTargetEntries: []vmiMountTargetEntry{
				{
					TargetFile: fileSystemVolume,
				},
				{
					TargetFile: blockVolume,
				},
			},
			UsesSafePaths: true,
		}
		expectedBytes, err := json.Marshal(record)
		Expect(err).ToNot(HaveOccurred())
		bytes, err := ioutil.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
		Expect(err).ToNot(HaveOccurred())
		Expect(bytes).To(Equal(expectedBytes))
		_, err = os.Stat(fileSystemVolume)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(blockVolume)
		Expect(err).ToNot(HaveOccurred())

		err = m.UnmountAll(vmi)
		Expect(err).ToNot(HaveOccurred())
		_, err = ioutil.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
		Expect(err).To(HaveOccurred(), "record file still exists %s", filepath.Join(tempDir, string(vmi.UID)))
		_, err = os.Stat(fileSystemVolume)
		Expect(err).To(HaveOccurred(), "filesystem volume file still exists %s", fileSystemVolume)
		_, err = os.Stat(blockVolume)
		Expect(err).To(HaveOccurred(), "block device volume still exists %s", blockVolume)
	})
})

type mockIsolationDetector struct {
	pid        int
	slice      string
	controller []string
	err        error
}

func (i *mockIsolationDetector) Detect(_ *v1.VirtualMachineInstance) (isolation.IsolationResult, error) {
	return isolation.NewIsolationResult(i.pid, i.slice, i.controller), i.err
}

func (i *mockIsolationDetector) DetectForSocket(_ *v1.VirtualMachineInstance, _ string) (isolation.IsolationResult, error) {
	if i.pid == 1 {
		return isolation.NewIsolationResult(i.pid, tempDir, []string{}), nil
	}
	return nil, fmt.Errorf("isolation error")
}

func (i *mockIsolationDetector) Whitelist(_ []string) isolation.PodIsolationDetector {
	return i
}

func (i *mockIsolationDetector) AdjustResources(_ *v1.VirtualMachineInstance) error {
	return nil
}

func newFile(baseDir string, elems ...string) (*safepath.Path, error) {
	targetPath := filepath.Join(append([]string{baseDir}, elems...)...)
	err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(targetPath)
	if err != nil {
		return nil, err
	}
	f.Close()
	return safepath.JoinAndResolveWithRelativeRoot(baseDir, elems...)
}

func newDir(baseDir string, elems ...string) (*safepath.Path, error) {
	targetPath := filepath.Join(append([]string{baseDir}, elems...)...)
	err := os.MkdirAll(targetPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return safepath.JoinAndResolveWithRelativeRoot(baseDir, elems...)
}

func fakeStat(isDevice bool, mode os.FileMode, dev uint64) os.FileInfo {
	return fakeFileInfo{isDevice: isDevice, mode: mode, dev: dev}
}

type fakeFileInfo struct {
	isDevice bool
	mode     os.FileMode
	dev      uint64
}

func (f fakeFileInfo) Name() string {
	//TODO implement me
	panic("implement me")
}

func (f fakeFileInfo) Size() int64 {
	//TODO implement me
	panic("implement me")
}

func (f fakeFileInfo) Mode() os.FileMode {
	if f.isDevice {
		return f.mode | os.ModeDevice
	}
	return f.mode
}

func (f fakeFileInfo) ModTime() time.Time {
	//TODO implement me
	panic("implement me")
}

func (f fakeFileInfo) IsDir() bool {
	//TODO implement me
	panic("implement me")
}

func (f fakeFileInfo) Sys() interface{} {
	return &syscall.Stat_t{Rdev: f.dev}
}
