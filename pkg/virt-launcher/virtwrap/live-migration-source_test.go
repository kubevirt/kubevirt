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
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	utilheap "kubevirt.io/kubevirt/pkg/util/heap"
	migrationutils "kubevirt.io/kubevirt/pkg/util/migrations"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

func parseDefaultStallDetectorFactor(value string) float64 {
	factor, err := virtconfig.ParseFactor(value, virtconfig.StallDetectorFactorPrecision)
	Expect(err).NotTo(HaveOccurred())
	return factor
}

var _ = Describe("Live migration source", func() {
	var ctrl *gomock.Controller
	var libvirtDomainManager *LibvirtDomainManager
	var mockConn *cli.MockConnection
	var testDomainName string
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
		testDomainName = api.VMINamespaceKeyFunc(vmi)

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
				false, // libvirt hooks server and client enabled
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
				expectedFile, expectedDataStoreFile, expectedDataStoreBlockDev string) {
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
				if expectedDataStoreBlockDev != "" {
					Expect(domcfg.Devices.Disks[0].Source.DataStore.Source.Block).NotTo(BeNil())
					Expect(domcfg.Devices.Disks[0].Source.DataStore.Source.Block.Dev).To(Equal(expectedDataStoreBlockDev))
					Expect(domcfg.Devices.Disks[0].Source.DataStore.Source.File).To(BeNil())
				}
			},
			Entry("plain file disk: spec path overwrites stale libvirt XML path",
				api.DiskSource{File: "/var/run/kubevirt-private/vmi-disks/vol/disk.img"},
				&libvirtxml.DomainDiskSource{
					File: &libvirtxml.DomainDiskSourceFile{File: "/var/run/kubevirt-private/vmi-disks/vol/old-disk.img"},
				},
				"/var/run/kubevirt-private/vmi-disks/vol/disk.img", "", ""),

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
				"/var/run/kubevirt-private/vmi-disks/vol/disk.img", ""),

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
				"/var/lib/libvirt/qemu/cbt/vol.qcow2", "", "/dev/vol"),
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

	Context("Migration monitor stall detector", func() {
		const (
			testCompletionTimeSec int64  = 300
			testSwitchoverTimeout uint64 = 60
			testMaxDowntimeMs     uint64 = 900
		)

		var monitor *migrationMonitor
		var sd *stallDetector

		// gets the timestamp needed to exceed the timeout
		pastTimeoutNs := func() int64 {
			return (testCompletionTimeSec + 1) * int64(time.Second)
		}

		setupSuccessfulAbortContext := func(mockDomain *cli.MockVirDomain) {
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_UNBOUNDED}, nil)
			mockDomain.EXPECT().AbortJob().Return(nil)
			mockDomain.EXPECT().Free()
		}

		expectMigrationAbortSucceeded := func() {
			Eventually(func() string {
				migration, _ := libvirtDomainManager.metadataCache.Migration.Load()
				return migration.AbortStatus
			}).WithTimeout(500 * time.Millisecond).Should(Equal(string(v1.MigrationAbortSucceeded)))
		}

		BeforeEach(func() {
			options := &cmdclient.MigrationOptions{
				StallDetectorOptions: &cmdclient.StallDetectorOptions{
					StallMargin:               float64(4) / 100,
					StallProgressTimeout:      25,
					SwitchoverTimeout:         testSwitchoverTimeout,
					EwmaAlpha:                 parseDefaultStallDetectorFactor("0.4"),
					PrecopyPossibleFactor:     parseDefaultStallDetectorFactor("1.5"),
					PatienceWindowDecayFactor: parseDefaultStallDetectorFactor("0.5"),
					SearchLocalMinima:         true,
					CompletionTimeoutFactor:   parseDefaultStallDetectorFactor("2.0"),
				},
				MaxDowntimeMs: testMaxDowntimeMs,
			}
			sd = &stallDetector{
				stallDetectorOptions: *options.StallDetectorOptions,
				maxDowntimeMs:        options.MaxDowntimeMs,
			}
			monitor = &migrationMonitor{
				l:                        libvirtDomainManager,
				vmi:                      vmi,
				options:                  options,
				start:                    time.Now().UTC().UnixNano(),
				acceptableCompletionTime: testCompletionTimeSec,
				switchOverDeadline:       testCompletionTimeSec,
				stallDetectionEnabled:    true,
				stallDetector:            sd,
				logger:                   log.Log.Object(vmi),
			}
		})

		Describe("processInflightMigration", func() {
			var ctrl *gomock.Controller
			var mockDomain *cli.MockVirDomain

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				mockDomain = cli.NewMockVirDomain(ctrl)
			})

			It("should not panic when job stats are unavailable", func() {
				monitor.stallDetector.initialMaxDowntimeSet = true

				mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

				Expect(func() {
					monitor.processInflightMigration(mockDomain, nil, false, monitor.logger)
				}).NotTo(Panic())
			})

			It("should set initial max downtime to MaxDowntimeMs when it is lower than 300", func() {
				monitor.options.MaxDowntimeMs = 150
				monitor.stallDetector.initialMaxDowntimeSet = false

				mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(150), uint32(0)).Times(1).Return(nil)

				stats := &libvirt.DomainJobInfo{}
				monitor.processInflightMigration(mockDomain, stats, false, monitor.logger)
				Expect(monitor.stallDetector.initialMaxDowntimeSet).To(BeTrue())
			})

			It("should set initial max downtime to 300 when MaxDowntimeMs is higher than 300", func() {
				monitor.options.MaxDowntimeMs = 5000
				monitor.stallDetector.initialMaxDowntimeSet = false

				mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(300), uint32(0)).Times(1).Return(nil)

				stats := &libvirt.DomainJobInfo{}
				monitor.processInflightMigration(mockDomain, stats, false, monitor.logger)
				Expect(monitor.stallDetector.initialMaxDowntimeSet).To(BeTrue())
			})

			It("should invoke stall detection and trigger convergence action when stats indicate a stall at an iteration boundary", func() {
				monitor.options.MaxDowntimeMs = testMaxDowntimeMs
				sd.allowWorkloadDisruption = true
				sd.initialMaxDowntimeSet = true
				sd.ewmaBandwidthBps = 1000
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 1000}

				mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(migrationutils.QEMUMaxMigrationDowntimeMS), uint32(0)).Times(1).Return(nil)

				stats := &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemainingSet: true,
					DataRemaining:    1000,
					TimeElapsedSet:   true,
					TimeElapsed:      30_000,
					MemIterationSet:  true,
					MemIteration:     5,
				}
				monitor.processInflightMigration(mockDomain, stats, true, monitor.logger)
				Expect(sd.stallDetected).To(BeTrue())
				Expect(sd.switchoverInitiated).To(BeTrue())
			})

		})

		Describe("shouldTriggerTimeout", func() {
			It("when migration is paused, should use acceptableCompletionTime to calculate whether we timed out", func() {
				monitor.l.updateVMIMigrationMode(v1.MigrationPaused)

				Expect(monitor.shouldTriggerTimeout(testCompletionTimeSec*int64(time.Second), monitor.logger)).To(BeFalse())
				Expect(monitor.shouldTriggerTimeout(pastTimeoutNs(), monitor.logger)).To(BeTrue())
			})

			It("else use switchOverDeadline", func() {
				monitor.l.updateVMIMigrationMode(v1.MigrationPreCopy)
				monitor.switchOverDeadline = 200

				Expect(monitor.shouldTriggerTimeout(200*int64(time.Second), monitor.logger)).To(BeFalse())
				Expect(monitor.shouldTriggerTimeout(201*int64(time.Second), monitor.logger)).To(BeTrue())
			})
		})

		Describe("shouldAssistMigrationToComplete", func() {
			It("should always return false when stall detection is enabled", func() {
				monitor.options.AllowWorkloadDisruption = true
				monitor.l.updateVMIMigrationMode(v1.MigrationPreCopy)

				Expect(monitor.shouldAssistMigrationToComplete(pastTimeoutNs(), monitor.logger)).To(BeFalse())

				monitor.stallDetectionEnabled = false
				Expect(monitor.shouldAssistMigrationToComplete(pastTimeoutNs(), monitor.logger)).To(BeTrue())
			})
		})

		Describe("updateBandwidthEstimate", func() {
			It("EWMA calculation is correct", func() {
				alpha := sd.stallDetectorOptions.EwmaAlpha

				sd.updateBandwidthEstimate(1000, monitor.logger)
				Expect(sd.ewmaBandwidthBps).To(Equal(float64(1000)))

				sd.updateBandwidthEstimate(2000, monitor.logger)
				Expect(sd.ewmaBandwidthBps).To(Equal(alpha*2000 + (1-alpha)*1000))

				sd.updateBandwidthEstimate(500, monitor.logger)
				Expect(sd.ewmaBandwidthBps).To(Equal(alpha*500 + (1-alpha)*(alpha*2000+(1-alpha)*1000)))
			})

			It("should use configured EwmaAlpha from StallDetectorOptions", func() {
				sd.stallDetectorOptions.EwmaAlpha = 0.2
				sd.updateBandwidthEstimate(1000, monitor.logger)
				sd.updateBandwidthEstimate(2000, monitor.logger)
				Expect(sd.ewmaBandwidthBps).To(Equal(1200.0))
			})
		})

		Describe("updateCandidates", func() {
			It("should skip candidates larger than minRecordOutsideWindow", func() {
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 100}
				sd.updateCandidates(iterationRecord{elapsedMs: 10_000, remainingBytes: 110}, monitor.logger)
				Expect(sd.minCandidates).To(BeEmpty())

				sd.updateCandidates(iterationRecord{elapsedMs: 10_000, remainingBytes: 95}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(1))
				Expect(sd.minCandidates[0].remainingBytes).To(Equal(uint64(95)))
			})

			It("should update candidates correctly and skip out of window min", func() {
				sd.updateCandidates(iterationRecord{elapsedMs: 0, remainingBytes: 2048}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(1))
				Expect(sd.minCandidates[0].remainingBytes).To(Equal(uint64(2048)))
				Expect(sd.minRecordOutsideWindow).To(Equal(iterationRecord{}))

				sd.updateCandidates(iterationRecord{elapsedMs: 100_000, remainingBytes: 413}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(1))
				Expect(sd.minCandidates[0].remainingBytes).To(Equal(uint64(413)))
				Expect(sd.minRecordOutsideWindow).NotTo(Equal(iterationRecord{}))
				Expect(sd.minRecordOutsideWindow.remainingBytes).To(Equal(uint64(2048)))
			})

			It("should skip candidates preceded by a smaller value", func() {
				sd.updateCandidates(iterationRecord{elapsedMs: 0, remainingBytes: 100}, monitor.logger)
				sd.updateCandidates(iterationRecord{elapsedMs: 1000, remainingBytes: 150}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(1))
				Expect(sd.minCandidates[0].remainingBytes).To(Equal(uint64(100)))
			})

			It("should age out candidates when they are too old", func() {
				// StallProgressTimeout is 25s, so progressTimeoutMs is 25000

				sd.updateCandidates(iterationRecord{elapsedMs: 0, remainingBytes: 500}, monitor.logger)
				sd.updateCandidates(iterationRecord{elapsedMs: 5000, remainingBytes: 400}, monitor.logger)
				sd.updateCandidates(iterationRecord{elapsedMs: 10000, remainingBytes: 300}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(3))
				Expect(sd.minRecordOutsideWindow).To(Equal(iterationRecord{}))

				// Advance time so the first two candidates (t=0, t=5000) age out.
				// The smallest aged-out value (400) becomes minRecordOutsideWindow.
				sd.updateCandidates(iterationRecord{elapsedMs: 30000, remainingBytes: 200}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(2))
				Expect(sd.minCandidates[0].remainingBytes).To(Equal(uint64(300)))
				Expect(sd.minCandidates[1].remainingBytes).To(Equal(uint64(200)))
				Expect(sd.minRecordOutsideWindow).NotTo(Equal(iterationRecord{}))
				Expect(sd.minRecordOutsideWindow.remainingBytes).To(Equal(uint64(400)))

				// Age out all remaining candidates. minRecordOutsideWindow
				// should update to the new global minimum (200).
				sd.updateCandidates(iterationRecord{elapsedMs: 60000, remainingBytes: 100}, monitor.logger)
				Expect(sd.minCandidates).To(HaveLen(1))
				Expect(sd.minCandidates[0].remainingBytes).To(Equal(uint64(100)))
				Expect(sd.minRecordOutsideWindow.remainingBytes).To(Equal(uint64(200)))
			})
		})

		Describe("checkStallCondition", func() {
			It("should return false when minRecordOutsideWindow isn't set", func() {
				Expect(sd.checkStallCondition(100, monitor.logger)).To(BeFalse())
			})

			It("should return false when not stalled", func() {
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 1000}
				// threshold is 1000 * 0.96 = 960
				Expect(sd.checkStallCondition(900, monitor.logger)).To(BeFalse())
			})

			It("should return true when stalled", func() {
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 1000}
				Expect(sd.checkStallCondition(970, monitor.logger)).To(BeTrue())
			})

			It("should use configured StallMargin from StallDetectorOptions", func() {
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 1000}
				sd.stallDetectorOptions.StallMargin = 0.10
				Expect(sd.checkStallCondition(955, monitor.logger)).To(BeTrue())
				sd.stallDetectorOptions.StallMargin = float64(4) / 100
				Expect(sd.checkStallCondition(955, monitor.logger)).To(BeFalse())
			})
		})

		Describe("findBestRemainingBytes", func() {
			It("should find the min record", func() {
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 500}

				// minRecordOutsideWindow is smallest
				sd.minCandidates = []iterationRecord{
					{remainingBytes: 600},
					{remainingBytes: 700},
				}
				Expect(sd.findBestRemainingBytes(monitor.logger)).To(Equal(uint64(500)))

				// candidate is smallest
				sd.minCandidates = []iterationRecord{
					{remainingBytes: 600},
					{remainingBytes: 400},
				}
				Expect(sd.findBestRemainingBytes(monitor.logger)).To(Equal(uint64(400)))

				// no candidates
				sd.minCandidates = []iterationRecord{}
				Expect(sd.findBestRemainingBytes(monitor.logger)).To(Equal(uint64(500)))
			})
		})

		Describe("relaxBestRemainingBytes", func() {
			It("should progressively relax bestRemainingBytes to next smallest observed value", func() {
				// Simulate stall detected at t=0 with a pre-stall best of 100.
				sd.bestRemainingBytes = 100
				sd.initializeRelaxationState(iterationRecord{elapsedMs: 0}, monitor.logger)
				// relaxationPatienceMs = 25*1000 = 25000, deadline = 25000

				// Push post-stall observations (all larger than the pre-stall best).
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 0, remainingBytes: 300}, monitor.logger)
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 0, remainingBytes: 500}, monitor.logger)
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 0, remainingBytes: 200}, monitor.logger)
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 0, remainingBytes: 600}, monitor.logger)

				// Before the deadline: no relaxation.
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 10000, remainingBytes: 9999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(100)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(25000)))

				// At the deadline: pop the smallest (200).
				// patience = 25000/2 = 12500, new deadline = 25000+12500 = 37500
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 25000, remainingBytes: 9999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(200)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(12500)))

				// At the next deadline: pop 300.
				// patience = 12500/2 = 6250, new deadline = 37500+6250 = 43750
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 37500, remainingBytes: 9999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(300)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(6250)))

				// At the next deadline: pop 500.
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 43750, remainingBytes: 9999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(500)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(3125)))

				// At the next deadline: pop 600.
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 50000, remainingBytes: 9999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(600)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(1562)))
			})

			It("should use zero relaxation patience when StallProgressTimeout is zero", func() {
				sd.stallDetectorOptions.StallProgressTimeout = 0
				sd.initializeRelaxationState(iterationRecord{elapsedMs: 5000}, monitor.logger)
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(0)))
				Expect(sd.relaxationDeadlineMs).To(Equal(uint64(5000)))
			})

			It("should pop on every call once patience reaches zero", func() {
				sd.bestRemainingBytes = 100
				sd.remainingBytesHistory = utilheap.NewMin[uint64]()

				// Set a high deadline to accumulate history without popping
				sd.relaxationDeadlineMs = 1000
				sd.relaxationPatienceMs = 1000

				// Push some values before the deadline
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 0, remainingBytes: 200}, monitor.logger)
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 100, remainingBytes: 300}, monitor.logger)
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 200, remainingBytes: 400}, monitor.logger)

				// Ensure nothing popped yet
				Expect(sd.bestRemainingBytes).To(Equal(uint64(100)))

				// Now simulate patience reaching 1ms and expiring
				sd.relaxationPatienceMs = 1
				sd.relaxationDeadlineMs = 0

				// First call at t=300: deadline is 0, pops 200.
				// patience = 1/2 = 0, new deadline = 300+0 = 300
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 300, remainingBytes: 999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(200)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(0)))

				// Patience is now 0, so advancing by just 1ms still pops.
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 301, remainingBytes: 999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(300)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(0)))

				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 302, remainingBytes: 999}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(400)))
				Expect(sd.relaxationPatienceMs).To(Equal(uint64(0)))
			})
		})

		Describe("processStallDetectionIteration", func() {
			BeforeEach(func() {
				sd.ewmaBandwidthBps = 1000
				sd.minRecordOutsideWindow = iterationRecord{remainingBytes: 1000}
			})

			It("should detect a new stall and initialize relaxation state", func() {
				// checkStallCondition: 1000 >= 1000 * 0.96 = 960 → stalled
				Expect(sd.processStallDetectionIteration(iterationRecord{elapsedMs: 100, remainingBytes: 1000}, monitor.logger)).To(BeTrue())
				Expect(sd.stallDetected).To(BeTrue())
				Expect(sd.bestRemainingBytes).To(Equal(uint64(1000)))
			})

			It("should return true when stall is already detected", func() {
				sd.stallDetected = true
				sd.remainingBytesHistory = utilheap.NewMin[uint64]()
				Expect(sd.processStallDetectionIteration(iterationRecord{elapsedMs: 100, remainingBytes: 1000}, monitor.logger)).To(BeTrue())
			})

			It("should return false when not stalled", func() {
				// checkStallCondition: 900 >= 1000 * 0.96 = 960 → not stalled
				Expect(sd.processStallDetectionIteration(iterationRecord{elapsedMs: 100, remainingBytes: 900}, monitor.logger)).To(BeFalse())
				Expect(sd.stallDetected).To(BeFalse())
			})

			It("should return false when switchover is already initiated", func() {
				sd.switchoverInitiated = true
				Expect(sd.processStallDetectionIteration(iterationRecord{elapsedMs: 100, remainingBytes: 900}, monitor.logger)).To(BeFalse())
			})

			It("should return false when bandwidth data is unavailable", func() {
				sd.ewmaBandwidthBps = 0
				Expect(sd.processStallDetectionIteration(iterationRecord{elapsedMs: 100, remainingBytes: 900}, monitor.logger)).To(BeFalse())
			})

			It("should detect stall on the next iteration when StallProgressTimeout is zero", func() {
				sd.stallDetectorOptions.StallProgressTimeout = 0
				sd.ewmaBandwidthBps = 1000
				sd.updateCandidates(iterationRecord{elapsedMs: 0, remainingBytes: 1000}, monitor.logger)
				Expect(sd.processStallDetectionIteration(iterationRecord{elapsedMs: 1, remainingBytes: 1000}, monitor.logger)).To(BeTrue())
				Expect(sd.stallDetected).To(BeTrue())
			})
		})

		Describe("decideAction", func() {
			BeforeEach(func() {
				sd.bestRemainingBytes = 0
				sd.ewmaBandwidthBps = 1000
			})

			It("should return actionNothing when switchover was already initiated", func() {
				sd.switchoverInitiated = true
				action, _ := sd.decideAction(iterationRecord{}, 500, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionNothing))
			})

			It("should return actionNothing when not at a local minima", func() {
				sd.bestRemainingBytes = 100
				// target = 100 * 1.04 = 104; remaining 105 > 104
				action, _ := sd.decideAction(iterationRecord{remainingBytes: 105}, 500, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionNothing))
			})

			It("should skip local minima search when SearchLocalMinima is false", func() {
				sd.bestRemainingBytes = 100
				sd.stallDetectorOptions.SearchLocalMinima = false
				action, _ := sd.decideAction(iterationRecord{remainingBytes: 200}, 500, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionSoftStopAndCopy))
			})

			It("should return actionNothing when migration cannot finish by deadline", func() {
				action, _ := sd.decideAction(iterationRecord{}, 999_999, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionNothing))
			})

			It("should return actionPostCopy when AllowPostCopy is enabled and completable", func() {
				sd.allowPostCopy = true
				sd.allowWorkloadDisruption = true
				action, _ := sd.decideAction(iterationRecord{}, 500, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionPostCopy))
			})

			It("should return actionHardStopAndCopy when AllowWorkloadDisruption is enabled and completable", func() {
				sd.allowWorkloadDisruption = true
				action, _ := sd.decideAction(iterationRecord{}, 500, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionHardStopAndCopy))
			})

			It("should return actionSoftStopAndCopy when estimated downtime is within max allowed downtime", func() {
				action, _ := sd.decideAction(iterationRecord{}, uint32(sd.maxDowntimeMs), monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionSoftStopAndCopy))
			})

			It("should return actionSoftStopAndCopy when estimated downtime is within tolerable factor of max allowed downtime", func() {
				action, _ := sd.decideAction(iterationRecord{}, uint32(sd.maxDowntimeMs)+100, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionSoftStopAndCopy))
			})

			It("should return actionAbort when estimated downtime far exceeds max allowed downtime", func() {
				sd.stallDetectorOptions.PrecopyPossibleFactor = 2.0
				estimatedDowntimeMs := uint32(float64(sd.maxDowntimeMs)*2.0) + 1
				action, _ := sd.decideAction(iterationRecord{}, estimatedDowntimeMs, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionAbort))
			})

			It("should fall through to actionHardStopAndCopy for VFIO VMI even when AllowPostCopy is true", func() {
				sd.allowPostCopy = true
				sd.allowWorkloadDisruption = true
				sd.hasVFIO = true
				action, _ := sd.decideAction(iterationRecord{}, 500, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionHardStopAndCopy))
			})

			It("should transition to a convergence action after bestRemainingBytes relaxes to current level", func() {
				sd.bestRemainingBytes = 100
				sd.remainingBytesHistory = utilheap.NewMin[uint64]()
				sd.relaxationPatienceMs = 25_000
				sd.relaxationDeadlineMs = 25_000

				remainingBytes := uint64(200)
				estimatedDowntimeMs := uint32(500)

				// Before relaxation: 200 > 100 * 1.04 = 104 → not at local minima
				action, _ := sd.decideAction(iterationRecord{remainingBytes: remainingBytes}, estimatedDowntimeMs, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionNothing))

				// Push post-stall observations; deadline not yet reached
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 1000, remainingBytes: 200}, monitor.logger)
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 5000, remainingBytes: 300}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(100)))

				// Exceed deadline → pops smallest (200) into bestRemainingBytes
				sd.relaxBestRemainingBytes(iterationRecord{elapsedMs: 25_000, remainingBytes: 400}, monitor.logger)
				Expect(sd.bestRemainingBytes).To(Equal(uint64(200)))

				// After relaxation: 200 <= 200 * 1.04 = 208 → at local minima
				action, _ = sd.decideAction(iterationRecord{remainingBytes: remainingBytes}, estimatedDowntimeMs, monitor.start, testCompletionTimeSec, monitor.logger)
				Expect(action).To(Equal(actionSoftStopAndCopy))
			})
		})

		Describe("triggerConvergenceAction", func() {
			var ctrl *gomock.Controller
			var mockDomain *cli.MockVirDomain

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				mockDomain = cli.NewMockVirDomain(ctrl)
				monitor.l.updateVMIMigrationMode(v1.MigrationPreCopy)
			})

			It("should do nothing for actionNothing", func() {
				monitor.triggerConvergenceAction(mockDomain, actionNothing, "test", monitor.logger)
				Expect(sd.switchoverInitiated).To(BeFalse())
			})

			It("should abort migration for actionAbort", func() {
				setupSuccessfulAbortContext(mockDomain)
				monitor.triggerConvergenceAction(mockDomain, actionAbort, "test abort", monitor.logger)
				Expect(monitor.isAbortInProgress()).To(BeTrue())
				Expect(sd.switchoverInitiated).To(BeTrue())
				expectMigrationAbortSucceeded()
			})

			It("should start post-copy for actionPostCopy", func() {
				mockDomain.EXPECT().MigrateStartPostCopy(gomock.Any()).Times(1).Return(nil)

				monitor.triggerConvergenceAction(mockDomain, actionPostCopy, "test post-copy", monitor.logger)
				Expect(sd.switchoverInitiated).To(BeTrue())
				Expect(monitor.isMigrationPostCopy()).To(BeTrue())
			})

			It("should reset switchoverInitiated when MigrateStartPostCopy fails", func() {
				mockDomain.EXPECT().MigrateStartPostCopy(gomock.Any()).Times(1).Return(fmt.Errorf("post-copy failed"))

				monitor.triggerConvergenceAction(mockDomain, actionPostCopy, "test post-copy", monitor.logger)
				Expect(sd.switchoverInitiated).To(BeFalse())
				Expect(monitor.isMigrationPostCopy()).To(BeFalse())
			})

			It("should set max downtime to QEMUMaxMigrationDowntimeMS for actionHardStopAndCopy", func() {
				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(migrationutils.QEMUMaxMigrationDowntimeMS), uint32(0)).Times(1).Return(nil)

				monitor.triggerConvergenceAction(mockDomain, actionHardStopAndCopy, "test hard stop", monitor.logger)
				Expect(sd.switchoverInitiated).To(BeTrue())
				elapsedSeconds := (time.Now().UTC().UnixNano() - monitor.start) / int64(time.Second)
				Expect(monitor.switchOverDeadline).To(BeNumerically("~", elapsedSeconds+int64(testSwitchoverTimeout), 2))
			})

			It("should set max downtime to maxDowntimeMs for actionSoftStopAndCopy", func() {
				mockDomain.EXPECT().MigrateSetMaxDowntime(sd.maxDowntimeMs, uint32(0)).Times(1).Return(nil)

				monitor.triggerConvergenceAction(mockDomain, actionSoftStopAndCopy, "test soft stop", monitor.logger)
				Expect(sd.switchoverInitiated).To(BeTrue())
				elapsedSeconds := (time.Now().UTC().UnixNano() - monitor.start) / int64(time.Second)
				Expect(monitor.switchOverDeadline).To(BeNumerically("~", elapsedSeconds+int64(testSwitchoverTimeout), 2))
			})

			It("should reset switchoverInitiated when MigrateSetMaxDowntime fails for actionHardStopAndCopy", func() {
				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(migrationutils.QEMUMaxMigrationDowntimeMS), uint32(0)).Times(1).Return(fmt.Errorf("set max downtime failed"))

				monitor.triggerConvergenceAction(mockDomain, actionHardStopAndCopy, "test hard stop failure", monitor.logger)
				Expect(sd.switchoverInitiated).To(BeFalse())
			})

		})

		Describe("reconcilePauseState", func() {
			var ctrl *gomock.Controller
			var mockDomain *cli.MockVirDomain

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				mockDomain = cli.NewMockVirDomain(ctrl)
				monitor.l.updateVMIMigrationMode(v1.MigrationPreCopy)
			})

			It("should transition to MigrationPaused when domain is paused for migration", func() {
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_MIGRATION), nil)

				monitor.reconcilePauseState(mockDomain, monitor.logger)
				Expect(monitor.isPausedMigration()).To(BeTrue())
			})

			It("should not transition when paused for a reason other than migration", func() {
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_USER), nil)

				monitor.reconcilePauseState(mockDomain, monitor.logger)
				Expect(monitor.isPausedMigration()).To(BeFalse())
			})

			It("should not transition when already in post-copy mode even if libvirt briefly reports DOMAIN_PAUSED_MIGRATION", func() {
				monitor.l.updateVMIMigrationMode(v1.MigrationPostCopy)
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, int(libvirt.DOMAIN_PAUSED_MIGRATION), nil)

				monitor.reconcilePauseState(mockDomain, monitor.logger)
				Expect(monitor.isMigrationPostCopy()).To(BeTrue())
				Expect(monitor.isPausedMigration()).To(BeFalse())
			})

			It("should not panic when GetState returns an error", func() {
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_NOSTATE, 0, fmt.Errorf("connection lost"))

				Expect(func() {
					monitor.reconcilePauseState(mockDomain, monitor.logger)
				}).NotTo(Panic())
				Expect(monitor.isPausedMigration()).To(BeFalse())
			})
		})

		Describe("canFinishByDeadline", func() {
			It("should return false when bandwidth data is unavailable", func() {
				sd.ewmaBandwidthBps = 0
				Expect(sd.canFinishByDeadline(0, 600, 100, monitor.logger)).To(BeFalse())
			})

			It("should return true when estimated downtime fits within remaining budget", func() {
				sd.ewmaBandwidthBps = 1000
				// budget = (600 - 100) * 1000 = 500_000ms; 5000 <= 500_000
				Expect(sd.canFinishByDeadline(100, 600, 5000, monitor.logger)).To(BeTrue())
			})

			It("should return false when estimated downtime exceeds remaining budget", func() {
				sd.ewmaBandwidthBps = 1000
				// budget = (600 - 100) * 1000 = 500_000ms; 600_000 > 500_000
				Expect(sd.canFinishByDeadline(100, 600, 600_000, monitor.logger)).To(BeFalse())
			})

			It("should return false when elapsed exceeds deadline", func() {
				sd.ewmaBandwidthBps = 1000
				// budget = (600 - 700) * 1000 = -100_000ms; 100 > -100_000 is false... wait
				// actually 100 <= -100_000 is false
				Expect(sd.canFinishByDeadline(700, 600, 100, monitor.logger)).To(BeFalse())
			})

			It("should return true when estimated downtime is zero", func() {
				sd.ewmaBandwidthBps = 1000
				Expect(sd.canFinishByDeadline(100, 600, 0, monitor.logger)).To(BeTrue())
			})
		})

		Describe("estimateDowntimeMs", func() {
			DescribeTable("should estimate correctly",
				func(ewmaBandwidthBps float64, remainingBytes uint64, expected uint32) {
					sd.ewmaBandwidthBps = ewmaBandwidthBps
					record := iterationRecord{remainingBytes: remainingBytes}
					Expect(sd.estimateDowntimeMs(record, monitor.logger)).To(Equal(expected))
				},
				Entry("returns 0 when bandwidth is zero",
					float64(0), uint64(5000), uint32(0)),
				Entry("returns 0 when remaining bytes is zero",
					float64(2000), uint64(0), uint32(0)),
				Entry("calculates correctly for typical values",
					float64(2000), uint64(5000), uint32(2500)),
				Entry("caps at MaxUint32 for extremely small bandwidth",
					float64(0.001), uint64(math.MaxUint64), uint32(math.MaxUint32)),
			)
		})

		Describe("processCompletionTimeouts", func() {
			var ctrl *gomock.Controller
			var mockDomain *cli.MockVirDomain

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				mockDomain = cli.NewMockVirDomain(ctrl)
				monitor.l.updateVMIMigrationMode(v1.MigrationPreCopy)
			})

			It("should not trigger abort when timeout has not been reached", func() {
				monitor.processCompletionTimeouts(mockDomain, 100*int64(time.Second), 0, monitor.logger)
				Expect(monitor.isAbortInProgress()).To(BeFalse())
			})

			It("should not re-trigger post-copy when already in post-copy mode", func() {
				mockDomain.EXPECT().MigrateStartPostCopy(gomock.Any()).Times(0)
				monitor.l.updateVMIMigrationMode(v1.MigrationPostCopy)
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 0, monitor.logger)
			})

			It("should start post-copy when AllowPostCopy is true and migration can finish by deadline", func() {
				monitor.options.AllowPostCopy = true
				sd.ewmaBandwidthBps = 1000

				mockDomain.EXPECT().MigrateStartPostCopy(gomock.Any()).Times(1).Return(nil)

				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 500, monitor.logger)
				Expect(monitor.isMigrationPostCopy()).To(BeTrue())
				Expect(sd.switchoverInitiated).To(BeTrue())
			})

			It("should force switchover when AllowWorkloadDisruption is true and migration can finish by deadline", func() {
				monitor.options.AllowWorkloadDisruption = true
				sd.ewmaBandwidthBps = 1000

				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(migrationutils.QEMUMaxMigrationDowntimeMS), uint32(0)).Times(1).Return(nil)

				originalTimeout := monitor.acceptableCompletionTime
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 500, monitor.logger)
				Expect(sd.switchoverInitiated).To(BeTrue())
				Expect(monitor.acceptableCompletionTime).To(Equal(originalTimeout * 2))
				elapsedSeconds := pastTimeoutNs() / int64(time.Second)
				Expect(monitor.switchOverDeadline).To(Equal(elapsedSeconds + int64(testSwitchoverTimeout)))
			})

			It("should scale acceptableCompletionTime by CompletionTimeoutFactor when forcing switchover", func() {
				monitor.options.AllowWorkloadDisruption = true
				monitor.options.StallDetectorOptions.CompletionTimeoutFactor = 3
				sd.ewmaBandwidthBps = 1000

				mockDomain.EXPECT().MigrateSetMaxDowntime(uint64(migrationutils.QEMUMaxMigrationDowntimeMS), uint32(0)).Times(1).Return(nil)

				originalTimeout := monitor.acceptableCompletionTime
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 500, monitor.logger)
				Expect(monitor.acceptableCompletionTime).To(Equal(originalTimeout * 3))
			})

			It("should trigger abort when AllowPostCopy is true but migration cannot finish by deadline", func() {
				monitor.options.AllowPostCopy = true
				sd.ewmaBandwidthBps = 100

				setupSuccessfulAbortContext(mockDomain)
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 999_999_999, monitor.logger)
				Expect(monitor.isAbortInProgress()).To(BeTrue())
				expectMigrationAbortSucceeded()
			})

			It("should trigger abort when neither AllowPostCopy nor AllowWorkloadDisruption is set", func() {
				setupSuccessfulAbortContext(mockDomain)
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 0, monitor.logger)
				Expect(monitor.isAbortInProgress()).To(BeTrue())
				expectMigrationAbortSucceeded()
			})

			It("should trigger abort directly when switchover was already initiated", func() {
				monitor.options.AllowPostCopy = true
				sd.ewmaBandwidthBps = 1000
				sd.switchoverInitiated = true

				setupSuccessfulAbortContext(mockDomain)
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 500, monitor.logger)
				Expect(monitor.isAbortInProgress()).To(BeTrue())
				expectMigrationAbortSucceeded()
			})

			It("should trigger abort when ewmaBandwidthBps is zero and timeout is reached", func() {
				sd.ewmaBandwidthBps = 0

				setupSuccessfulAbortContext(mockDomain)
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), 0, monitor.logger)
				Expect(monitor.isAbortInProgress()).To(BeTrue())
				expectMigrationAbortSucceeded()
			})

			It("should eventually abort when MigrateSetMaxDowntime keeps failing until switchover is no longer completable", func() {
				monitor.options.AllowWorkloadDisruption = true
				sd.ewmaBandwidthBps = 1000
				estimatedDowntimeMs := uint32(500)

				mockDomain.EXPECT().MigrateSetMaxDowntime(migrationutils.QEMUMaxMigrationDowntimeMS, uint32(0)).Times(1).Return(fmt.Errorf("set max downtime failed"))

				originalTimeout := monitor.acceptableCompletionTime
				monitor.processCompletionTimeouts(mockDomain, pastTimeoutNs(), estimatedDowntimeMs, monitor.logger)
				Expect(sd.switchoverInitiated).To(BeFalse())
				Expect(monitor.acceptableCompletionTime).To(Equal(originalTimeout))
				Expect(monitor.isAbortInProgress()).To(BeFalse())

				elapsedPastSwitchoverBudget := (monitor.acceptableCompletionTime*2 + 1) * int64(time.Second)
				setupSuccessfulAbortContext(mockDomain)
				monitor.processCompletionTimeouts(mockDomain, elapsedPastSwitchoverBudget, estimatedDowntimeMs, monitor.logger)
				Expect(sd.switchoverInitiated).To(BeFalse())
				Expect(monitor.isAbortInProgress()).To(BeTrue())
				expectMigrationAbortSucceeded()
			})

		})

		It("newMigrationMonitor should pass StallDetectorOptions to the stall detector", func() {
			customOptions := &cmdclient.MigrationOptions{
				MaxDowntimeMs: testMaxDowntimeMs,
				StallDetectorOptions: &cmdclient.StallDetectorOptions{
					StallMargin:             0.08,
					StallProgressTimeout:    10,
					SwitchoverTimeout:       42,
					EwmaAlpha:               0.25,
					PrecopyPossibleFactor:   2.0,
					SearchLocalMinima:       false,
					CompletionTimeoutFactor: 2,
				},
			}
			m := newMigrationMonitor(vmi, libvirtDomainManager, customOptions, make(chan struct{}, 1))
			Expect(m.stallDetector.stallDetectorOptions).To(Equal(*customOptions.StallDetectorOptions))
			Expect(m.stallDetector.maxDowntimeMs).To(Equal(customOptions.MaxDowntimeMs))
		})
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
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec, false)
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
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec, false)
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
					VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
				},
				DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{
					ClaimName:  destPvcName,
					VolumeMode: pointer.P(k8sv1.PersistentVolumeFilesystem),
				},
			},
		}
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec, false)
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
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
})
