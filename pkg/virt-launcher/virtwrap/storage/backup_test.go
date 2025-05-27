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
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"

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
			BackupName:      "test-backup",
			BackupStartTime: &now,
			Mode:            backupv1.PushMode,
			PushPath:        pointer.P(tempDir),
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
	})

	Describe("backup with freeze/thaw", func() {
		var domainXML string

		BeforeEach(func() {
			backupOptions.SkipQuiesce = false
			domainXML = `<domain type='kvm'>
				<devices>
					<disk type='file' device='disk'>
						<driver name='qemu' type='qcow2'/>
						<source file='/path/to/disk.qcow2'/>
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

			domainBackup, domainCheckpoint := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainBackup).ToNot(BeNil())
			Expect(domainBackup.Mode).To(Equal(string(backupv1.PushMode)))
			Expect(domainBackup.BackupDisks).ToNot(BeNil())
			Expect(domainBackup.BackupDisks.Disks).To(HaveLen(1))
			Expect(domainBackup.BackupDisks.Disks[0].Name).To(Equal("vda"))
			Expect(domainBackup.BackupDisks.Disks[0].Backup).To(Equal("yes"))
			Expect(domainBackup.BackupDisks.Disks[0].Type).To(Equal("file"))

			Expect(domainCheckpoint).ToNot(BeNil())
			Expect(domainCheckpoint.Name).To(ContainSubstring("test-backup"))
			Expect(domainCheckpoint.CheckpointDisks).ToNot(BeNil())
			Expect(domainCheckpoint.CheckpointDisks.Disks).To(HaveLen(1))
			Expect(domainCheckpoint.CheckpointDisks.Disks[0].Checkpoint).To(Equal("bitmap"))
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

			domainBackup, domainCheckpoint := generateDomainBackup(disks, backupOptions, tempDir)

			Expect(domainBackup.BackupDisks.Disks).To(HaveLen(1))
			Expect(domainBackup.BackupDisks.Disks[0].Backup).To(Equal("no"))
			Expect(domainCheckpoint.CheckpointDisks.Disks[0].Checkpoint).To(Equal("no"))
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

		Context("when job type is not completed", func() {
			It("should log warning but still update metadata", func() {
				event.Info.Type = libvirt.DOMAIN_JOB_FAILED
				mockDomain.EXPECT().GetJobStats(gomock.Any()).Return(&libvirt.DomainJobInfo{
					Type: libvirt.DOMAIN_JOB_FAILED,
				}, nil)

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
				invalidBackupOptions.PushPath = pointer.P("/dev/null/subdir")

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
})
