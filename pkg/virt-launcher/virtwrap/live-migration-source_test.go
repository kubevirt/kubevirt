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

package virtwrap

import (
	"encoding/json"
	"encoding/xml"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var _ = Describe("Live migration source", func() {
	var libvirtDomainManager *LibvirtDomainManager

	Context("Migration result", func() {
		BeforeEach(func() {
			vmi := &v1.VirtualMachineInstance{
				Status: v1.VirtualMachineInstanceStatus{
					MigrationState: &v1.VirtualMachineInstanceMigrationState{
						MigrationUID: types.UID(fmt.Sprintf("%v", GinkgoRandomSeed())),
					},
				},
			}

			mockConn := &cli.MockConnection{}
			testVirtShareDir := fmt.Sprintf("fake-virt-share-%d", GinkgoRandomSeed())
			testEphemeralDiskDir := fmt.Sprintf("fake-ephemeral-disk-%d", GinkgoRandomSeed())
			ephemeralDiskCreatorMock := &fake.MockEphemeralDiskImageCreator{}
			metadataCache := metadata.NewCache()

			manager, _ := NewLibvirtDomainManager(
				mockConn,
				testVirtShareDir,
				testEphemeralDiskDir,
				nil, // agent store
				virtconfig.DefaultARCHOVMFPath,
				ephemeralDiskCreatorMock,
				metadataCache,
				nil, //stop chn
				virtconfig.DefaultDiskVerificationMemoryLimitBytes,
				fakeCpuSetGetter,
				false, // image volume enabled
				false, // libvirt hooks server and client enabled
				nil,
				v1.KvmHypervisorName,
				nil,
			)
			libvirtDomainManager = manager.(*LibvirtDomainManager)
			libvirtDomainManager.initializeMigrationMetadata(vmi, v1.MigrationPreCopy)
		})

		It("should only be set once", func() {
			libvirtDomainManager.setMigrationResult(false, "", "")
			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.EndTimestamp).ToNot(BeNil(), "migration EndTimestamp not set")
			Expect(migrationMetadata.Failed).To(BeFalse(), "migration has failed")

			endTimestamp := migrationMetadata.EndTimestamp.DeepCopy()

			libvirtDomainManager.setMigrationResult(true, "", "")
			Expect(exists).To(BeTrue())
			Expect(migrationMetadata.EndTimestamp).To(Equal(endTimestamp), "migrationMetadata changed")
			Expect(migrationMetadata.Failed).To(BeFalse(), "migration has failed")
		})

		DescribeTable("EndTimestamp", func(isFailed bool, abortStatus v1.MigrationAbortStatus, shouldEndTimestampBeSet bool) {
			libvirtDomainManager.setMigrationResult(isFailed, "", abortStatus)
			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.Failed).To(Equal(isFailed), "migration result is wrong")

			if shouldEndTimestampBeSet {
				Expect(migrationMetadata.EndTimestamp).ToNot(BeNil(), "migration EndTimestamp not set")
			} else {
				Expect(migrationMetadata.EndTimestamp).To(BeNil(), "migration EndTimestamp is set")
			}
		},
			Entry("should be set when the migration is successful", false, v1.MigrationAbortStatus(""), true),
			Entry("should be set when the migration has failed", true, v1.MigrationAbortStatus(""), true),
			Entry("should be set when the migration has been aborted", false, v1.MigrationAbortSucceeded, true),
			Entry("should not be set when an abortion request does not succeed", false, v1.MigrationAbortFailed, false),
			Entry("should not be set when an abortion request is still in progress", false, v1.MigrationAbortInProgress, false),
		)

		DescribeTable("when an abortion is in progress", func(isFailed bool, abortStatus v1.MigrationAbortStatus, shouldEndTimestampBeSet bool, expectedError error) {
			// make it in progress first
			libvirtDomainManager.setMigrationResult(false, "", v1.MigrationAbortInProgress)

			err := libvirtDomainManager.setMigrationResult(isFailed, "", abortStatus)
			if expectedError != nil {
				Expect(err).To(Equal(expectedError))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}

			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")

			if shouldEndTimestampBeSet {
				Expect(migrationMetadata.EndTimestamp).ToNot(BeNil(), "migration EndTimestamp not set")
			} else {
				Expect(migrationMetadata.EndTimestamp).To(BeNil(), "migration EndTimestamp is set")
			}

			if expectedError != nil {
				Expect(migrationMetadata.Failed).To(BeFalse(), "migration has failed")
			} else {
				Expect(migrationMetadata.Failed).To(BeTrue(), "migration has not failed")
			}
		},
			Entry("setting 'Aborting' should return an error", false, v1.MigrationAbortInProgress, false, errors.MigrationAbortInProgressError),
			Entry("setting 'Succeeded' should return an error", true, v1.MigrationAbortSucceeded, true, nil),
			Entry("setting 'Failed' should return an error", true, v1.MigrationAbortFailed, false, nil),
			Entry("marking the migration as failed without an abortion result should return an error", true, v1.MigrationAbortStatus(""), false, errors.MigrationAbortInProgressError),
			Entry("marking the migration as completed without an abortion result should return an error", false, v1.MigrationAbortStatus(""), false, errors.MigrationAbortInProgressError),
		)
	})

	Context("classifyVolumesForMigration", func() {
		It("should classify shared volumes to migrated when they are part of the migrated volumes set", func() {
			const vol = "vol"
			vmi := libvmi.New(
				libvmi.WithHostDiskAndCapacity(vol, "/disk.img", v1.HostDiskExistsOrCreate, "1G", libvmi.WithSharedHostDisk(true)), libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithMigratedVolume(v1.StorageMigratedVolumeInfo{
							VolumeName: vol,
						}),
					),
				))
			Expect(classifyVolumesForMigration(vmi)).To(PointTo(Equal(
				migrationDisks{
					shared:         map[string]bool{},
					generated:      map[string]bool{},
					localToMigrate: map[string]bool{vol: true},
				})))
		})
	})

	Context("getDiskPathFromSource", func() {
		DescribeTable("path resolution",
			func(source *libvirtxml.DomainDiskSource, expectedPath string, expectedErr bool) {
				path, err := getDiskPathFromSource(source)
				if expectedErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(path).To(Equal(expectedPath))
				}
			},
			Entry("resolves block device path",
				&libvirtxml.DomainDiskSource{Block: &libvirtxml.DomainDiskSourceBlock{Dev: "/dev/vda"}},
				"/dev/vda", false),

			Entry("resolves file path",
				&libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: "/test/disk.img"}},
				"/test/disk.img", false),

			Entry("resolves DataStore source path",
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/overlay/path"},
					DataStore: &libvirtxml.DomainDiskDataStore{
						Source: &libvirtxml.DomainDiskSource{
							Block: &libvirtxml.DomainDiskSourceBlock{Dev: "/base/path"},
						},
					},
				},
				"/base/path", false),
			Entry("returns error when DataStore source is nil",
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/overlay/path"},
					DataStore: &libvirtxml.DomainDiskDataStore{
						Source: nil,
					},
				},
				"", true),
			Entry("returns error when DataStore source has no path set",
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/overlay/path"},
					DataStore: &libvirtxml.DomainDiskDataStore{
						Source: &libvirtxml.DomainDiskSource{},
					},
				},
				"", true),

			Entry("returns error for nil source", nil, "", true),

			Entry("returns error when no path is set in source",
				&libvirtxml.DomainDiskSource{}, "", true),
		)
	})

	Context("updateFilePathsToNewDomain", func() {
		targetNS := "target-ns"
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "source-ns",
			},
			Status: v1.VirtualMachineInstanceStatus{
				MigrationState: &v1.VirtualMachineInstanceMigrationState{
					TargetState: &v1.VirtualMachineInstanceMigrationTargetState{
						VirtualMachineInstanceCommonMigrationState: v1.VirtualMachineInstanceCommonMigrationState{
							DomainNamespace: &targetNS,
						},
					},
				},
			},
		}

		DescribeTable("namespace replacement in disk file paths",
			func(source api.DiskSource, expectedFile, expectedDataStoreFile string) {
				domSpec := &api.DomainSpec{
					Devices: api.Devices{
						Disks: []api.Disk{
							{Alias: api.NewUserDefinedAlias("disk0"), Source: source},
						},
					},
				}
				updateFilePathsToNewDomain(vmi, domSpec)
				Expect(domSpec.Devices.Disks[0].Source.File).To(Equal(expectedFile))
				if expectedDataStoreFile != "" {
					Expect(domSpec.Devices.Disks[0].Source.DataStore.Source.File).To(Equal(expectedDataStoreFile))
				}
			},
			Entry("plain file path containing source namespace is rewritten to target namespace",
				api.DiskSource{File: "source-ns/my-vm/disk.img"},
				"target-ns/my-vm/disk.img", ""),

			Entry("plain file path without namespace is left unchanged",
				api.DiskSource{File: "disk.img"},
				"disk.img", ""),

			Entry("overlay disk: overlay qcow2 file containing namespace is rewritten",
				api.DiskSource{
					File: "source-ns/cbt/vol.qcow2",
					DataStore: &api.DataStore{
						Source: &api.DiskSource{File: "/var/run/kubevirt-private/vmi-disks/vol/disk.img"},
					},
				},
				"target-ns/cbt/vol.qcow2",
				"/var/run/kubevirt-private/vmi-disks/vol/disk.img"),

			Entry("overlay disk: DataStore backing file containing namespace is rewritten",
				api.DiskSource{
					File: "/var/lib/libvirt/qemu/cbt/vol.qcow2",
					DataStore: &api.DataStore{
						Source: &api.DiskSource{File: "source-ns/vol/disk.img"},
					},
				},
				"/var/lib/libvirt/qemu/cbt/vol.qcow2",
				"target-ns/vol/disk.img"),

			Entry("overlay disk: both qcow2 and DataStore backing file contain namespace, both rewritten",
				api.DiskSource{
					File: "source-ns/cbt/vol.qcow2",
					DataStore: &api.DataStore{
						Source: &api.DiskSource{File: "source-ns/vol/disk.img"},
					},
				},
				"target-ns/cbt/vol.qcow2",
				"target-ns/vol/disk.img"),

			Entry("overlay disk: DataStore backing file without namespace is left unchanged",
				api.DiskSource{
					File: "/var/lib/libvirt/qemu/cbt/vol.qcow2",
					DataStore: &api.DataStore{
						Source: &api.DiskSource{File: "/var/run/kubevirt-private/vmi-disks/vol/disk.img"},
					},
				},
				"/var/lib/libvirt/qemu/cbt/vol.qcow2",
				"/var/run/kubevirt-private/vmi-disks/vol/disk.img"),
		)
	})

	Context("convertDisks", func() {
		DescribeTable("syncing spec file paths into migratable libvirt XML",
			func(specSource api.DiskSource, libvirtSource *libvirtxml.DomainDiskSource,
				expectedFile, expectedDataStoreFile string) {
				domSpec := &api.DomainSpec{
					Devices: api.Devices{
						Disks: []api.Disk{
							{Alias: api.NewUserDefinedAlias("disk0"), Source: specSource},
						},
					},
				}
				domcfg := &libvirtxml.Domain{
					Devices: &libvirtxml.DomainDeviceList{
						Disks: []libvirtxml.DomainDisk{
							{Alias: &libvirtxml.DomainAlias{Name: "ua-disk0"}, Source: libvirtSource},
						},
					},
				}
				Expect(convertDisks(domSpec, domcfg)).To(Succeed())
				Expect(domcfg.Devices.Disks[0].Source.File.File).To(Equal(expectedFile))
				if expectedDataStoreFile != "" {
					Expect(domcfg.Devices.Disks[0].Source.DataStore.Source.File.File).To(Equal(expectedDataStoreFile))
				}
			},
			Entry("plain file disk: spec path overwrites stale libvirt XML path",
				api.DiskSource{File: "/var/run/kubevirt-private/vmi-disks/vol/disk.img"},
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/var/run/kubevirt-private/vmi-disks/vol/old-disk.img"},
				},
				"/var/run/kubevirt-private/vmi-disks/vol/disk.img", ""),

			Entry("CBT overlay disk with file backend: both qcow2 and backing store overwrite stale libvirt XML",
				api.DiskSource{
					File: "/var/lib/libvirt/qemu/cbt/vol.qcow2",
					DataStore: &api.DataStore{
						Source: &api.DiskSource{File: "/var/run/kubevirt-private/vmi-disks/vol/disk.img"},
					},
				},
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/qemu/cbt/old-vol.qcow2"},
					DataStore: &libvirtxml.DomainDiskDataStore{
						Source: &libvirtxml.DomainDiskSource{
							File: &libvirtxml.DomainDiskSourceFile{File: "/var/run/kubevirt-private/vmi-disks/vol/old-disk.img"},
						},
					},
				},
				"/var/lib/libvirt/qemu/cbt/vol.qcow2",
				"/var/run/kubevirt-private/vmi-disks/vol/disk.img"),

			Entry("CBT overlay disk with block backend: only qcow2 overwritten, block backend unchanged",
				api.DiskSource{
					File: "/var/lib/libvirt/qemu/cbt/vol.qcow2",
					DataStore: &api.DataStore{
						Source: &api.DiskSource{Dev: "/dev/vol"},
					},
				},
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/qemu/cbt/old-vol.qcow2"},
					DataStore: &libvirtxml.DomainDiskDataStore{
						Source: &libvirtxml.DomainDiskSource{
							Block: &libvirtxml.DomainDiskSourceBlock{Dev: "/dev/vol"},
						},
					},
				},
				"/var/lib/libvirt/qemu/cbt/vol.qcow2", ""),
		)
	})

	Context("configureLocalDiskToMigrate", func() {
		const (
			testvol = "test"
			src     = "src"
			dst     = "dst"
		)

		fsMode := k8sv1.PersistentVolumeFilesystem
		blockMode := k8sv1.PersistentVolumeBlock
		infoFs := v1.PersistentVolumeClaimInfo{
			ClaimName:  src,
			VolumeMode: &fsMode,
		}
		infoBlock := v1.PersistentVolumeClaimInfo{
			ClaimName:  src,
			VolumeMode: &blockMode,
		}

		createDomWithFsImage := func(name string) *libvirtxml.Domain {
			return &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Disks: []libvirtxml.DomainDisk{
						{
							Source: &libvirtxml.DomainDiskSource{
								File: &libvirtxml.DomainDiskSourceFile{
									File: getFsImagePath(name),
								},
							},
							Alias: &libvirtxml.DomainAlias{
								Name: fmt.Sprintf("ua-%s", name),
							},
						},
					},
				},
			}
		}
		createDomWithBlock := func(name string) *libvirtxml.Domain {
			return &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Disks: []libvirtxml.DomainDisk{
						{
							Source: &libvirtxml.DomainDiskSource{
								Block: &libvirtxml.DomainDiskSourceBlock{
									Dev: getBlockPath(name),
								},
							},
							Alias: &libvirtxml.DomainAlias{
								Name: fmt.Sprintf("ua-%s", name),
							},
						},
					},
				},
			}
		}
		volPVC := v1.Volume{
			Name: testvol,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: src,
					},
				},
			},
		}
		volDV := v1.Volume{
			Name: testvol,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: src,
				},
			},
		}
		volHostDisk := v1.Volume{
			Name: testvol,
			VolumeSource: v1.VolumeSource{
				HostDisk: &v1.HostDisk{
					Path: getFsImagePath(testvol),
				},
			},
		}

		DescribeTable("replace filesystem and block migrated volumes", func(isSrcBlock, isDstBlock bool, vol v1.Volume) {
			retDiskSize := func(disk *libvirtxml.DomainDisk) (int64, error) {
				return 2028994560, nil
			}
			getDiskVirtualSizeFunc = retDiskSize
			var dom *libvirtxml.Domain
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{vol},
				},
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName: testvol,
						},
					},
					VolumeStatus: []v1.VolumeStatus{
						{
							Name: testvol,
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName: src,
							},
						},
					},
				},
			}
			if isSrcBlock {
				vmi.Status.MigratedVolumes[0].SourcePVCInfo = &infoBlock
				dom = createDomWithBlock(testvol)
			} else {
				vmi.Status.MigratedVolumes[0].SourcePVCInfo = &infoFs
				dom = createDomWithFsImage(testvol)
			}
			if isDstBlock {
				vmi.Status.MigratedVolumes[0].DestinationPVCInfo = &infoBlock
			} else {
				vmi.Status.MigratedVolumes[0].DestinationPVCInfo = &infoFs
			}

			err := configureLocalDiskToMigrate(dom, vmi)
			Expect(err).ToNot(HaveOccurred())

			if isDstBlock {
				Expect(dom.Devices.Disks[0].Source.File).To(BeNil())
				Expect(dom.Devices.Disks[0].Source.Block).NotTo(BeNil())
				Expect(dom.Devices.Disks[0].Source.Block.Dev).To(Equal(getBlockPath(testvol)))

			} else {
				Expect(dom.Devices.Disks[0].Source.Block).To(BeNil())
				Expect(dom.Devices.Disks[0].Source.File).NotTo(BeNil())
				Expect(dom.Devices.Disks[0].Source.File.File).To(Equal(getFsImagePath(testvol)))
			}
		},
			Entry("filesystem source and destination", false, false, volPVC),
			Entry("filesystem source and block destination", false, true, volPVC),
			Entry("block source and filesystem destination", true, false, volPVC),
			Entry("block source and destination", true, true, volPVC),
			Entry("filesystem source and block destination with DV", false, true, volDV),
			Entry("block source and filesystem destination with DV", true, false, volDV),
			Entry("filesystem source and block destination with hostdisks", false, true, volHostDisk),
			Entry("block source and filesystem destination with hostdisks", true, false, volHostDisk),
		)

		DescribeTable("replace filesystem and block migrated volumes with CBT overlay", func(isSrcBlock, isDstBlock bool) {
			retDiskSize := func(disk *libvirtxml.DomainDisk) (int64, error) {
				return 2028994560, nil
			}
			getDiskVirtualSizeFunc = retDiskSize

			cbtOverlayPath := "/var/lib/libvirt/qemu/cbt/" + testvol + ".qcow2"
			var backendSrc *libvirtxml.DomainDiskSource
			if isSrcBlock {
				backendSrc = &libvirtxml.DomainDiskSource{Block: &libvirtxml.DomainDiskSourceBlock{Dev: getBlockPath(testvol)}}
			} else {
				backendSrc = &libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: getFsImagePath(testvol)}}
			}
			dom := &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Disks: []libvirtxml.DomainDisk{
						{
							Source: &libvirtxml.DomainDiskSource{
								File:      &libvirtxml.DomainDiskSourceFile{File: cbtOverlayPath},
								DataStore: &libvirtxml.DomainDiskDataStore{Source: backendSrc},
							},
							Alias: &libvirtxml.DomainAlias{Name: fmt.Sprintf("ua-%s", testvol)},
						},
					},
				},
			}
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{volPVC},
				},
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName: testvol,
						},
					},
					VolumeStatus: []v1.VolumeStatus{
						{
							Name:                      testvol,
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{ClaimName: src},
						},
					},
				},
			}
			if isSrcBlock {
				vmi.Status.MigratedVolumes[0].SourcePVCInfo = &infoBlock
			} else {
				vmi.Status.MigratedVolumes[0].SourcePVCInfo = &infoFs
			}
			if isDstBlock {
				vmi.Status.MigratedVolumes[0].DestinationPVCInfo = &infoBlock
			} else {
				vmi.Status.MigratedVolumes[0].DestinationPVCInfo = &infoFs
			}

			err := configureLocalDiskToMigrate(dom, vmi)
			Expect(err).ToNot(HaveOccurred())

			diskSrc := dom.Devices.Disks[0].Source
			Expect(diskSrc.File).NotTo(BeNil())
			Expect(diskSrc.File.File).To(Equal(cbtOverlayPath))
			Expect(diskSrc.DataStore).NotTo(BeNil())
			Expect(diskSrc.DataStore.Source).NotTo(BeNil())
			if isDstBlock {
				Expect(diskSrc.DataStore.Source.File).To(BeNil())
				Expect(diskSrc.DataStore.Source.Block).NotTo(BeNil())
				Expect(diskSrc.DataStore.Source.Block.Dev).To(Equal(getBlockPath(testvol)))
			} else {
				Expect(diskSrc.DataStore.Source.Block).To(BeNil())
				Expect(diskSrc.DataStore.Source.File).NotTo(BeNil())
				Expect(diskSrc.DataStore.Source.File.File).To(Equal(getFsImagePath(testvol)))
			}
		},
			Entry("filesystem source and block destination", false, true),
			Entry("block source and filesystem destination", true, false),
		)
	})

	Context("shouldConfigureParallelMigration", func() {
		DescribeTable("should not configure parallel migration", func(options *cmdclient.MigrationOptions) {
			shouldConfigure, _ := shouldConfigureParallelMigration(options)
			Expect(shouldConfigure).To(BeFalse())
		},
			Entry("with nil options", nil),
			Entry("with nil migration threads", &cmdclient.MigrationOptions{ParallelMigrationThreads: nil}),
			Entry("with nil migration threads and post-copy allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: nil, AllowPostCopy: true}),
		)

		DescribeTable("should configure parallel migration", func(options *cmdclient.MigrationOptions) {
			shouldConfigure, _ := shouldConfigureParallelMigration(options)
			Expect(shouldConfigure).To(BeTrue())
		},
			Entry("with non-nil migration threads and post-copy not allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: virtpointer.P(uint(3)), AllowPostCopy: false}),
			Entry("with non-nil migration threads and post-copy allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: virtpointer.P(uint(3)), AllowPostCopy: true}),
		)
	})

	Context("getDiskTargetsForMigration", func() {
		var ctrl *gomock.Controller
		var mockLibvirt *testing.Libvirt
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockLibvirt = testing.NewLibvirt(ctrl)
		})
		It("should correctly collect a list of disks for migration", func() {
			_true := true
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
					},
				},
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
				{
					Name: "myvolumehost",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_true,
						},
					},
				},
			}
			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)

			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(embedMigrationDomain, nil)

			copyDisks := getDiskTargetsForMigration(mockLibvirt.VirtDomain, vmi)
			Expect(copyDisks).Should(ConsistOf("vdb", "vdd"))
		})
	})

	Context("generateMigrationFlags", func() {
		DescribeTable("check migration flags",
			func(migrationType string) {
				isBlockMigration := migrationType == "block"
				isVmiPaused := migrationType == "paused"

				options := &cmdclient.MigrationOptions{
					UnsafeMigration:   migrationType == "unsafe",
					AllowAutoConverge: migrationType == "autoConverge",
					AllowPostCopy:     migrationType == "postCopy",
				}

				shouldConfigureParallel, parallelMigrationThreads := shouldConfigureParallelMigration(options)
				if shouldConfigureParallel {
					options.ParallelMigrationThreads = virtpointer.P(uint(parallelMigrationThreads))
				}

				flags := generateMigrationFlags(isBlockMigration, isVmiPaused, options)
				expectedMigrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

				if isBlockMigration {
					expectedMigrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
				} else if migrationType == "unsafe" {
					expectedMigrateFlags |= libvirt.MIGRATE_UNSAFE
				}
				if options.AllowAutoConverge {
					expectedMigrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
				}
				if migrationType == "postCopy" {
					expectedMigrateFlags |= libvirt.MIGRATE_POSTCOPY
				}
				if migrationType == "paused" {
					expectedMigrateFlags |= libvirt.MIGRATE_PAUSED
				}
				if shouldConfigureParallel {
					expectedMigrateFlags |= libvirt.MIGRATE_PARALLEL
				}
				Expect(flags).To(Equal(expectedMigrateFlags), "libvirt migration flags are not set as expected")
			},
			Entry("with block migration", "block"),
			Entry("without block migration", "live"),
			Entry("unsafe migration", "unsafe"),
			Entry("migration auto converge", "autoConverge"),
			Entry("migration using postcopy", "postCopy"),
			Entry("migration of paused vmi", "paused"),
		)
	})
})

var _ = Describe("migratableDomXML", func() {
	var ctrl *gomock.Controller
	var mockLibvirt *testing.Libvirt
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)
	})
	It("should parse the XML with the metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
    </kubevirt>
   </metadata>
</domain>`
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
    </kubevirt>
   </metadata>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
	It("should change CPU pinning according to migration metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="4"></vcpupin>
    <vcpupin vcpu="1" cpuset="5"></vcpupin>
  </cputune>
</domain>`
		// migratableDomXML() removes the migration block but not its ident, which is its own token, hence the blank line below
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="6"></vcpupin>
    <vcpupin vcpu="1" cpuset="7"></vcpupin>
  </cputune>
  <cpu>
    <topology sockets="1" cores="2" threads="1"></topology>
  </cpu>
</domain>`

		By("creating a VMI with dedicated CPU cores")
		vmi := newVMI("testns", "kubevirt")
		vmi.Spec.Domain.CPU = &v1.CPU{
			Cores:                 2,
			DedicatedCPUPlacement: true,
		}

		By("making up a target topology")
		topology := &cmdv1.Topology{NumaCells: []*cmdv1.Cell{{
			Id: 0,
			Cpus: []*cmdv1.CPU{
				{
					Id:       6,
					Siblings: []uint32{6},
				},
				{
					Id:       7,
					Siblings: []uint32{7},
				},
			},
		}}}
		targetNodeTopology, err := json.Marshal(topology)
		Expect(err).NotTo(HaveOccurred(), "failed to marshall the topology")

		By("saving that topology in the migration state of the VMI")
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetCPUSet:       []int{6, 7},
			TargetNodeTopology: string(targetNodeTopology),
		}

		By("generated the domain XML for a migration to that target")
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		Expect(domSpec.VCPU).NotTo(BeNil())
		Expect(domSpec.CPUTune).NotTo(BeNil())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred(), "failed to generate target domain XML")

		By("ensuring the generated XML is accurate")
		Expect(newXML).To(Equal(expectedXML), "the target XML is not as expected")
	})
	DescribeTable("slices section", func(domXML string) {
		retDiskSize := func(disk *libvirtxml.DomainDisk) (int64, error) {
			return 2028994560, nil
		}
		getDiskVirtualSizeFunc = retDiskSize
		const (
			volName       = "datavolumedisk1"
			sourcePvcName = "src-pvc"
			destPvcName   = "dst-pvc"
		)
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type="file" device="disk" model="virtio-non-transitional">
      <driver name="qemu" type="raw" cache="none" error_policy="stop" discard="unmap"></driver>
      <source file="/var/run/kubevirt-private/vmi-disks/datavolumedisk1/disk.img" index="1">
        <slices>
          <slice type="storage" offset="0" size="2028994560"></slice>
        </slices>
      </source>
      <backingStore></backingStore>
      <target dev="vda" bus="virtio"></target>
      <alias name="ua-datavolumedisk1"></alias>
      <address type="pci" domain="0x0000" bus="0x07" slot="0x00" function="0x0"></address>
    </disk>
  </devices>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		vmi.Spec.Volumes = append(vmi.Spec.Volumes,
			v1.Volume{
				Name: volName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: sourcePvcName,
					},
				},
			})
		vmi.Status.MigratedVolumes = []v1.StorageMigratedVolumeInfo{
			{
				VolumeName: volName,
				SourcePVCInfo: &v1.PersistentVolumeClaimInfo{
					ClaimName:  sourcePvcName,
					VolumeMode: virtpointer.P(k8sv1.PersistentVolumeFilesystem),
				},
				DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{
					ClaimName:  destPvcName,
					VolumeMode: virtpointer.P(k8sv1.PersistentVolumeFilesystem),
				},
			},
		}
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	},
		Entry("add slices section", `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-private/vmi-disks/datavolumedisk1/disk.img' index='1'/>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <alias name='ua-datavolumedisk1'/>
      <address type='pci' domain='0x0000' bus='0x07' slot='0x00' function='0x0'/>
    </disk>
  </devices>
</domain>`),
		Entry("slices section already set", `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-private/vmi-disks/datavolumedisk1/disk.img' index='1'>
        <slices>
          <slice type='storage' offset='0' size='2028994560'></slice>
        </slices>
      </source>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <alias name='ua-datavolumedisk1'/>
      <address type='pci' domain='0x0000' bus='0x07' slot='0x00' function='0x0'/>
    </disk>
  </devices>
</domain>`),
	)
	It("should generate correct xml for user data for copied disks during the migration", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-ephemeral-disks/cloud-init-data/default/vm-dv/noCloud.iso' index='1'/>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <alias name='ua-cloudinitdisk'/>
      <address type='pci' domain='0x0000' bus='0x07' slot='0x00' function='0x0'/>
    </disk>
  </devices>
</domain>`
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type="file" device="disk" model="virtio-non-transitional">
      <driver name="qemu" type="raw" cache="none" error_policy="stop" discard="unmap"></driver>
      <source file="/var/run/kubevirt-ephemeral-disks/cloud-init-data/default/vm-dv/noCloud.iso" index="1"></source>
      <backingStore></backingStore>
      <target dev="vda" bus="virtio"></target>
      <alias name="ua-cloudinitdisk"></alias>
      <address type="pci" domain="0x0000" bus="0x07" slot="0x00" function="0x0"></address>
    </disk>
  </devices>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		userData := "fake\nuser\ndata\n"
		networkData := "FakeNetwork"
		addCloudInitDisk(vmi, userData, networkData)
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
})
