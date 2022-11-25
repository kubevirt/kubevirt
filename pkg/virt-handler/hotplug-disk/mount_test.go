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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/api"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"

	hotplugdisk "kubevirt.io/kubevirt/pkg/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

const (
	findmntByVolumeRes = "{\"filesystems\": [{\"target\":\"/%s\", \"source\":\"/dev/testvolume[%s]\", \"fstype\":\"xfs\", \"options\":\"rw,relatime,seclabel,attr2,inode64,logbufs=8,logbsize=32k,noquota\"}]}"
)

var (
	tempDir                string
	tmpDirSafe             *safepath.Path
	orgIsoDetector         = isolationDetector
	orgDeviceBasePath      = deviceBasePath
	orgStatSourceCommand   = statSourceDevice
	orgStatCommand         = statDevice
	orgMknodCommand        = mknodCommand
	orgMountCommand        = mountCommand
	orgUnMountCommand      = unmountCommand
	orgIsMounted           = isMounted
	orgIsBlockDevice       = isBlockDevice
	orgFindMntByVolume     = findMntByVolume
	orgFindMntByDevice     = findMntByDevice
	orgNodeIsolationResult = nodeIsolationResult
	orgParentPathForMount  = parentPathForMount
)

var _ = Describe("HotplugVolume", func() {
	var (
		ctrl               *gomock.Controller
		expectedCgroupRule *devices.Rule
		cgroupManagerMock  *cgroup.MockManager
		ownershipManager   *diskutils.MockOwnershipManagerInterface
	)

	expectCgroupRule := func(t devices.Type, major, minor int64, allow bool) {
		expectedCgroupRule = &devices.Rule{
			Type:  t,
			Major: major,
			Minor: minor,
			Allow: allow,
		}
	}

	areRulesEqual := func(rule1, rule2 *devices.Rule) bool {
		Expect(rule1).ToNot(BeNil())
		Expect(rule2).ToNot(BeNil())

		return rule1.Type == rule2.Type &&
			rule1.Major == rule2.Major &&
			rule1.Minor == rule2.Minor &&
			rule1.Allow == rule2.Allow
	}

	getCgroupManager = func(_ *v1.VirtualMachineInstance) (cgroup.Manager, error) {
		return cgroupManagerMock, nil
	}

	cgroupMockSet := func(r *runc_configs.Resources) {
		if expectedCgroupRule == nil {
			return
		}

		foundExpectedRule := false
		for _, deviceRule := range r.Devices {
			if areRulesEqual(deviceRule, expectedCgroupRule) {
				foundExpectedRule = true
				break
			}
		}

		Expect(foundExpectedRule).To(BeTrue(), "expected rule needs to be applied as a cgroup rule")
	}

	setExpectedCgroupRuns := func(runsExpected int) {
		cgroupManagerMock.EXPECT().Set(gomock.Any()).Do(cgroupMockSet).Times(runsExpected)
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		// cgroups mock setup
		cgroupManagerMock = cgroup.NewMockManager(ctrl)
		cgroupManagerMock.EXPECT().GetCgroupVersion().AnyTimes()
		expectedCgroupRule = nil
		ownershipManager = diskutils.NewMockOwnershipManagerInterface(ctrl)

		nodeIsolationResult = func() isolation.IsolationResult {
			return isolation.NewIsolationResult(os.Getpid(), os.Getppid())
		}

		parentPathForMount = func(
			parent isolation.IsolationResult,
			_ isolation.IsolationResult,
			findmntInfo FindmntInfo,
		) (*safepath.Path, error) {
			path, err := parent.MountRoot()
			Expect(err).ToNot(HaveOccurred())
			return path.AppendAndResolveWithRelativeRoot(findmntInfo.GetSourcePath())
		}
	})

	Context("mount target records", func() {
		var (
			m      *volumeMounter
			err    error
			vmi    *v1.VirtualMachineInstance
			record *vmiMountTargetRecord
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			tmpDirSafe, err = safepath.JoinAndResolveWithRelativeRoot(tempDir)
			Expect(err).ToNot(HaveOccurred())
			vmi = api.NewMinimalVMI("fake-vmi")
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
			bytes, err := os.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
			Expect(err).ToNot(HaveOccurred())
			Expect(bytes).To(Equal(expectedBytes))
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
			Expect(equality.Semantic.DeepEqual(*res, *record)).To(BeTrue())
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
			Expect(equality.Semantic.DeepEqual(*res, *cacheRecord)).To(BeTrue())
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
			err := os.WriteFile(filepath.Join(tempDir, "test"), []byte("test"), 0644)
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

	Context("block devices", func() {
		var (
			m             *volumeMounter
			err           error
			vmi           *v1.VirtualMachineInstance
			record        *vmiMountTargetRecord
			targetPodPath string
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			tmpDirSafe, err = safepath.JoinAndResolveWithRelativeRoot(tempDir)
			Expect(err).ToNot(HaveOccurred())
			vmi = api.NewMinimalVMI("fake-vmi")
			vmi.UID = "1234"
			activePods := make(map[types.UID]string, 0)
			activePods["abcd"] = "host"
			vmi.Status.ActivePods = activePods

			targetPodPath = filepath.Join(tempDir, "abcd/volumes/kubernetes.io~empty-dir/hotplug-disks/")
			err = os.MkdirAll(targetPodPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			record = &vmiMountTargetRecord{}

			m = &volumeMounter{
				mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
				mountStateDir:      tempDir,
				skipSafetyCheck:    true,
				hotplugDiskManager: hotplugdisk.NewHotplugDiskWithOptions(tempDir),
				ownershipManager:   ownershipManager,
			}

			deviceBasePath = func(sourceUID types.UID) (*safepath.Path, error) {
				return newDir(tempDir, string(sourceUID), "volumes")
			}
			statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0777, 123456), nil
			}

		})

		AfterEach(func() {
			deviceBasePath = orgDeviceBasePath
			statSourceDevice = orgStatSourceCommand
			mknodCommand = orgMknodCommand
			isBlockDevice = orgIsBlockDevice
			nodeIsolationResult = orgNodeIsolationResult
		})

		It("isBlockVolume should determine if we have a block volume", func() {
			err = os.RemoveAll(filepath.Join(tempDir, string(vmi.UID), "volumes"))
			Expect(err).ToNot(HaveOccurred())
			vmi.Status.VolumeStatus = make([]v1.VolumeStatus, 0)
			By("Passing invalid volume, should return false")
			res := m.isBlockVolume(&vmi.Status, "invalid")
			Expect(res).To(BeFalse())
			By("Not having persistent volume info, should return false")
			vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
				Name: "test",
			})
			res = m.isBlockVolume(&vmi.Status, "test")
			Expect(res).To(BeFalse())
			By("Not having volume mode, should return false")
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo = &v1.PersistentVolumeClaimInfo{}
			res = m.isBlockVolume(&vmi.Status, "test")
			Expect(res).To(BeFalse())
			By("Having volume mode be filesystem, should return false")
			fs := k8sv1.PersistentVolumeFilesystem
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo = &v1.PersistentVolumeClaimInfo{
				VolumeMode: &fs,
			}
			res = m.isBlockVolume(&vmi.Status, "test")
			Expect(res).To(BeFalse())
			By("Having volume mode be block, should return true")
			block := k8sv1.PersistentVolumeBlock
			vmi.Status.VolumeStatus[0].PersistentVolumeClaimInfo = &v1.PersistentVolumeClaimInfo{
				VolumeMode: &block,
			}
			res = m.isBlockVolume(&vmi.Status, "test")
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

		It("mountBlockHotplugVolume and unmountBlockHotplugVolumes should make appropriate calls", func() {
			By("Initializing cgroup mock files")
			blockSourcePodUID := types.UID("fghij")
			targetPodPath := hotplugdisk.TargetPodBasePath(tempDir, m.findVirtlauncherUID(vmi))
			err = os.MkdirAll(targetPodPath, 0755)
			Expect(err).ToNot(HaveOccurred())
			deviceFile, err := newFile(tempDir, string(blockSourcePodUID), "volumes", "testvolume", "file")
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(unsafepath.UnsafeAbsolute(deviceFile.Raw()), []byte("test"), 0644)).To(Succeed())

			vmi.Status.VolumeStatus = []v1.VolumeStatus{{Name: "testvolume", HotplugVolume: &v1.HotplugVolumeStatus{}}}
			targetDevicePath, err := newFile(targetPodPath, "testvolume")
			Expect(err).ToNot(HaveOccurred())
			ownershipManager.EXPECT().SetFileOwnership(targetDevicePath)
			statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0666, 123456), nil
			}
			statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0666, 123456), nil
			}
			isBlockDevice = func(fileName *safepath.Path) (bool, error) {
				return true, nil
			}

			By("Mounting and validating expected rule is set")
			setExpectedCgroupRuns(2)
			expectCgroupRule(devices.BlockDevice, 482, 64, true)
			err = m.mountBlockHotplugVolume(vmi, "testvolume", blockSourcePodUID, record)
			Expect(err).ToNot(HaveOccurred())

			By("Unmounting, we verify the reverse process happens")
			expectCgroupRule(devices.BlockDevice, 482, 64, false)
			err = m.unmountBlockHotplugVolumes(deviceFile, vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("getSourceMajorMinor should return an error if no uid", func() {
			vmi.UID = ""
			_, _, err := m.getSourceMajorMinor("fghij", "invalid")
			Expect(err).To(HaveOccurred())
		})

		It("getSourceMajorMinor should succeed if file exists", func() {
			deviceFile := filepath.Join(tempDir, "fghij", "volumes", "test-volume", "file")
			err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(deviceFile, []byte("test"), 0644)
			Expect(err).ToNot(HaveOccurred())
			numbers, perm, err := m.getSourceMajorMinor("fghij", "test-volume")
			Expect(err).ToNot(HaveOccurred())
			Expect(unix.Major(numbers)).To(Equal(uint32(482)))
			Expect(unix.Minor(numbers)).To(Equal(uint32(64)))
			Expect(perm & 0777).To(Equal(os.FileMode(0777)))
		})

		It("getSourceMajorMinor should return error if file doesn't exists", func() {
			deviceFile := filepath.Join(tempDir, "fghij", "volumes", "file")
			err = os.MkdirAll(filepath.Dir(deviceFile), 0755)
			Expect(err).ToNot(HaveOccurred())
			_, _, err := m.getSourceMajorMinor("fghij", "test-volume")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
		})

		DescribeTable("Should return proper values", func(stat func(safePath *safepath.Path) (os.FileInfo, error), major, minor uint32, perm os.FileMode, expectErr bool) {
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
			Entry("Should return values if stat command successful", func(safePath *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0664, 123456), nil
			}, uint32(482), uint32(64), os.FileMode(0664), false),
			Entry("Should not return error if stat command errors", func(safePath *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0664, 123456), fmt.Errorf("Error")
			}, uint32(0), uint32(0), os.FileMode(0), true),
		)

		It("should write properly to allow/deny files if able", func() {
			setExpectedCgroupRuns(2)
			expectCgroupRule(devices.BlockDevice, 482, 64, true)
			err = m.allowBlockMajorMinor(123456, cgroupManagerMock)
			Expect(err).ToNot(HaveOccurred())

			expectCgroupRule(devices.BlockDevice, 482, 64, false)
			err = m.removeBlockMajorMinor(123456, cgroupManagerMock)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should remove the block device and permissions on unmount", func() {
			By("Initializing cgroup mock files")
			statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0664, 123456), nil
			}
			deviceFileName, err := newFile(tempDir, "devicefile")
			Expect(err).ToNot(HaveOccurred())

			By("Mounting and validating expected rule is set")
			setExpectedCgroupRuns(1)
			expectCgroupRule(devices.BlockDevice, 482, 64, false)
			err = m.unmountBlockHotplugVolumes(deviceFileName, vmi)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should return error if deviceFile doesn' exist", func() {
			statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0664, 123456), nil
			}
			deviceFileName, err := newFile(tempDir, "devicefile")
			Expect(err).ToNot(HaveOccurred())
			os.Remove(unsafepath.UnsafeAbsolute(deviceFileName.Raw()))
			err = m.unmountBlockHotplugVolumes(deviceFileName, vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no such file or directory"))
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

	})

	Context("filesystem volumes", func() {
		var (
			m             *volumeMounter
			err           error
			vmi           *v1.VirtualMachineInstance
			record        *vmiMountTargetRecord
			targetPodPath *safepath.Path
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			tmpDirSafe, err = safepath.JoinAndResolveWithRelativeRoot(tempDir)
			Expect(err).ToNot(HaveOccurred())

			volumeDir, err := newDir(tempDir, "volumes")
			Expect(err).ToNot(HaveOccurred())
			vmi = api.NewMinimalVMI("fake-vmi")
			vmi.UID = "1234"
			activePods := make(map[types.UID]string, 0)
			activePods["abcd"] = "host"
			vmi.Status.ActivePods = activePods

			targetPodPath, err = newDir(tempDir, "abcd/volumes/kubernetes.io~empty-dir/hotplug-disks")
			Expect(err).ToNot(HaveOccurred())

			record = &vmiMountTargetRecord{}

			m = &volumeMounter{
				mountRecords:       make(map[types.UID]*vmiMountTargetRecord),
				mountStateDir:      tempDir,
				hotplugDiskManager: hotplugdisk.NewHotplugDiskWithOptions(tempDir),
				ownershipManager:   ownershipManager,
			}

			deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
				return volumeDir, nil
			}
			isolationDetector = func(path string) isolation.PodIsolationDetector {
				return &mockIsolationDetector{
					pid: os.Getpid(),
				}
			}
		})

		AfterEach(func() {
			findMntByVolume = orgFindMntByVolume
			deviceBasePath = orgDeviceBasePath
			mountCommand = orgMountCommand
			unmountCommand = orgUnMountCommand
			isMounted = orgIsMounted
			isolationDetector = orgIsoDetector
		})

		It("getSourcePodFile should find the disk.img file, if it exists", func() {
			path, err := newDir(tempDir, "ghfjk", "volumes")
			Expect(err).ToNot(HaveOccurred())
			findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
				return []byte(fmt.Sprintf(findmntByVolumeRes, "pvc", unsafepath.UnsafeAbsolute(path.Raw()))), nil
			}
			_, err = newFile(unsafepath.UnsafeAbsolute(path.Raw()), "disk.img")
			Expect(err).ToNot(HaveOccurred())
			file, err := m.getSourcePodFilePath("ghfjk", vmi, "pvc")
			Expect(err).ToNot(HaveOccurred())
			Expect(unsafepath.UnsafeRelative(file.Raw())).To(Equal(unsafepath.UnsafeAbsolute(path.Raw())))
		})

		It("getSourcePodFile should return error if no UID", func() {
			_, err := m.getSourcePodFilePath("", vmi, "")
			Expect(err).To(HaveOccurred())
		})

		It("getSourcePodFile should return error if disk.img doesn't exist", func() {
			_, err := newDir(tempDir, "ghfjk", "volumes")
			Expect(err).ToNot(HaveOccurred())
			_, err = m.getSourcePodFilePath("ghfjk", vmi, "")
			Expect(err).To(HaveOccurred())
		})

		It("getSourcePodFile should return error if iso detection returns error", func() {
			_, err := newDir(tempDir, "ghfjk", "volumes")
			Expect(err).ToNot(HaveOccurred())
			isolationDetector = func(path string) isolation.PodIsolationDetector {
				return &mockIsolationDetector{
					pid: 9999,
				}
			}
			_, err = m.getSourcePodFilePath("ghfjk", vmi, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("isolation error"))
		})

		It("getSourcePodFile should return error if find mounts returns error", func() {
			_, err := newDir(tempDir, "ghfjk", "volumes")
			Expect(err).ToNot(HaveOccurred())
			findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
				return []byte(""), fmt.Errorf("findmnt error")
			}
			_, err = m.getSourcePodFilePath("ghfjk", vmi, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("findmnt error"))
		})

		It("getSourcePodFile should return the findmnt value", func() {
			expectedPath, err := newDir(tempDir, "ghfjk", "volumes")
			Expect(err).ToNot(HaveOccurred())
			findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
				return []byte(fmt.Sprintf(findmntByVolumeRes, "pvc", unsafepath.UnsafeAbsolute(expectedPath.Raw()))), nil
			}
			res, err := m.getSourcePodFilePath("ghfjk", vmi, "pvc")
			Expect(err).ToNot(HaveOccurred())
			Expect(unsafepath.UnsafeRelative(res.Raw())).To(Equal(unsafepath.UnsafeAbsolute(expectedPath.Raw())))
		})

		It("should properly mount and unmount filesystem", func() {
			sourcePodUID := "ghfjk"
			path, err := newDir(tempDir, sourcePodUID, "volumes")
			Expect(err).ToNot(HaveOccurred())
			diskFile, err := newFile(unsafepath.UnsafeAbsolute(path.Raw()), "disk.img")
			Expect(err).ToNot(HaveOccurred())
			findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
				return []byte(fmt.Sprintf(findmntByVolumeRes, "testvolume", unsafepath.UnsafeAbsolute(path.Raw()))), nil
			}
			targetFilePath, err := newFile(unsafepath.UnsafeAbsolute(targetPodPath.Raw()), "testvolume.img")
			Expect(err).ToNot(HaveOccurred())
			mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
				Expect(unsafepath.UnsafeRelative(sourcePath.Raw())).To(Equal(unsafepath.UnsafeAbsolute(diskFile.Raw())))
				Expect(targetPath).To(Equal(targetFilePath))
				return []byte("Success"), nil
			}
			ownershipManager.EXPECT().SetFileOwnership(targetFilePath)

			err = m.mountFileSystemHotplugVolume(vmi, "testvolume", types.UID(sourcePodUID), record, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(record.MountTargetEntries).To(HaveLen(1))
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

	Context("volumes", func() {
		var (
			m             *volumeMounter
			err           error
			vmi           *v1.VirtualMachineInstance
			targetPodPath string
		)

		BeforeEach(func() {
			tempDir = GinkgoT().TempDir()
			tmpDirSafe, err = safepath.JoinAndResolveWithRelativeRoot(tempDir)
			Expect(err).ToNot(HaveOccurred())
			vmi = api.NewMinimalVMI("fake-vmi")
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
				ownershipManager:   ownershipManager,
			}

			deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
				return newDir(tempDir, string(podUID), "volumes")
			}
			statSourceDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0777, 123456), nil
			}
			statDevice = func(fileName *safepath.Path) (os.FileInfo, error) {
				return fakeStat(true, 0777, 123456), nil
			}
			isolationDetector = func(path string) isolation.PodIsolationDetector {
				return &mockIsolationDetector{
					pid: os.Getpid(),
				}
			}
		})

		AfterEach(func() {
			deviceBasePath = orgDeviceBasePath
			mountCommand = orgMountCommand
			unmountCommand = orgUnMountCommand
			isMounted = orgIsMounted
			statSourceDevice = orgStatSourceCommand
			statDevice = orgStatCommand
			mknodCommand = orgMknodCommand
			isBlockDevice = orgIsBlockDevice
			findMntByVolume = orgFindMntByVolume
		})

		It("mount and umount should work for filesystem volumes", func() {
			setExpectedCgroupRuns(2)

			sourcePodUID := types.UID("klmno")
			volumeStatuses := make([]v1.VolumeStatus, 0)
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "permanent",
			})
			block := k8sv1.PersistentVolumeBlock
			fs := k8sv1.PersistentVolumeFilesystem
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "filesystemvolume",
				PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
					VolumeMode: &fs,
				},
				HotplugVolume: &v1.HotplugVolumeStatus{
					AttachPodName: "pod",
					AttachPodUID:  sourcePodUID,
				},
			})
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "blockvolume",
				PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
					VolumeMode: &block,
				},
				HotplugVolume: &v1.HotplugVolumeStatus{
					AttachPodName: "pod",
					AttachPodUID:  sourcePodUID,
				},
			})
			vmi.Status.VolumeStatus = volumeStatuses
			deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
				return newDir(tempDir, string(podUID), "volumeDevices")
			}
			blockDevicePath, err := newDir(tempDir, string(sourcePodUID), "volumeDevices", "blockvolume")
			Expect(err).ToNot(HaveOccurred())
			fileSystemPath, err := newDir(tempDir, string(sourcePodUID), "volumes")
			Expect(err).ToNot(HaveOccurred())

			deviceFile, err := newFile(unsafepath.UnsafeAbsolute(blockDevicePath.Raw()), "blockvolumefile")
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(unsafepath.UnsafeAbsolute(deviceFile.Raw()), []byte("test"), 0644)
			Expect(err).ToNot(HaveOccurred())

			findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
				return []byte(fmt.Sprintf(findmntByVolumeRes, "filesystemvolume", unsafepath.UnsafeAbsolute(fileSystemPath.Raw()))), nil
			}

			diskFile, err := newFile(unsafepath.UnsafeAbsolute(fileSystemPath.Raw()), "disk.img")
			Expect(err).ToNot(HaveOccurred())
			mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
				_, err := newFile(unsafepath.UnsafeAbsolute(basePath.Raw()), deviceName)
				Expect(err).ToNot(HaveOccurred())
				return nil
			}

			blockVolume := filepath.Join(targetPodPath, "blockvolume")
			targetFilePath := filepath.Join(targetPodPath, "filesystemvolume.img")
			isBlockDevice = func(path *safepath.Path) (bool, error) {
				return strings.Contains(unsafepath.UnsafeAbsolute(path.Raw()), "blockvolume"), nil
			}
			mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
				Expect(unsafepath.UnsafeRelative(sourcePath.Raw())).To(Equal(unsafepath.UnsafeAbsolute(diskFile.Raw())))
				Expect(unsafepath.UnsafeAbsolute(targetPath.Raw())).To(Equal(targetFilePath))
				return []byte("Success"), nil
			}

			expectedPaths := []string{targetFilePath, blockVolume}
			capturedPaths := []string{}

			ownershipManager.EXPECT().SetFileOwnership(gomock.Any()).Times(2).DoAndReturn(func(path *safepath.Path) error {
				capturedPaths = append(capturedPaths, unsafepath.UnsafeAbsolute(path.Raw()))
				return nil
			})

			err = m.Mount(vmi)
			Expect(err).ToNot(HaveOccurred())
			By("Verifying there are 2 records in tempDir/1234")
			record := &vmiMountTargetRecord{
				MountTargetEntries: []vmiMountTargetEntry{
					{
						TargetFile: targetFilePath,
					},
					{
						TargetFile: blockVolume,
					},
				},
				UsesSafePaths: true,
			}
			expectedBytes, err := json.Marshal(record)
			Expect(err).ToNot(HaveOccurred())
			bytes, err := os.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
			Expect(err).ToNot(HaveOccurred())
			Expect(bytes).To(Equal(expectedBytes))
			_, err = os.Stat(targetFilePath)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(blockVolume)
			Expect(err).ToNot(HaveOccurred())
			Expect(capturedPaths).To(ContainElements(expectedPaths))

			volumeStatuses = make([]v1.VolumeStatus, 0)
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "permanent",
			})
			vmi.Status.VolumeStatus = volumeStatuses
			err = m.Unmount(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
			Expect(err).To(HaveOccurred(), "record file still exists %s", filepath.Join(tempDir, string(vmi.UID)))
			_, err = os.Stat(targetFilePath)
			Expect(err).To(HaveOccurred(), "filesystem volume file still exists %s", targetFilePath)
			_, err = os.Stat(blockVolume)
			Expect(err).To(HaveOccurred(), "block device volume still exists %s", blockVolume)
		})

		It("Should not do anything if vmi has no hotplug volumes", func() {
			volumeStatuses := make([]v1.VolumeStatus, 0)
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "permanent",
			})
			vmi.Status.VolumeStatus = volumeStatuses
			Expect(m.Mount(vmi)).To(Succeed())
		})

		It("unmountAll should cleanup regardless of vmi volumestatuses", func() {
			setExpectedCgroupRuns(2)

			sourcePodUID := types.UID("klmno")
			volumeStatuses := make([]v1.VolumeStatus, 0)
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "permanent",
			})
			block := k8sv1.PersistentVolumeBlock
			fs := k8sv1.PersistentVolumeFilesystem
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "filesystemvolume",
				PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
					VolumeMode: &fs,
				},
				HotplugVolume: &v1.HotplugVolumeStatus{
					AttachPodName: "pod",
					AttachPodUID:  sourcePodUID,
				},
			})
			volumeStatuses = append(volumeStatuses, v1.VolumeStatus{
				Name: "blockvolume",
				PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
					VolumeMode: &block,
				},
				HotplugVolume: &v1.HotplugVolumeStatus{
					AttachPodName: "pod",
					AttachPodUID:  sourcePodUID,
				},
			})
			vmi.Status.VolumeStatus = volumeStatuses
			deviceBasePath = func(podUID types.UID) (*safepath.Path, error) {
				return newDir(tempDir, string(podUID), "volumeDevices")
			}
			blockDevicePath, err := newDir(tempDir, string(sourcePodUID), "volumeDevices", "blockvolume")
			Expect(err).ToNot(HaveOccurred())
			fileSystemPath, err := newDir(tempDir, string(sourcePodUID), "volumes")
			Expect(err).ToNot(HaveOccurred())

			deviceFile, err := newFile(unsafepath.UnsafeAbsolute(blockDevicePath.Raw()), "blockvolumefile")
			Expect(err).ToNot(HaveOccurred())
			err = os.WriteFile(unsafepath.UnsafeAbsolute(deviceFile.Raw()), []byte("test"), 0644)
			Expect(err).ToNot(HaveOccurred())

			findMntByVolume = func(volumeName string, pid int) ([]byte, error) {
				return []byte(fmt.Sprintf(findmntByVolumeRes, "filesystemvolume", unsafepath.UnsafeAbsolute(fileSystemPath.Raw()))), nil
			}

			diskFile, err := newFile(unsafepath.UnsafeAbsolute(fileSystemPath.Raw()), "disk.img")
			Expect(err).ToNot(HaveOccurred())
			mknodCommand = func(basePath *safepath.Path, deviceName string, dev uint64, blockDevicePermissions os.FileMode) error {
				_, err := newFile(unsafepath.UnsafeAbsolute(basePath.Raw()), deviceName)
				Expect(err).ToNot(HaveOccurred())
				return nil
			}

			blockVolume := filepath.Join(targetPodPath, "blockvolume")
			targetFilePath := filepath.Join(targetPodPath, "filesystemvolume.img")
			isBlockDevice = func(path *safepath.Path) (bool, error) {
				return strings.Contains(unsafepath.UnsafeAbsolute(path.Raw()), "blockvolume"), nil
			}
			mountCommand = func(sourcePath, targetPath *safepath.Path) ([]byte, error) {
				Expect(unsafepath.UnsafeRelative(sourcePath.Raw())).To(Equal(unsafepath.UnsafeAbsolute(diskFile.Raw())))
				Expect(unsafepath.UnsafeAbsolute(targetPath.Raw())).To(Equal(targetFilePath))
				return []byte("Success"), nil
			}

			expectedPaths := []string{targetFilePath, blockVolume}
			capturedPaths := []string{}

			ownershipManager.EXPECT().SetFileOwnership(gomock.Any()).Times(2).DoAndReturn(func(path *safepath.Path) error {
				capturedPaths = append(capturedPaths, unsafepath.UnsafeAbsolute(path.Raw()))
				return nil
			})

			err = m.Mount(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying there are 2 records in tempDir/1234")
			record := &vmiMountTargetRecord{
				MountTargetEntries: []vmiMountTargetEntry{
					{
						TargetFile: targetFilePath,
					},
					{
						TargetFile: blockVolume,
					},
				},
				UsesSafePaths: true,
			}
			expectedBytes, err := json.Marshal(record)
			Expect(err).ToNot(HaveOccurred())
			bytes, err := os.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
			Expect(err).ToNot(HaveOccurred())
			Expect(bytes).To(Equal(expectedBytes))
			_, err = os.Stat(targetFilePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(capturedPaths).To(ContainElements(expectedPaths))

			err = m.UnmountAll(vmi)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.ReadFile(filepath.Join(tempDir, string(vmi.UID)))
			Expect(err).To(HaveOccurred(), "record file still exists %s", filepath.Join(tempDir, string(vmi.UID)))
			_, err = os.Stat(targetFilePath)
			Expect(err).To(HaveOccurred(), "filesystem volume file still exists %s", targetFilePath)
			_, err = os.Stat(blockVolume)
			Expect(err).To(HaveOccurred(), "block device volume still exists %s", blockVolume)
		})
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
	if i.pid != 9999 {
		return isolation.NewIsolationResult(i.pid, i.ppid), nil
	}
	return nil, fmt.Errorf("isolation error")
}

func (i *mockIsolationDetector) Allowlist(_ []string) isolation.PodIsolationDetector {
	return i
}

func (i *mockIsolationDetector) AdjustResources(_ *v1.VirtualMachineInstance, _ *string) error {
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

func (f fakeFileInfo) Mode() fs.FileMode {
	if f.isDevice {
		return f.mode | fs.ModeDevice
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
