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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

var _ = Describe("Backup", func() {
	var (
		ctrl          *gomock.Controller
		mockConn      *cli.MockConnection
		mockDomain    *cli.MockVirDomain
		manager       *StorageManager
		metadataCache *metadata.Cache
		vmi           *v1.VirtualMachineInstance
		backupOptions *backupv1.BackupOptions
		tempDir       string
	)

	const backupName = "test-backup"

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		metadataCache = metadata.NewCache()
		manager = NewStorageManager(mockConn, metadataCache)

		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
				UID:       "test-uid",
			},
		}

		var err error
		tempDir, err = os.MkdirTemp("", "backup-test")
		Expect(err).ToNot(HaveOccurred())

		now := metav1.Now()
		backupOptions = &backupv1.BackupOptions{
			BackupName:      backupName,
			BackupStartTime: &now,
			Mode:            backupv1.PushMode,
			TargetPath:      pointer.P(tempDir),
			SkipQuiesce:     true,
		}
	})

	AfterEach(func() {
		ctrl.Finish()
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	Describe("BackupVirtualMachine", func() {
		Context("when migration is in progress", func() {
			It("should fail", func() {
				// Set up migration in progress
				migrationMetadata := api.MigrationMetadata{
					StartTimestamp: pointer.P(metav1.Now()),
				}
				metadataCache.Migration.Store(migrationMetadata)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("migration"))
			})
		})

		Context("when backup is already in progress", func() {
			It("should not start another backup", func() {
				// Set up existing backup
				existingBackup := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
				}
				metadataCache.Backup.Store(existingBackup)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when a different backup is already in progress", func() {
			It("should fail", func() {
				// Set up existing backup with different timestamp
				oldTime := metav1.Time{Time: time.Now().Add(-1 * time.Hour)}
				existingBackup := api.BackupMetadata{
					Name:           "old-backup",
					StartTimestamp: &oldTime,
				}
				metadataCache.Backup.Store(existingBackup)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already in progress"))
			})
		})

		Context("when backup has already completed", func() {
			It("should fail to reinitialize", func() {
				// Set up completed backup with same timestamp
				completedTime := metav1.Now()
				existingBackup := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
					EndTimestamp:   &completedTime,
					Completed:      true,
				}
				metadataCache.Backup.Store(existingBackup)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already executed"))
			})
		})

		It("backup after failure should allow retry", func() {
			// simulate empty backup metadata from previous failure
			emptyBackup := api.BackupMetadata{}
			metadataCache.Backup.Store(emptyBackup)

			domainXML := `<domain type='kvm'>
					<devices>
						<disk type='file' device='disk'>
							<driver name='qemu' type='qcow2'/>
							<source file='/path/to/disk.qcow2'/>
							<target dev='vda' bus='virtio'/>
							<alias name='ua-disk0'/>
						</disk>
					</devices>
				</domain>`

			mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
			mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDomain.EXPECT().Free().Return(nil)

			err := manager.BackupVirtualMachine(vmi, backupOptions)
			Expect(err).ToNot(HaveOccurred())

			// Verify new backup metadata was properly initialized
			newMetadata, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(newMetadata.Name).To(Equal("test-backup"))
			Expect(newMetadata.StartTimestamp).To(Equal(backupOptions.BackupStartTime))
		})

		It("incremental backup should store checkpoint name in metadata", func() {
			backupOptions.Incremental = pointer.P("previous-checkpoint")

			domainXML := `<domain type='kvm'>
				<devices>
					<disk type='file' device='disk'>
						<driver name='qemu' type='qcow2'/>
						<source file='/path/to/disk.qcow2'/>
						<target dev='vda' bus='virtio'/>
						<alias name='ua-disk0'/>
					</disk>
				</devices>
			</domain>`

			mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
			mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDomain.EXPECT().Free().Return(nil)

			err := manager.BackupVirtualMachine(vmi, backupOptions)
			Expect(err).ToNot(HaveOccurred())

			backupMetadata, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(backupMetadata.CheckpointName).ToNot(BeEmpty())
			Expect(backupMetadata.CheckpointName).To(ContainSubstring("test-backup"))
		})

		It("should successfully initiate a pull mode backup", func() {
			backupOptions.Mode = backupv1.PullMode
			domainXML := `<domain><devices><disk type='file'><source file='/tmp/foo'/><target dev='vda'/><alias name='disk0'/></disk></devices></domain>`

			mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)

			mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDomain.EXPECT().Free().Return(nil)

			err := manager.BackupVirtualMachine(vmi, backupOptions)
			Expect(err).ToNot(HaveOccurred())

			backupMetadata, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(backupMetadata.Mode).To(Equal(string(backupv1.PullMode)))
		})
	})

	Describe("backup with freeze/thaw", func() {
		var domainXML string

		BeforeEach(func() {
			backupOptions.SkipQuiesce = false
			domainXML = `<domain type='kvm'>
				<devices>
					<disk type='file' device='disk'>
						<driver name='qemu' type='qcow2'/>
						<source file='/path/to/disk.qcow2'>
							<dataStore type='file'/>
						</source>
						<target dev='vda' bus='virtio'/>
						<alias name='ua-disk0'/>
					</disk>
				</devices>
			</domain>`
		})

		Context("successful backup with freeze and thaw", func() {
			It("should freeze, start backup, and thaw", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().FSThaw(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())

				// Verify backup metadata was initialized
				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.Name).To(Equal("test-backup"))
				Expect(backupMetadata.StartTimestamp).To(Equal(backupOptions.BackupStartTime))
				Expect(backupMetadata.CheckpointName).ToNot(BeEmpty())
				Expect(backupMetadata.CheckpointName).To(ContainSubstring("test-backup"))
				Expect(backupMetadata.Volumes).ToNot(BeEmpty())
				Expect(backupMetadata.Volumes).To(ContainSubstring("disk0"))
				Expect(backupMetadata.Volumes).To(ContainSubstring("vda"))
			})
		})

		Context("when freeze fails", func() {
			It("should continue backup without freeze and skip thaw", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(fmt.Errorf("freeze error"))
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				// FSThaw should NOT be called since freeze failed
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())

				// Verify backup message was set
				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.BackupMsg).To(ContainSubstring("Failed freezing guest filesystem"))
			})
		})

		Context("when thaw fails", func() {
			It("should record thaw failure in metadata", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().FSThaw(gomock.Any(), gomock.Any()).Return(fmt.Errorf("thaw error"))
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())

				// Verify thaw failure was recorded
				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.BackupMsg).To(Equal(unfreezeFailedMsg))
			})
		})

		Context("when BackupBegin fails after freeze", func() {
			It("should still thaw the filesystem", func() {
				backupOptions.SkipQuiesce = false // Ensure quiesce is enabled

				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("backup begin failed"))
				mockDomain.EXPECT().FSThaw(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("backup begin failed"))

				// Verify backup metadata was cleared due to failure
				// The metadata cache stores an empty BackupMetadata on failure
				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue()) // An empty backup metadata is stored
				Expect(backupMetadata.Name).To(BeEmpty())
			})
		})

		Context("when SkipQuiesce is true", func() {
			It("should not freeze or thaw", func() {
				backupOptions.SkipQuiesce = true

				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				// FSFreeze and FSThaw should NOT be called
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("generateDomainBackup", func() {
		It("should generate backup XML for disks with DataStore", func() {
			disks := []api.Disk{
				{
					Target: api.DiskTarget{
						Device: "vda",
					},
					Source: api.DiskSource{
						DataStore: &api.DataStore{},
					},
					Alias: api.NewUserDefinedAlias("disk0"),
				},
			}

			domainBackup, domainCheckpoint, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainBackup).ToNot(BeNil())
			Expect(domainBackup.Mode).To(Equal(string(backupv1.PushMode)))
			Expect(domainBackup.Incremental).To(BeNil())
			Expect(domainBackup.BackupDisks).ToNot(BeNil())
			Expect(domainBackup.BackupDisks.Disks).To(HaveLen(1))
			Expect(domainBackup.BackupDisks.Disks[0].Name).To(Equal("vda"))
			Expect(domainBackup.BackupDisks.Disks[0].Backup).To(Equal("yes"))
			Expect(domainBackup.BackupDisks.Disks[0].Type).To(Equal("file"))
			Expect(domainBackup.BackupDisks.Disks[0].ExportName).To(BeEmpty())
			Expect(domainBackup.BackupDisks.Disks[0].ExportBitmap).To(BeEmpty())

			Expect(domainCheckpoint).ToNot(BeNil())
			Expect(domainCheckpoint.Name).To(ContainSubstring("test-backup"))
			Expect(domainCheckpoint.CheckpointDisks).ToNot(BeNil())
			Expect(domainCheckpoint.CheckpointDisks.Disks).To(HaveLen(1))
			Expect(domainCheckpoint.CheckpointDisks.Disks[0].Checkpoint).To(Equal("bitmap"))
			Expect(volumesInfo).To(HaveLen(1))
			Expect(volumesInfo[0].VolumeName).To(Equal("disk0"))
			Expect(volumesInfo[0].DiskTarget).To(Equal("vda"))
		})

		It("should populate exportbitmap and exportname for disks with a DataStore for pull mode backup", func() {
			backupOptions.Mode = backupv1.PullMode
			disks := []api.Disk{
				{
					Target: api.DiskTarget{
						Device: "vda",
					},
					Source: api.DiskSource{
						DataStore: &api.DataStore{},
					},
					Alias: api.NewUserDefinedAlias("disk0"),
				},
			}
			domainBackup, domainCheckpoint, _ := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainCheckpoint).ToNot(BeNil())
			Expect(domainBackup).ToNot(BeNil())
			Expect(domainBackup.Mode).To(Equal(string(backupv1.PullMode)))
			Expect(domainBackup.Incremental).To(BeNil())
			Expect(domainBackup.BackupDisks).ToNot(BeNil())
			Expect(domainBackup.BackupDisks.Disks).To(HaveLen(1))
			Expect(domainBackup.BackupDisks.Disks[0].Name).To(Equal("vda"))
			Expect(domainBackup.BackupDisks.Disks[0].Backup).To(Equal("yes"))
			Expect(domainBackup.BackupDisks.Disks[0].Type).To(Equal("file"))
			Expect(domainBackup.BackupDisks.Disks[0].ExportName).To(Equal("disk0"))
			Expect(domainBackup.BackupDisks.Disks[0].ExportBitmap).To(Equal(domainCheckpoint.Name))

		})

		It("should skip disks without DataStore", func() {
			disks := []api.Disk{
				{
					Target: api.DiskTarget{
						Device: "vda",
					},
					Source: api.DiskSource{
						// No DataStore
					},
					Alias: api.NewUserDefinedAlias("disk0"),
				},
			}

			domainBackup, domainCheckpoint, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainBackup.BackupDisks.Disks).To(HaveLen(1))
			Expect(domainBackup.BackupDisks.Disks[0].Backup).To(Equal("no"))
			Expect(domainCheckpoint.CheckpointDisks.Disks[0].Checkpoint).To(Equal("no"))
			Expect(volumesInfo).To(BeEmpty())
		})
		It("should handle incremental backups", func() {
			incremental := "previous-checkpoint"
			backupOptions.Incremental = &incremental

			disks := []api.Disk{
				{
					Target: api.DiskTarget{Device: "vda"},
					Source: api.DiskSource{DataStore: &api.DataStore{}},
					Alias:  api.NewUserDefinedAlias("disk0"),
				},
			}

			domainBackup, _, _ := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainBackup.Incremental).ToNot(BeNil())
			Expect(*domainBackup.Incremental).To(Equal("previous-checkpoint"))
		})

		It("should not set incremental field when Incremental is empty string", func() {
			backupOptions.Incremental = pointer.P("")

			disks := []api.Disk{
				{
					Target: api.DiskTarget{Device: "vda"},
					Source: api.DiskSource{DataStore: &api.DataStore{}},
					Alias:  api.NewUserDefinedAlias("disk0"),
				},
			}

			domainBackup, _, _ := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainBackup.Incremental).To(BeNil())
		})

		It("should return volumes info for multiple disks with DataStore", func() {
			disks := []api.Disk{
				{
					Target: api.DiskTarget{Device: "vda"},
					Source: api.DiskSource{DataStore: &api.DataStore{}},
					Alias:  api.NewUserDefinedAlias("rootdisk"),
				},
				{
					Target: api.DiskTarget{Device: "vdb"},
					Source: api.DiskSource{DataStore: &api.DataStore{}},
					Alias:  api.NewUserDefinedAlias("datadisk"),
				},
				{
					Target: api.DiskTarget{Device: "sda"},
					Source: api.DiskSource{
						// No DataStore - should be skipped
					},
					Alias: api.NewUserDefinedAlias("cdrom"),
				},
			}

			_, _, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(volumesInfo).To(HaveLen(2))
			Expect(volumesInfo[0].VolumeName).To(Equal("rootdisk"))
			Expect(volumesInfo[0].DiskTarget).To(Equal("vda"))
			Expect(volumesInfo[1].VolumeName).To(Equal("datadisk"))
			Expect(volumesInfo[1].DiskTarget).To(Equal("vdb"))
		})
	})

	Describe("HandleBackupJobCompletedEvent", func() {
		var (
			mockDomain *cli.MockVirDomain
			event      *libvirt.DomainEventJobCompleted
		)

		BeforeEach(func() {
			mockDomain = cli.NewMockVirDomain(ctrl)
			event = &libvirt.DomainEventJobCompleted{
				Info: libvirt.DomainJobInfo{
					Type: libvirt.DOMAIN_JOB_COMPLETED,
				},
			}

			// Initialize backup metadata
			backupMetadata := api.BackupMetadata{
				Name:           "test-backup",
				StartTimestamp: backupOptions.BackupStartTime,
				SkipQuiesce:    false,
			}
			metadataCache.Backup.Store(backupMetadata)
		})

		Context("when backup completes successfully", func() {
			It("should update metadata with completion", func() {
				mockDomain.EXPECT().GetJobStats(gomock.Any()).Return(&libvirt.DomainJobInfo{
					Type: libvirt.DOMAIN_JOB_COMPLETED,
				}, nil)

				HandleBackupJobCompletedEvent(mockDomain, event, metadataCache)

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.Completed).To(BeTrue())
				Expect(backupMetadata.EndTimestamp).ToNot(BeNil())
			})
		})

		Context("when no backup metadata exists", func() {
			It("should log warning and return", func() {
				// Create fresh cache with no metadata
				freshCache := metadata.NewCache()

				// HandleBackupJobCompletedEvent should not call GetJobStats if metadata doesn't exist
				HandleBackupJobCompletedEvent(mockDomain, event, freshCache)

				// Should not panic and backup metadata should remain empty
				backupMetadata, exists := freshCache.Backup.Load()
				Expect(exists).To(BeFalse())
				Expect(backupMetadata.Name).To(BeEmpty())
			})
		})

		Context("when GetJobStats fails", func() {
			It("should still complete the backup", func() {
				mockDomain.EXPECT().GetJobStats(gomock.Any()).Return(nil, fmt.Errorf("stats error"))

				HandleBackupJobCompletedEvent(mockDomain, event, metadataCache)

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.Completed).To(BeTrue())
			})
		})

		Context("when domain is nil", func() {
			It("should still complete the backup", func() {
				HandleBackupJobCompletedEvent(nil, event, metadataCache)

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.Completed).To(BeTrue())
			})
		})

		Context("backup failure handling", func() {
			DescribeTable("should update the metadata correctly for backup failures", func(domainJobInfo libvirt.DomainJobInfo, message string) {
				metadataCache.Backup.WithSafeBlock(func(backupMetadata *api.BackupMetadata, _ bool) {
					backupMetadata.Mode = string(backupv1.PushMode)
				})
				event.Info = domainJobInfo
				mockDomain.EXPECT().GetJobStats(gomock.Any()).Return(&domainJobInfo, nil)

				HandleBackupJobCompletedEvent(mockDomain, event, metadataCache)

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.Completed).To(BeTrue())
				Expect(backupMetadata.Failed).To(BeTrue())
				Expect(backupMetadata.BackupMsg).To(Equal(message))
			},
				Entry("with cancellation for a Push mode backup", libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_CANCELLED}, "backup aborted"),
				Entry("with failure and error message", libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_FAILED, ErrorMessageSet: true, ErrorMessage: "failure"}, "failure"),
				Entry("with failure and no error message", libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_FAILED}, "unknown failure reason"),
				Entry("with an unknown job completion type", libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_BOUNDED}, fmt.Sprintf("unexpected job completion type: %d", libvirt.DOMAIN_JOB_BOUNDED)),
			)
		})

		Context("abort backup", func() {
			DescribeTable("should successfully abort an ongoing backup and update the status",
				func(backupMode backupv1.BackupMode, failed types.GomegaMatcher) {
					backupMetadata := api.BackupMetadata{
						Name:           backupOptions.BackupName,
						Mode:           string(backupMode),
						StartTimestamp: backupOptions.BackupStartTime,
					}
					metadataCache.Backup.Store(backupMetadata)

					validJob := &libvirt.DomainJobInfo{
						Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
						Type:      libvirt.DOMAIN_JOB_UNBOUNDED,
					}
					mockConn.EXPECT().LookupDomainByName(gomock.Any()).MaxTimes(1).Return(mockDomain, nil)
					mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(validJob, nil)
					mockDomain.EXPECT().AbortJob().Return(nil)
					mockDomain.EXPECT().Free().MaxTimes(1).Return(nil)

					Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(Succeed())

					event := &libvirt.DomainEventJobCompleted{}
					mockDomain.EXPECT().GetJobStats(gomock.Any()).Return(&libvirt.DomainJobInfo{
						Type: libvirt.DOMAIN_JOB_CANCELLED,
					}, nil)

					HandleBackupJobCompletedEvent(mockDomain, event, metadataCache)

					newMetadata, exists := metadataCache.Backup.Load()
					Expect(exists).To(BeTrue())
					Expect(newMetadata.Failed).To(failed)

				},
				Entry("marking the backup as failed for push mode", backupv1.PushMode, BeTrue()),
				Entry("not marking the backup as failed for pull mode", backupv1.PullMode, BeFalse()),
			)

			DescribeTable("should fail to abort a backup that is not associated with the domain", func(name string, timestamp metav1.Time) {
				backupMetadata := api.BackupMetadata{
					Name:           name,
					Mode:           string(backupv1.PushMode),
					StartTimestamp: &timestamp,
				}
				metadataCache.Backup.Store(backupMetadata)
				backupOptions.BackupStartTime = pointer.P(metav1.Date(1, 1, 1, 1, 1, 1, 1, time.Local))

				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(
						ContainSubstring("failed to abort backup: requested backup differs from ongoing one"),
					),
				)
			},
				Entry("with backup name mismatch",
					"wrong-name",
					metav1.Date(1, 1, 1, 1, 1, 1, 1, time.Local),
				),
				Entry("with start timestamp mismatch",
					backupName,
					metav1.Date(2, 2, 2, 2, 2, 2, 2, time.Local),
				),
			)

			It("should fail to abort a backup that hasn't started yet", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: nil,
				}
				metadataCache.Backup.Store(backupMetadata)

				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(
						ContainSubstring("failed to abort backup: backup did not start yet"),
					),
				)
			})

			It("should fail to abort a backup that already completed", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
					Completed:      true,
				}
				metadataCache.Backup.Store(backupMetadata)

				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(
						ContainSubstring("failed to abort backup: backup already completed"),
					),
				)
			})

			It("should return an error when the libvirt domain is not found", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
				}
				metadataCache.Backup.Store(backupMetadata)

				noDomainError := libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).MaxTimes(1).Return(nil, noDomainError)
				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(
						ContainSubstring(noDomainError.Error()),
					),
				)
			})

			It("should return an error when failing to get domain job info", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
				}
				metadataCache.Backup.Store(backupMetadata)

				jobInfoError := libvirt.Error{Code: libvirt.ERR_INTERNAL_ERROR}
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(nil, jobInfoError)
				mockDomain.EXPECT().Free().MaxTimes(1).Return(nil)
				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(
						ContainSubstring(jobInfoError.Error()),
					),
				)
			})

			DescribeTable("should return an error when the domain job is wrong", func(jobOperation libvirt.DomainJobOperationType, jobType libvirt.DomainJobType) {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
				}
				metadataCache.Backup.Store(backupMetadata)

				wrongJob := &libvirt.DomainJobInfo{
					Type:      jobType,
					Operation: jobOperation,
				}
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(wrongJob, nil)
				mockDomain.EXPECT().Free().MaxTimes(1).Return(nil)
				expectedErr := fmt.Sprintf("cannot abort backup, wrong operation or type: %d, %d", jobOperation, jobType)
				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(ContainSubstring(expectedErr)),
				)
			},
				Entry("with wrong job operation", libvirt.DOMAIN_JOB_OPERATION_MIGRATION_IN, libvirt.DOMAIN_JOB_UNBOUNDED),
				Entry("with wrong job type", libvirt.DOMAIN_JOB_OPERATION_BACKUP, libvirt.DOMAIN_JOB_BOUNDED),
			)

			It("should return an error when the abort job call fails", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
				}
				metadataCache.Backup.Store(backupMetadata)

				validJob := &libvirt.DomainJobInfo{
					Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
					Type:      libvirt.DOMAIN_JOB_UNBOUNDED,
				}
				abortError := libvirt.Error{Code: libvirt.ERR_INTERNAL_ERROR}
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(validJob, nil)
				mockDomain.EXPECT().AbortJob().Return(abortError)
				mockDomain.EXPECT().Free().MaxTimes(1).Return(nil)
				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(
					MatchError(
						ContainSubstring(abortError.Error()),
					),
				)
			})
		})
	})

	Describe("utility functions", func() {
		Describe("getBackupPath", func() {
			It("should create correct path", func() {
				path := getBackupPath(backupOptions, "test-vmi")
				Expect(path).To(ContainSubstring(tempDir))
				Expect(path).To(ContainSubstring("test-vmi"))
				Expect(path).To(ContainSubstring("test-backup"))
			})
		})

		Describe("targetQCOW2File", func() {
			It("should create correct filename", func() {
				file := targetQCOW2File(tempDir, "backup1", "disk0")
				Expect(file).To(Equal(filepath.Join(tempDir, "backup1-disk0.qcow2")))
			})
		})

		Describe("backupTimeFormatted", func() {
			It("should format time correctly", func() {
				now := metav1.Now()
				formatted := backupTimeFormatted(&now)
				Expect(formatted).To(MatchRegexp(`\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}`))
			})
		})
	})

	Describe("error handling during backup", func() {
		Context("when domain lookup fails", func() {
			It("should return error", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(nil, fmt.Errorf("domain not found"))

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("domain not found"))
			})
		})

		Context("when getting domain disks fails", func() {
			It("should return error", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return("", fmt.Errorf("xml error"))
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating backup directory fails", func() {
			It("should return error", func() {
				// Use a path where a file exists as parent - mkdir will fail
				// because you can't create a directory inside a file
				invalidBackupOptions := backupOptions.DeepCopy()
				invalidBackupOptions.TargetPath = pointer.P("/dev/null/subdir")

				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(`<domain/>`, nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, invalidBackupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("error creating dir for backup"))
			})
		})
	})

	Describe("cleanup on failure", func() {
		Context("when backup fails after creating directory", func() {
			It("should clean up the backup directory", func() {
				domainXML := `<domain><devices><disk><target dev='vda'/><alias name='ua-disk0'/></disk></devices></domain>`
				backupOptions.SkipQuiesce = true // Skip freeze/thaw for this test

				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("backup failed"))
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())

				// Directory should be cleaned up
				backupPath := getBackupPath(backupOptions, vmi.Name)
				_, err = os.Stat(backupPath)
				Expect(os.IsNotExist(err)).To(BeTrue())
			})
		})
	})

	Describe("findDisksWithCheckpointBitmap", func() {
		const checkpointName = "checkpoint-1"

		It("should find disks with checkpoint bitmap and ignore disks without DataStore", func() {
			domainXML := `<domain>
				<devices>
					<disk type="file" device="disk">
						<source file="/var/run/kubevirt-private/vmi-disks/disk1/disk.qcow2">
							<dataStore>
								<source file="/var/lib/kubevirt/disks/disk1-backing.qcow2"/>
							</dataStore>
						</source>
						<target dev="vda"/>
					</disk>
					<disk type="file" device="cdrom">
						<source file="/var/run/kubevirt-private/cdrom/cd.iso"/>
						<target dev="sda"/>
					</disk>
				</devices>
			</domain>`

			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
			getDiskInfoWithForceShare = mockGetDiskInfoWithForceShare(checkpointName)

			result, disksWithoutBitmap, err := findDisksWithCheckpointBitmap(mockDomain, checkpointName)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Disks).To(HaveLen(1))
			Expect(result.Disks[0].Name).To(Equal("vda"))
			Expect(result.Disks[0].Checkpoint).To(Equal("bitmap"))
			Expect(disksWithoutBitmap).To(BeEmpty())
		})

		It("should return disk in disksWithoutBitmap when bitmap is not found", func() {
			domainXML := `<domain>
				<devices>
					<disk type="file" device="disk">
						<source file="/var/run/kubevirt-private/vmi-disks/disk1/disk.qcow2">
							<dataStore>
								<source file="/var/lib/kubevirt/disks/disk1-backing.qcow2"/>
							</dataStore>
						</source>
						<target dev="vda"/>
					</disk>
				</devices>
			</domain>`

			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
			getDiskInfoWithForceShare = mockGetDiskInfoWithForceShare("other-checkpoint")

			result, disksWithoutBitmap, err := findDisksWithCheckpointBitmap(mockDomain, checkpointName)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Disks).To(BeEmpty())
			Expect(disksWithoutBitmap).To(HaveLen(1))
			Expect(disksWithoutBitmap[0]).To(Equal("vda"))
		})

		It("should find multiple disks with checkpoint bitmap", func() {
			domainXML := `<domain>
				<devices>
					<disk type="file" device="disk">
						<source file="/var/run/kubevirt-private/vmi-disks/disk1/disk.qcow2">
							<dataStore>
								<source file="/var/lib/kubevirt/disks/disk1-backing.qcow2"/>
							</dataStore>
						</source>
						<target dev="vda"/>
					</disk>
					<disk type="file" device="disk">
						<source file="/var/run/kubevirt-private/vmi-disks/disk2/disk.qcow2">
							<dataStore>
								<source file="/var/lib/kubevirt/disks/disk2-backing.qcow2"/>
							</dataStore>
						</source>
						<target dev="vdb"/>
					</disk>
				</devices>
			</domain>`

			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
			getDiskInfoWithForceShare = mockGetDiskInfoWithForceShare(checkpointName)

			result, disksWithoutBitmap, err := findDisksWithCheckpointBitmap(mockDomain, checkpointName)

			Expect(err).ToNot(HaveOccurred())
			Expect(result.Disks).To(HaveLen(2))
			Expect(result.Disks[0].Name).To(Equal("vda"))
			Expect(result.Disks[1].Name).To(Equal("vdb"))
			Expect(disksWithoutBitmap).To(BeEmpty())
		})
	})
})

func mockGetDiskInfoWithForceShare(bitmapName string) func(path string) (*osdisk.DiskInfo, error) {
	return func(path string) (*osdisk.DiskInfo, error) {
		var bitmaps []osdisk.BitmapInfo
		if bitmapName != "" {
			bitmaps = []osdisk.BitmapInfo{{Name: bitmapName, Granularity: 65536}}
		}
		return &osdisk.DiskInfo{
			Format:      "qcow2",
			VirtualSize: 10737418240,
			FormatSpecific: &osdisk.FormatSpecific{
				Type: "qcow2",
				Data: &osdisk.FormatSpecificData{Bitmaps: bitmaps},
			},
		}, nil
	}
}
