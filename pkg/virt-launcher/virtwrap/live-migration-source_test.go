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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var _ = Describe("Live migration source", func() {
	var ctrl *gomock.Controller
	var libvirtDomainManager *LibvirtDomainManager
	var mockConn *cli.MockConnection
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)

		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "test-namespace",
			},
			Status: v1.VirtualMachineInstanceStatus{
				MigrationState: &v1.VirtualMachineInstanceMigrationState{
					MigrationUID: types.UID(fmt.Sprintf("%v", GinkgoRandomSeed())),
				},
			},
		}

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
			nil,
			v1.KvmHypervisorName,
			nil,
			"", false,
			false, // firmware auto-selection
			false,
			nil,
		)
		libvirtDomainManager = manager.(*LibvirtDomainManager)
		libvirtDomainManager.initializeMigrationMetadata(vmi, v1.MigrationPreCopy)
	})

	Context("Migration result", func() {

		It("should only be set once", func() {
			libvirtDomainManager.setMigrationResult(false, "")
			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.EndTimestamp).ToNot(BeNil(), "migration EndTimestamp not set")
			Expect(migrationMetadata.Failed).To(BeFalse(), "migration has failed")

			endTimestamp := migrationMetadata.EndTimestamp.DeepCopy()

			libvirtDomainManager.setMigrationResult(true, "")
			Expect(exists).To(BeTrue())
			Expect(migrationMetadata.EndTimestamp).To(Equal(endTimestamp), "migrationMetadata changed")
			Expect(migrationMetadata.Failed).To(BeFalse(), "migration has failed")
		})

		DescribeTable("EndTimestamp", func(isFailed bool, abortStatus v1.MigrationAbortStatus) {
			if abortStatus != "" {
				libvirtDomainManager.setMigrationAbortStatus(abortStatus)
			}
			libvirtDomainManager.setMigrationResult(isFailed, "")
			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.Failed).To(Equal(isFailed), "migration result is wrong")
			Expect(migrationMetadata.EndTimestamp).ToNot(BeNil(), "migration EndTimestamp not set")
		},
			Entry("should be set when the migration is successful", false, v1.MigrationAbortStatus("")),
			Entry("should be set when the migration has failed", true, v1.MigrationAbortStatus("")),
			Entry("should be set when the migration has been aborted", false, v1.MigrationAbortSucceeded),
			Entry("should be set when an abortion request did not succeed", false, v1.MigrationAbortFailed),
			Entry("should be set when an abortion request is still in progress", false, v1.MigrationAbortInProgress),
		)

		DescribeTable("when an abortion is in progress", func(isFailed bool) {
			libvirtDomainManager.setMigrationAbortStatus(v1.MigrationAbortInProgress)

			libvirtDomainManager.setMigrationResult(isFailed, "")

			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.EndTimestamp).ToNot(BeNil(), "migration EndTimestamp not set")
			Expect(migrationMetadata.Failed).To(Equal(isFailed), "migration result is wrong")
			Expect(migrationMetadata.AbortStatus).To(Equal(string(v1.MigrationAbortInProgress)), "abort status should be unchanged")
		},
			Entry("marking the migration as failed should finalize", true),
			Entry("marking the migration as completed should finalize", false),
		)

		// When the target migration proxy is torn down before source abort completes,
		// libvirt reports "client socket is closed". setMigrationResult does not map
		// that to abortStatus=Succeeded, which is why ConfirmVMIPostMigrationAborted
		// fails on the source VMI in the decentralized delete-target-migration test.
		It("records failed migration with blank AbortStatus when libvirt fails with client socket is closed", func() {
			libvirtDomainManager.setMigrationResult(true, "client socket is closed")

			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.Failed).To(BeTrue())
			Expect(migrationMetadata.FailureReason).To(Equal("client socket is closed"))
			Expect(migrationMetadata.AbortStatus).To(BeEmpty())
		})
	})

	Context("Migration abort status", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
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
				nil,
				v1.KvmHypervisorName,
				nil,
				"", false,
				false, // firmware auto-selection
				false,
				nil,
			)
			libvirtDomainManager = manager.(*LibvirtDomainManager)
			libvirtDomainManager.initializeMigrationMetadata(vmi, v1.MigrationPreCopy)
		})

		DescribeTable("should set abort status", func(abortStatus v1.MigrationAbortStatus) {
			libvirtDomainManager.setMigrationAbortStatus(abortStatus)
			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue(), "migrationMetadata not found")
			Expect(migrationMetadata.AbortStatus).To(Equal(string(abortStatus)))
			Expect(migrationMetadata.EndTimestamp).To(BeNil(), "EndTimestamp should not be set by abort status")
		},
			Entry("to InProgress", v1.MigrationAbortInProgress),
			Entry("to Succeeded", v1.MigrationAbortSucceeded),
			Entry("to Failed", v1.MigrationAbortFailed),
		)

		It("should overwrite previous abort status", func() {
			libvirtDomainManager.setMigrationAbortStatus(v1.MigrationAbortInProgress)
			migrationMetadata, exists := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue())
			Expect(migrationMetadata.AbortStatus).To(Equal(string(v1.MigrationAbortInProgress)))

			libvirtDomainManager.setMigrationAbortStatus(v1.MigrationAbortFailed)
			migrationMetadata, exists = libvirtDomainManager.metadataCache.Migration.Load()
			Expect(exists).To(BeTrue())
			Expect(migrationMetadata.AbortStatus).To(Equal(string(v1.MigrationAbortFailed)))
		})
		It("cancelMigration should no-op when migration has no StartTimestamp", func() {
			libvirtDomainManager.metadataCache.Migration.WithSafeBlock(func(m *api.MigrationMetadata, _ bool) {
				m.StartTimestamp = nil
			})
			original, _ := libvirtDomainManager.metadataCache.Migration.Load()

			libvirtDomainManager.cancelMigration(vmi)

			after, _ := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(after.AbortStatus).To(Equal(original.AbortStatus))
		})

		It("cancelMigration should no-op when migration already has EndTimestamp", func() {
			libvirtDomainManager.metadataCache.Migration.WithSafeBlock(func(m *api.MigrationMetadata, _ bool) {
				m.EndTimestamp = pointer.P(metav1.Now())
			})

			libvirtDomainManager.cancelMigration(vmi)

			after, _ := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(after.AbortStatus).To(Equal(""))
		})

		It("cancelMigration should no-op when abort is already in progress", func() {
			libvirtDomainManager.setMigrationAbortStatus(v1.MigrationAbortInProgress)

			libvirtDomainManager.cancelMigration(vmi)

			after, _ := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(after.AbortStatus).To(Equal(string(v1.MigrationAbortInProgress)))
		})

		It("cancelMigration should no-op when abort already succeeded", func() {
			libvirtDomainManager.setMigrationAbortStatus(v1.MigrationAbortSucceeded)

			libvirtDomainManager.cancelMigration(vmi)

			after, _ := libvirtDomainManager.metadataCache.Migration.Load()
			Expect(after.AbortStatus).To(Equal(string(v1.MigrationAbortSucceeded)))
		})
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

	Context("configureLocalDiskToMigrate", func() {
		const (
			testvol = "test"
			src     = "src"
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
			Entry("filesystem source and filesystem destination", false, false),
			Entry("block source and block destination", true, true),
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
			Entry("with non-nil migration threads and post-copy not allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: pointer.P(uint(3)), AllowPostCopy: false}),
			Entry("with non-nil migration threads and post-copy allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: pointer.P(uint(3)), AllowPostCopy: true}),
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
					options.ParallelMigrationThreads = pointer.P(uint(parallelMigrationThreads))
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
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
	It("should not modify CPU pinning (handled by premigration hook server)", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="4"></vcpupin>
    <vcpupin vcpu="1" cpuset="5"></vcpupin>
  </cputune>
</domain>`
		expectedXML := domXML

		vmi := newVMI("testns", "kubevirt")
		vmi.Spec.Domain.CPU = &v1.CPU{
			Cores:                 2,
			DedicatedCPUPlacement: true,
		}

		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
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
					VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
				},
				DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{
					ClaimName:  destPvcName,
					VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
				},
			},
		}
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi)
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
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
})
