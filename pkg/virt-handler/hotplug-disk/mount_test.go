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
	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

const (
	findmntByVolumeRes = "{\"filesystems\": [{\"target\":\"/%s\", \"source\":\"/dev/testvolume[%s]\", \"fstype\":\"xfs\", \"options\":\"rw,relatime,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota\"}]}"
)

var (
	tempDir              string
	orgIsoDetector       = isolationDetector
	orgDeviceBasePath    = deviceBasePath
	orgStatCommand       = statCommand
	orgMknodCommand      = mknodCommand
	orgSourcePodBasePath = sourcePodBasePath
	orgMountCommand      = mountCommand
	orgUnMountCommand    = unmountCommand
	orgIsMounted         = isMounted
	orgIsBlockDevice     = isBlockDevice
	orgFindMntByVolume   = findMntByVolume
	orgFindMntByDevice   = findMntByDevice
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
			mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:      tempDir,
			hotplugDiskManager: hotplugdisk.NewHotplugDiskWithOptions(tempDir),
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
		_ = os.RemoveAll(tempDir)
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
		m             *volumeMounter
		err           error
		vmi           *v1.VirtualMachineInstance
		targetPodPath string
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"
		activePods := make(map[types.UID]string, 0)
		activePods["abcd"] = "host"
		vmi.Status.ActivePods = activePods

		targetPodPath = filepath.Join(tempDir, "abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/testvolume")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		m = &volumeMounter{
			mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:      tempDir,
			skipSafetyCheck:    true,
			hotplugDiskManager: hotplugdisk.NewHotplugDiskWithOptions(tempDir),
		}

		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumes")
		}
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("6,6,0777,block special file"), nil
		}

	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		statCommand = orgStatCommand
		mknodCommand = orgMknodCommand
		isBlockDevice = orgIsBlockDevice
	})

	It("isBlockVolume should determine if we have a block volume", func() {
		err = os.RemoveAll(filepath.Join(tempDir, string(vmi.UID), "volumes"))
		Expect(err).ToNot(HaveOccurred())
		By("Passing empty UID, should return false")
		res := m.isBlockVolume("", "invalid")
		Expect(res).To(BeFalse())
		By("Not having the volume directory, should return false")
		res = m.isBlockVolume(vmi.UID, "invalid")
		Expect(res).To(BeFalse())
		By("Creating the volume directory, should return true")
		err = os.MkdirAll(filepath.Join(tempDir, string(vmi.UID), "volumes", "volume-test"), 0755)
		Expect(err).ToNot(HaveOccurred())
		isBlockDevice = func(path string) (bool, error) {
			if strings.Contains(path, string(vmi.UID)) {
				return true, nil
			}
			return false, fmt.Errorf("Not a block device")
		}
		res = m.isBlockVolume(vmi.UID, "volume-test")
		Expect(res).To(BeTrue())
	})

	It("findVirtlauncherUID should find the right UID", func() {
		res := m.findVirtlauncherUID(vmi)
		Expect(res).To(BeEquivalentTo("abcd"))
		vmi.Status.ActivePods["abcde"] = "host"
		res = m.findVirtlauncherUID(vmi)
		Expect(res).To(BeEquivalentTo("abcd"))
		vmi.Status.ActivePods["abcdef"] = "host"
		err = os.MkdirAll(filepath.Join(tempDir, "abcdef/volumes/kubernetes.io~empty-dir/hotplug-disks"), 0755)
		res = m.findVirtlauncherUID(vmi)
		Expect(res).To(BeEquivalentTo(""))
	})

	It("getSourceMajorMinor should return an error if no uid", func() {
		vmi.UID = ""
		_, _, _, err := m.getSourceMajorMinor("fghij", "invalid")
		Expect(err).To(HaveOccurred())
	})

	It("getSourceMajorMinor should return file if exists", func() {
		deviceFile := filepath.Join(tempDir, "fghij", "volumes", "test-volume", "file")
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		err = ioutil.WriteFile(deviceFile, []byte("test"), 0644)
		Expect(err).ToNot(HaveOccurred())
		major, minor, perm, err := m.getSourceMajorMinor("fghij", "test-volume")
		Expect(err).ToNot(HaveOccurred())
		Expect(major).To(Equal(int64(6)))
		Expect(minor).To(Equal(int64(6)))
		Expect(perm).To(Equal("0777"))
	})

	It("getSourceMajorMinor should return error if file doesn't exists", func() {
		deviceFile := filepath.Join(tempDir, "fghij", "volumes", "file")
		err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
		Expect(err).ToNot(HaveOccurred())
		major, minor, perm, err := m.getSourceMajorMinor("fghij", "test-volume")
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
		table.Entry("Should not return values if stat command doesn't int major", func(fileName string) ([]byte, error) {
			return []byte("kk,32,0664,block special file"), nil
		}, -1, -1, "", true),
		table.Entry("Should not return values if stat command doesn't int minor", func(fileName string) ([]byte, error) {
			return []byte("254,gg,0664,block special file"), nil
		}, -1, -1, "", true),
	)

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

})

var _ = Describe("HotplugVolume filesystem volumes", func() {
	var (
		m             *volumeMounter
		err           error
		vmi           *v1.VirtualMachineInstance
		record        *vmiMountTargetRecord
		targetPodPath string
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"
		activePods := make(map[types.UID]string, 0)
		activePods["abcd"] = "host"
		vmi.Status.ActivePods = activePods

		targetPodPath = filepath.Join(tempDir, "abcd/volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		record = &vmiMountTargetRecord{}

		m = &volumeMounter{
			mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:      tempDir,
			hotplugDiskManager: hotplugdisk.NewHotplugDiskWithOptions(tempDir),
		}

		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumes")
		}
		isolationDetector = func(path string) isolation.PodIsolationDetector {
			return &mockIsolationDetector{
				pid: 1,
			}
		}
	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
		findMntByVolume = orgFindMntByVolume
		deviceBasePath = orgDeviceBasePath
		sourcePodBasePath = orgSourcePodBasePath
		mountCommand = orgMountCommand
		unmountCommand = orgUnMountCommand
		isMounted = orgIsMounted
		isolationDetector = orgIsoDetector
	})

	It("getSourcePodFile should find the disk.img file, if it exists", func() {
		path := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(path, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return path
		}
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(fmt.Sprintf(findmntByVolumeRes, "pvc", path)), nil
		}
		diskFile := filepath.Join(path, "disk.img")
		_, err := os.Create(diskFile)
		Expect(err).ToNot(HaveOccurred())
		file, err := m.getSourcePodFilePath("ghfjk", vmi, "pvc")
		Expect(err).ToNot(HaveOccurred())
		Expect(file).To(Equal(path))
	})

	It("getSourcePodFile should return error if no UID", func() {
		_, err := m.getSourcePodFilePath("", vmi, "")
		Expect(err).To(HaveOccurred())
	})

	It("getSourcePodFile should return error if disk.img doesn't exist", func() {
		path := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(path, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return path
		}
		Expect(err).ToNot(HaveOccurred())
		_, err := m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).To(HaveOccurred())
	})

	It("getSourcePodFile should return error if iso detection returns error", func() {
		expectedPath := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(expectedPath, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return expectedPath
		}
		isolationDetector = func(path string) isolation.PodIsolationDetector {
			return &mockIsolationDetector{
				pid: 40,
			}
		}

		Expect(err).ToNot(HaveOccurred())
		_, err := m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("isolation error"))
	})

	It("getSourcePodFile should return error if find mounts returns error", func() {
		expectedPath := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(expectedPath, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return expectedPath
		}
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(""), fmt.Errorf("findmnt error")
		}

		Expect(err).ToNot(HaveOccurred())
		_, err := m.getSourcePodFilePath("ghfjk", vmi, "")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("findmnt error"))
	})

	It("getSourcePodFile should return the findmnt value", func() {
		expectedPath := filepath.Join(tempDir, "ghfjk", "volumes")
		err = os.MkdirAll(expectedPath, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return expectedPath
		}
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(fmt.Sprintf(findmntByVolumeRes, "pvc", expectedPath)), nil
		}

		Expect(err).ToNot(HaveOccurred())
		res, err := m.getSourcePodFilePath("ghfjk", vmi, "pvc")
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(expectedPath))
	})

	It("should properly mount and unmount filesystem", func() {
		sourcePodUID := "ghfjk"
		path := filepath.Join(tempDir, sourcePodUID, "volumes", "disk.img")
		err = os.MkdirAll(path, 0755)
		sourcePodBasePath = func(podUID types.UID) string {
			return path
		}
		diskFile := filepath.Join(path, "disk.img")
		_, err := os.Create(diskFile)
		Expect(err).ToNot(HaveOccurred())
		findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
			return []byte(fmt.Sprintf(findmntByVolumeRes, "testvolume", path)), nil
		}
		targetFilePath := filepath.Join(targetPodPath, "testvolume.img")
		mountCommand = func(sourcePath, targetPath string) ([]byte, error) {
			Expect(sourcePath).To(Equal(diskFile))
			Expect(targetPath).To(Equal(targetFilePath))
			return []byte("Success"), nil
		}

		err = m.mountFileSystemHotplugVolume(vmi, "testvolume", types.UID(sourcePodUID), record)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(record.MountTargetEntries)).To(Equal(1))
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
		m             *volumeMounter
		err           error
		vmi           *v1.VirtualMachineInstance
		targetPodPath string
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "hotplug-volume-test")
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"
		activePods := make(map[types.UID]string, 0)
		activePods["abcd"] = "host"
		vmi.Status.ActivePods = activePods

		targetPodPath = filepath.Join(tempDir, "abcd/volumes/kubernetes.io~empty-dir/hotplug-disks")
		err = os.MkdirAll(targetPodPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		m = &volumeMounter{
			mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:      tempDir,
			skipSafetyCheck:    true,
			hotplugDiskManager: hotplugdisk.NewHotplugDiskWithOptions(tempDir),
		}

		deviceBasePath = func(sourceUID types.UID) string {
			return filepath.Join(tempDir, string(sourceUID), "volumes")
		}
		statCommand = func(fileName string) ([]byte, error) {
			return []byte("6,6,0777,block special file"), nil
		}
		isolationDetector = func(path string) isolation.PodIsolationDetector {
			return &mockIsolationDetector{
				pid: 1,
			}
		}
	})

	AfterEach(func() {
		_ = os.RemoveAll(tempDir)
		deviceBasePath = orgDeviceBasePath
		sourcePodBasePath = orgSourcePodBasePath
		mountCommand = orgMountCommand
		unmountCommand = orgUnMountCommand
		isMounted = orgIsMounted
		statCommand = orgStatCommand
		mknodCommand = orgMknodCommand
		isBlockDevice = orgIsBlockDevice
		findMntByVolume = orgFindMntByVolume
	})

	It("Should not do anything if vmi has no hotplug volumes", func() {
		volumeStatuses := make([]v1.VolumeStatus, 0)
		volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
			Name: "permanent",
		})
		vmi.Status.VolumeStatus = volumeStatuses
		Expect(m.Mount(vmi)).To(BeNil())
	})

})

type mockIsolationDetector struct {
	pid        int
	ppid       int
	slice      string
	controller []string
	err        error
}

func (i *mockIsolationDetector) Detect(_ *v1.VirtualMachineInstance) (isolation.IsolationResult, error) {
	return isolation.NewIsolationResult(i.pid, i.ppid), i.err
}

func (i *mockIsolationDetector) DetectForSocket(_ *v1.VirtualMachineInstance, _ string) (isolation.IsolationResult, error) {
	if i.pid == 1 {
		return isolation.NewIsolationResult(i.pid, i.ppid), nil
	}
	return nil, fmt.Errorf("isolation error")
}

func (i *mockIsolationDetector) Whitelist(_ []string) isolation.PodIsolationDetector {
	return i
}

func (i *mockIsolationDetector) AdjustResources(_ *v1.VirtualMachineInstance) error {
	return nil
}
