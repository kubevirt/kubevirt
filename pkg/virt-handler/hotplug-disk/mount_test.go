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

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var (
	tempDir              string
	orgDeviceBasePath    = deviceBasePath
	orgStatCommand       = statCommand
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
		statCommand = orgStatCommand
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

	It("setMountTargetRecord should not write record to file, if record already exists, and no changes", func() {
		recordFile := filepath.Join(tempDir, string(vmi.UID))
		info, err := os.Stat(recordFile)
		Expect(err).ToNot(HaveOccurred())
		expectedModTime := info.ModTime()
		m.mountRecords[vmi.UID] = record
		err = m.setMountTargetRecord(vmi, record)
		Expect(err).ToNot(HaveOccurred())
		resInfo, err := os.Stat(recordFile)
		Expect(err).ToNot(HaveOccurred())
		resModTime := resInfo.ModTime()
		Expect(expectedModTime).To(Equal(resModTime))
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
		Expect(res).To(Equal(&vmiMountTargetRecord{}))
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

		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumes")
		}
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("6,6,0777,block special file"), nil
		}
		cgroupsBasePath = func() string {
			return tempDir
		}

	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		statCommand = orgStatCommand
		cgroupsBasePath = orgCgroupsBasePath
		mknodCommand = orgMknodCommand
		isBlockDevice = orgIsBlockDevice
	})

	It("isBlockVolume should determine if we have a block volume", func() {
		err = os.RemoveAll(filepath.Join(tempDir, string(vmi.UID), "volumes"))
		Expect(err).ToNot(HaveOccurred())
		By("Passing empty UID, should return false")
		res := m.isBlockVolume("")
		Expect(res).To(BeFalse())
		By("Not having the volume directory, should return false")
		res = m.isBlockVolume(vmi.UID)
		Expect(res).To(BeFalse())
		By("Creating the volume directory, should return true")
		err = os.MkdirAll(filepath.Join(tempDir, string(vmi.UID), "volumes"), 0755)
		Expect(err).ToNot(HaveOccurred())
		isBlockDevice = func(path string) (bool, error) {
			if strings.Contains(path, string(vmi.UID)) {
				return true, nil
			}
			return false, fmt.Errorf("Not a block device")
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
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		deviceFile := filepath.Join(tempDir, string(blockSourcePodUID), "volumes", "file")
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(filepath.Join(tempDir, slicePath), 0755)
		Expect(err).ToNot(HaveOccurred())
		devicesFile := filepath.Join(tempDir, slicePath, "devices.list")
		allowFile := filepath.Join(tempDir, slicePath, "devices.allow")
		denyFile := filepath.Join(tempDir, slicePath, "devices.deny")
		_, err := os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(devicesFile)
		Expect(err).ToNot(HaveOccurred())
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(deviceFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		err = m.mountBlockHotplugVolume(vmi, "testvolume", blockSourcePodUID, record)
		Expect(err).ToNot(HaveOccurred())
		By("Verifying the block file exists")
		_, err = os.Stat(filepath.Join(targetPodPath, "testvolume"))
		Expect(err).ToNot(HaveOccurred())
		By("Verifying the correct values are written to the allow file")
		content, err := ioutil.ReadFile(allowFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 6:6 rwm").To(Equal(string(content)))

		By("Unmounting, we verify the reverse process happens")
		err = m.unmountBlockHotplugVolumes(filepath.Join(targetPodPath, "testvolume"), vmi)
		Expect(err).ToNot(HaveOccurred())
		content, err = ioutil.ReadFile(denyFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 6:6 rwm").To(Equal(string(content)))
		_, err = os.Stat(filepath.Join(targetPodPath, "testvolume"))
		Expect(err).To(HaveOccurred())
	})

	It("getSourceMajorMinor should return an error if no uid", func() {
		vmi.UID = ""
		_, _, _, err := m.getSourceMajorMinor(vmi, "fghij")
		Expect(err).To(HaveOccurred())
	})

	It("getSourceMajorMinor should return file if exists", func() {
		deviceFile := filepath.Join(tempDir, "fghij", "volumes", "file")
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(deviceFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		major, minor, perm, err := m.getSourceMajorMinor(vmi, "fghij")
		Expect(err).ToNot(HaveOccurred())
		Expect(major).To(Equal(int64(6)))
		Expect(minor).To(Equal(int64(6)))
		Expect(perm).To(Equal("0777"))
	})

	It("getSourceMajorMinor should return error if file doesn't exists", func() {
		deviceFile := filepath.Join(tempDir, "fghij", "volumes", "file")
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		major, minor, perm, err := m.getSourceMajorMinor(vmi, "fghij")
		Expect(err).To(HaveOccurred())
		Expect(major).To(Equal(int64(-1)))
		Expect(minor).To(Equal(int64(-1)))
		Expect(perm).To(Equal(""))
	})

	It("isBlockFile should return proper value based on stat command", func() {
		testFileName := "test-file"
		statCommand = func(fileName string) ([]byte, error) {
			Expect(testFileName).To(Equal(fileName))
			return []byte("6,6,0777,block special file"), nil
		}
		Expect(m.isBlockFile(testFileName)).To(BeTrue())
		statCommand = func(fileName string) ([]byte, error) {
			Expect(testFileName).To(Equal(fileName))
			return []byte("6,6,0777,block special file"), fmt.Errorf("Error")
		}
		Expect(m.isBlockFile(testFileName)).To(BeFalse())
		statCommand = func(fileName string) ([]byte, error) {
			Expect(testFileName).To(Equal(fileName))
			return []byte("6,6,0777"), nil
		}
		Expect(m.isBlockFile(testFileName)).To(BeFalse())
		statCommand = func(fileName string) ([]byte, error) {
			Expect(testFileName).To(Equal(fileName))
			return []byte("6,6,0777,block special"), nil
		}
		Expect(m.isBlockFile(testFileName)).To(BeFalse())
	})

	table.DescribeTable("Should return proper values", func(stat func(fileName string) ([]byte, error), major, minor int, perm string, expectErr bool) {
		testFileName := "test-file"
		statCommand = stat
		majorRes, minorRes, permRes, err := m.getBlockFileMajorMinor(testFileName)
		if expectErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
		// Values are translated to hex (245->580, 32->50)
		Expect(int64(major)).To(Equal(majorRes))
		Expect(int64(minor)).To(Equal(minorRes))
		Expect(perm).To(Equal(permRes))
	},
		table.Entry("Should return values if stat command successful", func(fileName string) ([]byte, error) {
			return []byte("245,32,0664,block special file"), nil
		}, 581, 50, "0664", false),
		table.Entry("Should not return values if stat command errors", func(fileName string) ([]byte, error) {
			return []byte("245,32,0664,block special file"), fmt.Errorf("Error")
		}, -1, -1, "", true),
		table.Entry("Should not return values if stat command doesn't return 4 fields", func(fileName string) ([]byte, error) {
			return []byte("245,32,0664"), nil
		}, -1, -1, "", true),
		table.Entry("Should not return values if stat command doesn't return block special file", func(fileName string) ([]byte, error) {
			return []byte("245,32,0664, block file"), nil
		}, -1, -1, "", true),
		table.Entry("Should not return values if stat command doesn't return block special file", func(fileName string) ([]byte, error) {
			return []byte("245,32,0664, block file"), nil
		}, -1, -1, "", true),
		table.Entry("Should not return values if stat command doesn't int major", func(fileName string) ([]byte, error) {
			return []byte("kk,32,0664,block special file"), nil
		}, -1, -1, "", true),
		table.Entry("Should not return values if stat command doesn't int minor", func(fileName string) ([]byte, error) {
			return []byte("254,gg,0664,block special file"), nil
		}, -1, -1, "", true),
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
		Expect(path).To(Equal(filepath.Join(tempDir, slicePath)))
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
		err = m.allowBlockMajorMinor(34, 53, filepath.Dir(allowFile))
		Expect(err).ToNot(HaveOccurred())
		content, err := ioutil.ReadFile(allowFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 34:53 rwm").To(Equal(string(content)))

		err = m.removeBlockMajorMinor(34, 53, filepath.Dir(denyFile))
		Expect(err).ToNot(HaveOccurred())
		content, err = ioutil.ReadFile(denyFile)
		Expect(err).ToNot(HaveOccurred())
		Expect("b 34:53 rwm").To(Equal(string(content)))
	})

	It("Should error if allow/deny cannot be found", func() {
		allowFile := filepath.Join(tempDir, "devices.allow")
		denyFile := filepath.Join(tempDir, "devices.deny")
		err = m.allowBlockMajorMinor(34, 53, filepath.Dir(allowFile))
		Expect(err).To(HaveOccurred())

		err = m.removeBlockMajorMinor(34, 53, filepath.Dir(denyFile))
		Expect(err).To(HaveOccurred())
	})

	It("Should attempt to create a block device file if it doesn't exist", func() {
		testFile := filepath.Join(tempDir, "testfile")
		testMajor := int64(100)
		testMinor := int64(53)
		testPerm := "0664"
		mknodCommand = func(deviceName string, major, minor int64, blockDevicePermissions string) ([]byte, error) {
			Expect(deviceName).To(Equal(testFile))
			Expect(major).To(Equal(testMajor))
			Expect(minor).To(Equal(testMinor))
			Expect(blockDevicePermissions).To(Equal(testPerm))
			return []byte("Yay"), nil
		}
		res, err := m.createBlockDeviceFile(testFile, testMajor, testMinor, testPerm)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(testFile))

		mknodCommand = func(deviceName string, major, minor int64, blockDevicePermissions string) ([]byte, error) {
			Expect(deviceName).To(Equal(testFile))
			Expect(major).To(Equal(testMajor))
			Expect(minor).To(Equal(testMinor))
			Expect(blockDevicePermissions).To(Equal(testPerm))
			return []byte("Yay"), fmt.Errorf("Error creating block file")
		}
		_, err = m.createBlockDeviceFile(testFile, testMajor, testMinor, testPerm)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Error creating block file"))
	})

	It("Should not attempt to create a block device file if it exists", func() {
		testFile := filepath.Join(tempDir, "testfile")
		testMajor := int64(100)
		testMinor := int64(53)
		testPerm := "0664"
		_, err = os.Create(testFile)
		Expect(err).ToNot(HaveOccurred())
		mknodCommand = func(deviceName string, major, minor int64, blockDevicePermissions string) ([]byte, error) {
			Fail("Should not get called")
			return nil, nil
		}
		res, err := m.createBlockDeviceFile(testFile, testMajor, testMinor, testPerm)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(testFile))
	})

	It("Should remove the block device and permissions on unmount", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(expectedCgroupPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("245,32,0664,block special file"), nil
		}
		deviceFileName := filepath.Join(tempDir, "devicefile")
		denyFile := filepath.Join(expectedCgroupPath, "devices.deny")
		listFile := filepath.Join(expectedCgroupPath, "devices.list")
		_, err := os.Create(deviceFileName)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		err = m.unmountBlockHotplugVolumes(deviceFileName, vmi)
		Expect(err).ToNot(HaveOccurred())
		content, err := ioutil.ReadFile(denyFile)
		Expect(err).ToNot(HaveOccurred())
		// Since stat returns values in hex, we need to get hex value as int.
		Expect("b 581:50 rwm").To(Equal(string(content)))
		_, err = os.Stat(deviceFileName)
		Expect(err).To(HaveOccurred())
	})

	It("Should return error if deviceFile doesn' exist", func() {
		slicePath := "slice"
		expectedCgroupPath := filepath.Join(tempDir, slicePath)
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(expectedCgroupPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("245,32,0664,block special file"), nil
		}
		deviceFileName := filepath.Join(tempDir, "devicefile")
		err = m.unmountBlockHotplugVolumes(deviceFileName, vmi)
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
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("245,32,0664,block special file"), nil
		}
		deviceFileName := filepath.Join(tempDir, "devicefile")
		_, err := os.Create(deviceFileName)
		Expect(err).ToNot(HaveOccurred())
		err = m.unmountBlockHotplugVolumes(deviceFileName, vmi)
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

		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumes")
		}
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		sourcePodBasePath = orgSourcePodBasePath
		mountCommand = orgMountCommand
		unmountCommand = orgUnMountCommand
		isMounted = orgIsMounted
	})

	It("getSourcePodFile should find the disk.img file, if it exists", func() {
		path := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(path, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return path
		}
		diskFile := filepath.Join(path, "disk.img")
		_, err := os.Create(diskFile)
		Expect(err).ToNot(HaveOccurred())
		file, err := m.getSourcePodFilePath("ghfjk")
		Expect(err).ToNot(HaveOccurred())
		Expect(file).To(Equal(path))
	})

	It("getSourcePodFile should return error if no UID", func() {
		_, err := m.getSourcePodFilePath("")
		Expect(err).To(HaveOccurred())
	})

	It("getSourcePodFile should return error if disk.img doesn't exist", func() {
		path := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(path, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return path
		}
		Expect(err).ToNot(HaveOccurred())
		_, err := m.getSourcePodFilePath("ghfjk")
		Expect(err).To(HaveOccurred())
	})

	It("should properly mount and unmount filesystem", func() {
		sourcePodUID := "ghfjk"
		path := filepath.Join(tempDir, sourcePodUID, "volumes")
		err = os.MkdirAll(path, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return path
		}
		diskFile := filepath.Join(path, "disk.img")
		_, err := os.Create(diskFile)
		Expect(err).ToNot(HaveOccurred())
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())
		targetFilePath := filepath.Join(targetPodPath, "testvolume")
		mountCommand = func(sourcePath, targetPath string) ([]byte, error) {
			Expect(sourcePath).To(Equal(path))
			Expect(targetPath).To(Equal(targetFilePath))
			return []byte("Success"), nil
		}

		err = m.mountFileSystemHotplugVolume(vmi, "testvolume", types.UID(sourcePodUID), record)
		Expect(err).ToNot(HaveOccurred())
		Expect(record.MountTargetEntries[0].TargetFile).To(Equal(targetFilePath))

		unmountCommand = func(diskPath string) ([]byte, error) {
			Expect(targetFilePath).To(Equal(diskPath))
			return []byte("Success"), nil
		}

		isMounted = func(diskPath string) (bool, error) {
			Expect(targetFilePath).To(Equal(diskPath))
			return true, nil
		}

		err = m.unmountFileSystemHotplugVolumes(record.MountTargetEntries[0].TargetFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(targetFilePath)
		Expect(err).To(HaveOccurred())
	})

	It("unmountFileSystemHotplugVolumes should return error if isMounted returns error", func() {
		testPath := "test"
		isMounted = func(diskPath string) (bool, error) {
			Expect(testPath).To(Equal(diskPath))
			return false, fmt.Errorf("isMounted error")
		}

		err = m.unmountFileSystemHotplugVolumes(testPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("isMounted error"))
	})

	It("unmountFileSystemHotplugVolumes should return nil if isMounted returns false", func() {
		testPath := "test"
		isMounted = func(diskPath string) (bool, error) {
			Expect(testPath).To(Equal(diskPath))
			return false, nil
		}

		err = m.unmountFileSystemHotplugVolumes(testPath)
		Expect(err).ToNot(HaveOccurred())
	})

	It("unmountFileSystemHotplugVolumes should return error if unmountCommand returns error", func() {
		testPath := "test"
		isMounted = func(diskPath string) (bool, error) {
			Expect(testPath).To(Equal(diskPath))
			return true, nil
		}
		unmountCommand = func(diskPath string) ([]byte, error) {
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

		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumes")
		}
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("6,6,0777,block special file"), nil
		}
		cgroupsBasePath = func() string {
			return tempDir
		}
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		sourcePodBasePath = orgSourcePodBasePath
		mountCommand = orgMountCommand
		unmountCommand = orgUnMountCommand
		isMounted = orgIsMounted
		statCommand = orgStatCommand
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
		isBlockDevice = func(path string) (bool, error) {
			log.DefaultLogger().Infof("Checking isBlockDevice for %s", path)
			if strings.Contains(path, string(blockSourcePodUID)) {
				return true, nil
			}
			log.DefaultLogger().Info("Not a block device")
			return false, fmt.Errorf("Not a block device")
		}
		vmi.Status.VolumeStatus = volumeStatuses
		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumeDevices")
		}
		blockDevicePath := filepath.Join(tempDir, string(blockSourcePodUID), "volumeDevices")
		fileSystemPath := filepath.Join(tempDir, string(fsSourcePodUID), "volumes")
		err = os.MkdirAll(blockDevicePath, 0755)
		Expect(err).ToNot(HaveOccurred())
		err = os.MkdirAll(fileSystemPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		deviceFile := filepath.Join(blockDevicePath, "file")
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(filepath.Join(tempDir, slicePath), 0755)
		Expect(err).ToNot(HaveOccurred())
		allowFile := filepath.Join(tempDir, slicePath, "devices.allow")
		listFile := filepath.Join(tempDir, slicePath, "devices.list")
		denyFile := filepath.Join(tempDir, slicePath, "devices.deny")
		_, err := os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(deviceFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())

		sourcePodBasePath = func(podUID types.UID) string {
			if podUID == blockSourcePodUID {
				return blockDevicePath
			}
			return fileSystemPath
		}
		diskFile := filepath.Join(fileSystemPath, "disk.img")
		_, err = os.Create(diskFile)
		Expect(err).ToNot(HaveOccurred())
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		fileSystemVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/filesystemvolume")
		blockVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/blockvolume")
		Expect(err).ToNot(HaveOccurred())
		targetFilePath := filepath.Join(targetPodPath, "filesystemvolume")
		mountCommand = func(sourcePath, targetPath string) ([]byte, error) {
			Expect(sourcePath).To(Equal(fileSystemPath))
			Expect(targetPath).To(Equal(targetFilePath))
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
		isBlockDevice = func(path string) (bool, error) {
			if strings.Contains(path, string(blockSourcePodUID)) {
				return true, nil
			}
			return false, fmt.Errorf("Not a block device")
		}

		vmi.Status.VolumeStatus = volumeStatuses
		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumeDevices")
		}
		blockDevicePath := filepath.Join(tempDir, string(blockSourcePodUID), "volumeDevices")
		fileSystemPath := filepath.Join(tempDir, string(fsSourcePodUID), "volumes")
		err = os.MkdirAll(blockDevicePath, 0755)
		Expect(err).ToNot(HaveOccurred())
		err = os.MkdirAll(fileSystemPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		deviceFile := filepath.Join(blockDevicePath, "file")
		slicePath := "slice"
		m.podIsolationDetector = &mockIsolationDetector{
			slice: slicePath,
		}
		err = os.MkdirAll(filepath.Join(tempDir, slicePath), 0755)
		Expect(err).ToNot(HaveOccurred())
		allowFile := filepath.Join(tempDir, slicePath, "devices.allow")
		listFile := filepath.Join(tempDir, slicePath, "devices.list")
		denyFile := filepath.Join(tempDir, slicePath, "devices.deny")
		_, err := os.Create(allowFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(denyFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(listFile)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(deviceFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())

		sourcePodBasePath = func(podUID types.UID) string {
			if podUID == blockSourcePodUID {
				return blockDevicePath
			}
			return fileSystemPath
		}
		diskFile := filepath.Join(fileSystemPath, "disk.img")
		_, err = os.Create(diskFile)
		Expect(err).ToNot(HaveOccurred())
		hotplugdisk.SetKubeletPodsDirectory(tempDir)
		targetPodPath := filepath.Join(tempDir, string(m.findVirtlauncherUID(vmi)), "volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		fileSystemVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/filesystemvolume")
		blockVolume := filepath.Join(tempDir, "/abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/blockvolume")
		Expect(err).ToNot(HaveOccurred())
		targetFilePath := filepath.Join(targetPodPath, "filesystemvolume")
		mountCommand = func(sourcePath, targetPath string) ([]byte, error) {
			Expect(sourcePath).To(Equal(fileSystemPath))
			Expect(targetPath).To(Equal(targetFilePath))
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

func (i *mockIsolationDetector) Detect(vm *v1.VirtualMachineInstance) (isolation.IsolationResult, error) {
	return isolation.NewIsolationResult(i.pid, i.slice, i.controller), i.err
}

func (i *mockIsolationDetector) DetectForSocket(vm *v1.VirtualMachineInstance, socket string) (isolation.IsolationResult, error) {
	return isolation.NewIsolationResult(1, tempDir, []string{}), nil
}

func (i *mockIsolationDetector) Whitelist(controller []string) isolation.PodIsolationDetector {
	return i
}

func (i *mockIsolationDetector) AdjustResources(vm *v1.VirtualMachineInstance) error {
	return nil
}
