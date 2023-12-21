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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package container_disk

import (
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"time"

	"kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"

	gomock "github.com/golang/mock/gomock"
	mount "github.com/moby/sys/mountinfo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomega_types "github.com/onsi/gomega/types"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

var _ = Describe("ContainerDisk", func() {
	var tmpDir string
	var m *mounter
	var err error
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tmpDir, err = os.MkdirTemp("", "containerdisktest")
		Expect(err).ToNot(HaveOccurred())
		vmi = api.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"

		m = &mounter{
			mountRecords:           make(map[types.UID]*vmiMountTargetRecord),
			mountStateDir:          tmpDir,
			suppressWarningTimeout: 1 * time.Minute,
			socketPathGetter:       containerdisk.NewSocketPathGetter(""),
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	BeforeEach(func() {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: "test",
			VolumeSource: v1.VolumeSource{
				ContainerDisk: testutils.NewFakeContainerDiskSource(),
			},
		})
	})

	Context("checking if containerDisks are ready", func() {
		It("should return false and no error if we are still within the tolerated retry period", func() {
			m.socketPathGetter = func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
				return "", fmt.Errorf("not found")
			}
			ready, err := m.ContainerDisksReady(vmi, time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(ready).To(BeFalse())
		})
		It("should return false and an error if we are outside the tolerated retry period", func() {
			m.socketPathGetter = func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
				return "", fmt.Errorf("not found")
			}
			ready, err := m.ContainerDisksReady(vmi, time.Now().Add(-2*time.Minute))
			Expect(err).To(HaveOccurred())
			Expect(ready).To(BeFalse())
		})
		It("should return true and no error once everything is ready and we are within the tolerated retry period", func() {
			m.socketPathGetter = func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
				return "someting", nil
			}
			ready, err := m.ContainerDisksReady(vmi, time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(ready).To(BeTrue())
		})
		It("should return true and no error once everything is ready when we are outside of the tolerated retry period", func() {
			m.socketPathGetter = func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
				return "someting", nil
			}
			ready, err := m.ContainerDisksReady(vmi, time.Now().Add(-2*time.Minute))
			Expect(err).ToNot(HaveOccurred())
			Expect(ready).To(BeTrue())
		})

		Context("with kernelBoot container", func() {

			BeforeEach(func() {
				vmi.Spec.Volumes = []v1.Volume{}

				vmi.Spec.Domain.Firmware = &v1.Firmware{
					KernelBoot: &v1.KernelBoot{
						Container: &v1.KernelBootContainer{
							KernelPath: "/fake-kernel",
							InitrdPath: "/fake-initrd",
						},
					},
				}
			})

			DescribeTable("should", func(
				pathGetter containerdisk.KernelBootSocketPathGetter,
				addedDelay time.Duration,
				errorMatcher gomega_types.GomegaMatcher,
				shouldBeReady bool,
			) {
				m.kernelBootSocketPathGetter = pathGetter
				ready, err := m.ContainerDisksReady(vmi, time.Now().Add(addedDelay))
				Expect(err).To(errorMatcher)
				Expect(ready).To(Equal(shouldBeReady))
			},
				Entry("return false and no error if we are still within the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "", fmt.Errorf("not found") },
					time.Duration(0),
					Succeed(),
					false,
				),
				Entry("return false and an error if we are outside the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "", fmt.Errorf("not found") },
					time.Duration(-2*time.Minute),
					HaveOccurred(),
					false,
				),
				Entry("return true and no error once everything is ready and we are within the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "someting", nil },
					time.Duration(0),
					Succeed(),
					true,
				),
				Entry("return true and no error once everything is ready when we are outside of the tolerated retry period",
					func(*v1.VirtualMachineInstance) (string, error) { return "someting", nil },
					time.Duration(-2*time.Minute),
					Succeed(),
					true,
				),
			)
		})
	})

	Context("verify mount target recording for vmi", func() {
		It("should set and get same results", func() {

			// verify reading non-existent results just returns empty slice
			record, err := m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(BeNil())

			// verify setting a result works
			record = &vmiMountTargetRecord{
				MountTargetEntries: []vmiMountTargetEntry{
					{
						TargetFile: "sometargetfile",
						SocketFile: "somesocketfile",
					},
				},
			}
			err = m.setMountTargetRecord(vmi, record)
			Expect(err).ToNot(HaveOccurred())

			// verify the file actually exists
			recordFile := filepath.Join(tmpDir, string(vmi.UID))
			exists, err := diskutils.FileExists(recordFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			// verify we can read a result
			record, err = m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record.MountTargetEntries).To(HaveLen(1))
			Expect(record.MountTargetEntries[0].TargetFile).To(Equal("sometargetfile"))
			Expect(record.MountTargetEntries[0].SocketFile).To(Equal("somesocketfile"))

			// verify we can read a result directly from disk if the entry
			// doesn't exist in the map
			delete(m.mountRecords, vmi.UID)
			record, err = m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record.MountTargetEntries).To(HaveLen(1))
			Expect(record.MountTargetEntries[0].TargetFile).To(Equal("sometargetfile"))
			Expect(record.MountTargetEntries[0].SocketFile).To(Equal("somesocketfile"))

			// verify the cache is populated again with the mount info after reading from disk
			_, ok := m.mountRecords[vmi.UID]
			Expect(ok).To(BeTrue())

			// verify delete results
			err = m.deleteMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())

			// verify the file is actually removed
			exists, err = diskutils.FileExists(recordFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

			// verify deleting results that don't exist won't fail
			err = m.deleteMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())

			// verify reading deleted results just returns empty slice
			record, err = m.getMountTargetRecord(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(record).To(BeNil())
		})
	})

	Context("containerdisks checksum", func() {
		var rootMountPoint string

		diskContent := []byte{0x6B, 0x75, 0x62, 0x65, 0x76, 0x69, 0x72, 0x74}

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			mockIsolationDetector := isolation.NewMockPodIsolationDetector(ctrl)
			mockNodeIsolationResult := isolation.NewMockIsolationResult(ctrl)
			mockPodIsolationResult := isolation.NewMockIsolationResult(ctrl)

			m.podIsolationDetector = mockIsolationDetector
			m.nodeIsolationResult = mockNodeIsolationResult

			m.socketPathGetter = func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
				return "somewhere", nil
			}
			m.kernelBootSocketPathGetter = func(vmi *v1.VirtualMachineInstance) (string, error) {
				return "somewhere-kernel", nil
			}

			mockIsolationDetector.EXPECT().DetectForSocket(gomock.Any(), gomock.Any()).Return(mockPodIsolationResult, nil)

			mockPodIsolationResult.EXPECT().Mounts(gomock.Any()).Return([]*mount.Info{&mount.Info{Root: "/", Mountpoint: "/disks"}}, nil)

			rootMountPoint, err = os.MkdirTemp(tmpDir, "root")
			Expect(err).ToNot(HaveOccurred())
			partentToChildMountPoint, err := os.MkdirTemp(rootMountPoint, "child")
			Expect(err).ToNot(HaveOccurred())
			mockNodeIsolationResult.EXPECT().Mounts(gomock.Any()).Return([]*mount.Info{&mount.Info{Root: partentToChildMountPoint}}, nil)

			rootMountPointSafePath, err := safepath.NewPathNoFollow(rootMountPoint)
			Expect(err).ToNot(HaveOccurred())
			mockNodeIsolationResult.EXPECT().MountRoot().Return(rootMountPointSafePath, nil)
		})

		Context("verification", func() {

			type args struct {
				storedChecksum uint32
				diskContent    []byte
				verifyMatcher  gomega_types.GomegaMatcher
			}

			DescribeTable(" should", func(args *args) {
				vmiVolume := vmi.Spec.Volumes[0]

				err := os.WriteFile(filepath.Join(rootMountPoint, vmiVolume.ContainerDisk.Path), args.diskContent, 0660)
				Expect(err).ToNot(HaveOccurred())

				vmi.Status.VolumeStatus = []v1.VolumeStatus{
					v1.VolumeStatus{
						Name:                vmiVolume.Name,
						ContainerDiskVolume: &v1.ContainerDiskInfo{Checksum: args.storedChecksum},
					},
				}

				err = VerifyChecksums(m, vmi)
				Expect(err).To(args.verifyMatcher)

			},
				Entry("succeed if source and target containerdisk match", &args{
					storedChecksum: crc32.ChecksumIEEE(diskContent),
					diskContent:    diskContent,
					verifyMatcher:  Not(HaveOccurred()),
				}),
				Entry("fail if checksum is not present", &args{
					storedChecksum: 0,
					diskContent:    diskContent,
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMissing)),
				}),
				Entry("fail if source and target containerdisk do not match", &args{
					storedChecksum: crc32.ChecksumIEEE([]byte{0xde, 0xad, 0xbe, 0xef}),
					diskContent:    diskContent,
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMismatch)),
				}),
			)
		})

		Context("with custom kernel artifacts", func() {

			type args struct {
				kernel         []byte
				initrd         []byte
				kernelChecksum uint32
				initrdChecksum uint32
				verifyMatcher  gomega_types.GomegaMatcher
			}

			DescribeTable("verification should", func(args *args) {
				kernelBootVMI := api.NewMinimalVMI("fake-vmi")
				kernelBootVMI.Spec.Domain.Firmware = &v1.Firmware{
					KernelBoot: &v1.KernelBoot{
						Container: &v1.KernelBootContainer{},
					},
				}

				if args.kernel != nil {
					kernelFile, err := os.CreateTemp(rootMountPoint, kernelBootVMI.Spec.Domain.Firmware.KernelBoot.Container.KernelPath)
					Expect(err).ToNot(HaveOccurred())
					defer kernelFile.Close()

					_, err = (kernelFile.Write(args.kernel))
					Expect(err).ToNot(HaveOccurred())

					kernelBootVMI.Spec.Domain.Firmware.KernelBoot.Container.KernelPath = filepath.Join("/", filepath.Base(kernelFile.Name()))
				}

				if args.initrd != nil {
					initrdFile, err := os.CreateTemp(rootMountPoint, kernelBootVMI.Spec.Domain.Firmware.KernelBoot.Container.InitrdPath)
					Expect(err).ToNot(HaveOccurred())
					defer initrdFile.Close()

					_, err = (initrdFile.Write(args.initrd))
					Expect(err).ToNot(HaveOccurred())

					kernelBootVMI.Spec.Domain.Firmware.KernelBoot.Container.InitrdPath = filepath.Join("/", filepath.Base(initrdFile.Name()))
				}

				kernelBootVMI.Status.KernelBootStatus = &v1.KernelBootStatus{}
				if args.kernel != nil {
					kernelBootVMI.Status.KernelBootStatus.KernelInfo = &v1.KernelInfo{Checksum: args.kernelChecksum}
				}
				if args.initrd != nil {
					kernelBootVMI.Status.KernelBootStatus.InitrdInfo = &v1.InitrdInfo{Checksum: args.initrdChecksum}
				}

				err = VerifyChecksums(m, kernelBootVMI)
				Expect(err).To(args.verifyMatcher)
			},
				Entry("succeed when source and target custom kernel match", &args{
					kernel:         diskContent,
					kernelChecksum: crc32.ChecksumIEEE(diskContent),
					verifyMatcher:  Not(HaveOccurred()),
				}),
				Entry("succeed when source and target custom initrd match", &args{
					initrd:         diskContent,
					initrdChecksum: crc32.ChecksumIEEE(diskContent),
					verifyMatcher:  Not(HaveOccurred()),
				}),
				Entry("succeed when source and target custom kernel and initrd match", &args{
					kernel:         []byte{0xA, 0xB, 0xC, 0xD},
					kernelChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					initrd:         []byte{0x1, 0x2, 0x3, 0x4},
					initrdChecksum: crc32.ChecksumIEEE([]byte{0x1, 0x2, 0x3, 0x4}),
					verifyMatcher:  Not(HaveOccurred()),
				}),
				Entry("fail when source and target custom kernel do not match", &args{
					kernel:         diskContent,
					kernelChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMismatch)),
				}),
				Entry("fail when source and target custom initrd do not match", &args{
					initrd:         diskContent,
					initrdChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMismatch)),
				}),
				Entry("fail when checksum is missing", &args{
					kernel:        diskContent,
					initrd:        diskContent,
					verifyMatcher: And(HaveOccurred(), MatchError(ErrChecksumMissing)),
				}),
				Entry("fail when source and target custom kernel match but initrd does not", &args{
					kernel:         []byte{0xA, 0xB, 0xC, 0xD},
					kernelChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					initrd:         []byte{0xF, 0xF, 0xE, 0xE},
					initrdChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMismatch)),
				}),
				Entry("fail when source and target custom initrd match but kernel does not", &args{
					kernel:         []byte{0xF, 0xF, 0xE, 0xE},
					kernelChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					initrd:         []byte{0x1, 0x2, 0x3, 0x4},
					initrdChecksum: crc32.ChecksumIEEE([]byte{0x1, 0x2, 0x3, 0x4}),
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMismatch)),
				}),
				Entry("fail when source and target custom initrd and kernel do not match", &args{
					kernel:         []byte{0xF, 0xF, 0xE, 0xE},
					kernelChecksum: crc32.ChecksumIEEE([]byte{0xA, 0xB, 0xC, 0xD}),
					initrd:         []byte{0xA, 0xB, 0xC, 0xD},
					initrdChecksum: crc32.ChecksumIEEE([]byte{0x1, 0x2, 0x3, 0x4}),
					verifyMatcher:  And(HaveOccurred(), MatchError(ErrChecksumMismatch)),
				}),
			)
		})
	})
})
