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
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	backupv1 "kubevirt.io/api/backup/v1beta1"
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

	const backupName = "test-backup"
	runningBackupJob := &libvirt.DomainJobInfo{Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP, Type: libvirt.DOMAIN_JOB_UNBOUNDED}

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
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
			mockDomain.EXPECT().Free().Return(nil)

			err := manager.BackupVirtualMachine(vmi, backupOptions)
			Expect(err).ToNot(HaveOccurred())

			// Verify new backup metadata was properly initialized
			newMetadata, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(newMetadata.Name).To(Equal("test-backup"))
			Expect(newMetadata.StartTimestamp).To(Equal(backupOptions.BackupStartTime))
			Expect(newMetadata.Mode).To(Equal(string(backupv1.PushMode)))
		})

		Context("incremental backup", func() {
			const baseCheckpoint = "previous-checkpoint"

			var redefinedDisks [][]string

			twoDiskDomainXML := `<domain type='kvm'>
				<devices>
					<disk type='file' device='disk'>
						<source file='/path/to/vda.qcow2'>
							<dataStore>
								<source file='/path/to/vda-backing.img'/>
							</dataStore>
						</source>
						<target dev='vda' bus='virtio'/>
						<alias name='ua-rootdisk'/>
					</disk>
					<disk type='file' device='disk'>
						<source file='/path/to/vdb.qcow2'>
							<dataStore>
								<source file='/path/to/vdb-backing.img'/>
							</dataStore>
						</source>
						<target dev='vdb' bus='virtio'/>
						<alias name='ua-datadisk'/>
					</disk>
				</devices>
			</domain>`

			expectBaseRedefinition := func(withoutBitmap ...string) {
				mockDomain.EXPECT().CreateCheckpointXML(gomock.Any(), libvirt.DOMAIN_CHECKPOINT_CREATE_REDEFINE|libvirt.DOMAIN_CHECKPOINT_CREATE_REDEFINE_VALIDATE).
					DoAndReturn(func(checkpointXML string, _ libvirt.DomainCheckpointCreateFlags) (*libvirt.DomainCheckpoint, error) {
						def := &api.DomainCheckpoint{}
						Expect(xml.Unmarshal([]byte(checkpointXML), def)).To(Succeed())
						Expect(def.Name).To(Equal(baseCheckpoint))
						Expect(def.CreationTime).To(HaveValue(Equal(uint64(backupOptions.IncrementalCreationTime.Unix()))))
						var disks []string
						for _, d := range def.CheckpointDisks.Disks {
							disks = append(disks, d.Name)
						}
						redefinedDisks = append(redefinedDisks, disks)
						for _, d := range disks {
							if slices.Contains(withoutBitmap, d) {
								return nil, libvirt.Error{Code: libvirt.ERR_CHECKPOINT_INCONSISTENT}
							}
						}
						return nil, nil
					}).AnyTimes()
			}

			captureBackupXML := func() *api.DomainBackup {
				domainBackup := &api.DomainBackup{}
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(backupXML, _ string, _ libvirt.DomainBackupBeginFlags) error {
						return xml.Unmarshal([]byte(backupXML), domainBackup)
					})
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
				return domainBackup
			}

			BeforeEach(func() {
				backupOptions.Incremental = pointer.P(baseCheckpoint)
				backupOptions.IncrementalCreationTime = pointer.P(metav1.NewTime(time.Now().Add(-time.Hour)))
				redefinedDisks = nil
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(twoDiskDomainXML, nil)
				mockDomain.EXPECT().Free().Return(nil)
			})

			It("should redefine the base checkpoint and back up every disk against it", func() {
				expectBaseRedefinition()
				domainBackup := captureBackupXML()

				Expect(manager.BackupVirtualMachine(vmi, backupOptions)).To(Succeed())

				Expect(redefinedDisks).To(HaveExactElements([]string{"vda"}, []string{"vdb"}, []string{"vda", "vdb"}))
				Expect(domainBackup.Incremental).To(HaveValue(Equal(baseCheckpoint)))
				Expect(domainBackup.BackupDisks.Disks).To(HaveEach(HaveField("BackupMode", BeEmpty())))

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.CheckpointName).To(ContainSubstring(backupName))
				Expect(backupMetadata.Volumes).To(MatchJSON(`[
					{"volumeName":"rootdisk","type":"Incremental"},
					{"volumeName":"datadisk","type":"Incremental"}
				]`))
			})

			It("should back up in full only the disks that lost the base bitmap", func() {
				expectBaseRedefinition("vdb")
				domainBackup := captureBackupXML()

				Expect(manager.BackupVirtualMachine(vmi, backupOptions)).To(Succeed())

				Expect(redefinedDisks).To(HaveExactElements([]string{"vda"}, []string{"vdb"}, []string{"vda"}))
				Expect(domainBackup.Incremental).To(HaveValue(Equal(baseCheckpoint)))
				Expect(domainBackup.BackupDisks.Disks).To(HaveExactElements(
					HaveField("BackupMode", BeEmpty()),
					HaveField("BackupMode", Equal("full")),
				))

				backupMetadata, _ := metadataCache.Backup.Load()
				Expect(backupMetadata.Volumes).To(MatchJSON(`[
					{"volumeName":"rootdisk","type":"Incremental"},
					{"volumeName":"datadisk","type":"Full"}
				]`))
			})

			It("should take a full backup when no disk kept the base bitmap", func() {
				expectBaseRedefinition("vda", "vdb")
				domainBackup := captureBackupXML()

				Expect(manager.BackupVirtualMachine(vmi, backupOptions)).To(Succeed())

				Expect(domainBackup.Incremental).To(BeNil())
				Expect(domainBackup.BackupDisks.Disks).To(HaveEach(HaveField("BackupMode", Equal("full"))))
			})

			It("should fail without starting a backup when the base cannot be redefined", func() {
				mockDomain.EXPECT().CreateCheckpointXML(gomock.Any(), gomock.Any()).
					Return(nil, libvirt.Error{Code: libvirt.ERR_OPERATION_INVALID, Message: "domain is not running"})

				Expect(manager.BackupVirtualMachine(vmi, backupOptions)).To(MatchError(ContainSubstring("domain is not running")))
			})
		})

		It("should successfully initiate a pull mode backup", func() {
			backupOptions.Mode = backupv1.PullMode
			DeferCleanup(func(f func(*v1.VirtualMachineInstance) (string, error)) { preparePullBackupSocket = f }, preparePullBackupSocket)
			preparePullBackupSocket = func(*v1.VirtualMachineInstance) (string, error) { return "/backup-nbd-sock", nil }
			domainXML := `<domain><devices><disk type='file'><source file='/tmp/foo'/><target dev='vda'/><alias name='disk0'/></disk></devices></domain>`

			mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)

			mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
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
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"thawed"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"frozen"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSThaw(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)
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
				Expect(backupMetadata.JobStarted).To(BeTrue())
			})
		})

		Context("when freeze fails", func() {
			It("should continue backup with QuiesceStatus=Failed", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"thawed"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(fmt.Errorf("freeze error"))
				mockDomain.EXPECT().Free().Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"thawed"}`, nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.QuiesceStatus).To(Equal(string(backupv1.QuiesceFailed)))
			})
		})

		Context("when thaw fails", func() {
			It("should record thaw failure in metadata", func() {
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"thawed"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"frozen"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSThaw(gomock.Any(), gomock.Any()).Return(fmt.Errorf("thaw error"))
				mockDomain.EXPECT().Free().Return(nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(backupMetadata.QuiesceStatus).To(Equal(string(backupv1.QuiesceSucceeded)))
			})
		})

		Context("when BackupBegin fails after freeze", func() {
			It("should still thaw the filesystem", func() {
				backupOptions.SkipQuiesce = false

				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"thawed"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSFreeze(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)
				mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("backup begin failed"))
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), gomock.Any()).Return(`{"return":"frozen"}`, nil)
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().FSThaw(gomock.Any(), gomock.Any()).Return(nil)
				mockDomain.EXPECT().Free().Return(nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("backup begin failed"))

				backupMetadata, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
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
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(runningBackupJob, nil)
				mockDomain.EXPECT().Free().Return(nil)

				err := manager.BackupVirtualMachine(vmi, backupOptions)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("should record the result of a job that ended before it was marked started", func() {
			backupOptions.SkipQuiesce = true
			mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXML, nil)
			mockDomain.EXPECT().BackupBegin(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_NONE}, nil)
			mockDomain.EXPECT().GetJobStats(libvirt.DOMAIN_JOB_STATS_COMPLETED|libvirt.DOMAIN_JOB_STATS_KEEP_COMPLETED).Return(&libvirt.DomainJobInfo{
				Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
				Type:      libvirt.DOMAIN_JOB_COMPLETED,
			}, nil)
			mockDomain.EXPECT().Free().Return(nil)

			Expect(manager.BackupVirtualMachine(vmi, backupOptions)).To(Succeed())

			backupMetadata, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(backupMetadata.Completed).To(BeTrue())
			Expect(backupMetadata.Failed).To(BeFalse())
		})
	})

	Describe("generateDomainBackup", func() {
		const nbdSocketPath = "/var/run/kubevirt-private/libvirt/qemu/cbt-nbd/backup-nbd-sock"

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

			domainBackup, domainCheckpoint, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, nil)

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
			domainBackup, domainCheckpoint, _ := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, nil)

			Expect(domainCheckpoint).ToNot(BeNil())
			Expect(domainBackup).ToNot(BeNil())
			Expect(domainBackup.Mode).To(Equal(string(backupv1.PullMode)))
			Expect(domainBackup.Server).To(Equal(&api.DomainBackupServer{
				Transport: api.BackupUnixTransport,
				Socket:    nbdSocketPath,
			}))
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

			domainBackup, domainCheckpoint, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, nil)

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

			domainBackup, _, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, nil)

			Expect(domainBackup.Incremental).ToNot(BeNil())
			Expect(*domainBackup.Incremental).To(Equal("previous-checkpoint"))
			Expect(domainBackup.BackupDisks.Disks[0].BackupMode).To(BeEmpty())
			Expect(volumesInfo).To(ConsistOf(
				v1.VirtualMachineInstanceBackupVolumeInfo{VolumeName: "disk0", Type: string(backupv1.Incremental)},
			))
		})

		It("should mark only the disks that lost their bitmap", func() {
			backupOptions.Incremental = pointer.P("previous-checkpoint")

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
			}

			domainBackup, _, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, sets.New("vdb"))

			Expect(domainBackup.Incremental).ToNot(BeNil())
			Expect(*domainBackup.Incremental).To(Equal("previous-checkpoint"))
			Expect(domainBackup.BackupDisks.Disks[0].BackupMode).To(BeEmpty())
			Expect(domainBackup.BackupDisks.Disks[1].BackupMode).To(Equal("full"))

			Expect(volumesInfo).To(ConsistOf(
				v1.VirtualMachineInstanceBackupVolumeInfo{VolumeName: "rootdisk", Type: string(backupv1.Incremental)},
				v1.VirtualMachineInstanceBackupVolumeInfo{VolumeName: "datadisk", Type: string(backupv1.Full)},
			))
		})

		It("should omit the incremental base when no disk kept its bitmap", func() {
			backupOptions.Incremental = pointer.P("previous-checkpoint")

			disks := []api.Disk{
				{
					Target: api.DiskTarget{Device: "vda"},
					Source: api.DiskSource{DataStore: &api.DataStore{}},
					Alias:  api.NewUserDefinedAlias("rootdisk"),
				},
			}

			domainBackup, _, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, sets.New("vda"))

			Expect(domainBackup.Incremental).To(BeNil())
			Expect(domainBackup.BackupDisks.Disks[0].BackupMode).To(Equal("full"))
			Expect(volumesInfo).To(ConsistOf(
				v1.VirtualMachineInstanceBackupVolumeInfo{VolumeName: "rootdisk", Type: string(backupv1.Full)},
			))
		})

		It("should report every volume as full for a non-incremental backup", func() {
			disks := []api.Disk{
				{
					Target: api.DiskTarget{Device: "vda"},
					Source: api.DiskSource{DataStore: &api.DataStore{}},
					Alias:  api.NewUserDefinedAlias("rootdisk"),
				},
			}

			domainBackup, _, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, nil)

			Expect(domainBackup.Incremental).To(BeNil())
			Expect(domainBackup.BackupDisks.Disks[0].BackupMode).To(BeEmpty())
			Expect(volumesInfo).To(ConsistOf(
				v1.VirtualMachineInstanceBackupVolumeInfo{VolumeName: "rootdisk", Type: string(backupv1.Full)},
			))
		})

		DescribeTable("should decide whether a backup is incremental from its base checkpoint",
			func(checkpoint *string, expected bool) {
				backupOptions.Incremental = checkpoint
				Expect(isIncrementalBackup(backupOptions)).To(Equal(expected))
			},
			Entry("no base checkpoint", nil, false),
			Entry("base checkpoint with no name", pointer.P(""), false),
			Entry("base checkpoint named", pointer.P("previous-checkpoint"), true),
		)

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

			_, _, volumesInfo := generateDomainBackup(disks, backupOptions, tempDir, nbdSocketPath, nil)

			Expect(volumesInfo).To(HaveLen(2))
			Expect(volumesInfo[0].VolumeName).To(Equal("rootdisk"))
			Expect(volumesInfo[1].VolumeName).To(Equal("datadisk"))
		})
	})

	Describe("SyncBackupResult", func() {
		const earlierWarning = "quiesce warning"
		completedJobStatsFlags := libvirt.DOMAIN_JOB_STATS_COMPLETED | libvirt.DOMAIN_JOB_STATS_KEEP_COMPLETED
		currentJobStatsFlags := libvirt.DomainGetJobStatsFlags(0)

		BeforeEach(func() {
			metadataCache.Backup.Store(api.BackupMetadata{
				Name:           backupName,
				StartTimestamp: backupOptions.BackupStartTime,
				JobStarted:     true,
				BackupMsg:      earlierWarning,
			})
		})

		DescribeTable("should record the result of an ended backup job", func(completedJob libvirt.DomainJobInfo, expectFailed bool, expectMsg string) {
			mockDomain.EXPECT().GetJobStats(currentJobStatsFlags).Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_NONE}, nil)
			mockDomain.EXPECT().GetJobStats(completedJobStatsFlags).Return(&completedJob, nil)

			SyncBackupResult(mockDomain, metadataCache)

			backupMetadata, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(backupMetadata.Completed).To(BeTrue())
			Expect(backupMetadata.EndTimestamp).ToNot(BeNil())
			Expect(backupMetadata.Failed).To(Equal(expectFailed))
			Expect(backupMetadata.BackupMsg).To(Equal(expectMsg))
		},
			Entry("as succeeded, keeping an earlier warning, when the job completed",
				libvirt.DomainJobInfo{Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP, Type: libvirt.DOMAIN_JOB_COMPLETED},
				false, earlierWarning),
			Entry("as failed with libvirt's error when the job failed",
				libvirt.DomainJobInfo{Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP, Type: libvirt.DOMAIN_JOB_FAILED, ErrorMessageSet: true, ErrorMessage: "I/O error"},
				true, backupJobFailedMsg+": I/O error"),
			Entry("as failed when the job was cancelled",
				libvirt.DomainJobInfo{Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP, Type: libvirt.DOMAIN_JOB_CANCELLED},
				true, backupAbortedMsg),
			Entry("as failed when libvirt no longer holds the result",
				libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_NONE},
				true, backupResultLostMsg),
		)

		It("should not record a result while the job is running", func() {
			mockDomain.EXPECT().GetJobStats(currentJobStatsFlags).Return(&libvirt.DomainJobInfo{
				Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
				Type:      libvirt.DOMAIN_JOB_UNBOUNDED,
			}, nil)

			SyncBackupResult(mockDomain, metadataCache)

			backupMetadata, _ := metadataCache.Backup.Load()
			Expect(backupMetadata.Completed).To(BeFalse())
		})

		It("should not record a result when libvirt cannot be queried", func() {
			mockDomain.EXPECT().GetJobStats(currentJobStatsFlags).Return(nil, libvirt.Error{Code: libvirt.ERR_INTERNAL_ERROR})

			Expect(SyncBackupResult(mockDomain, metadataCache)).ToNot(Succeed())

			backupMetadata, _ := metadataCache.Backup.Load()
			Expect(backupMetadata.Completed).To(BeFalse())
		})

		DescribeTable("should leave the metadata untouched without querying libvirt", func(backupMetadata api.BackupMetadata) {
			metadataCache.Backup.Store(backupMetadata)

			SyncBackupResult(mockDomain, metadataCache)

			got, exists := metadataCache.Backup.Load()
			Expect(exists).To(BeTrue())
			Expect(got).To(Equal(backupMetadata))
		},
			Entry("before the backup job started", api.BackupMetadata{Name: backupName}),
			Entry("once the result is recorded", api.BackupMetadata{Name: backupName, JobStarted: true, Completed: true, Failed: true, BackupMsg: backupAbortedMsg}),
		)
	})

	Describe("AbortVirtualMachineBackup", func() {
		Context("abort backup", func() {
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
				Entry("with no job before the backup job started", libvirt.DomainJobOperationType(0), libvirt.DOMAIN_JOB_NONE),
			)

			Context("when the backup job already ended", func() {
				noJob := &libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_NONE}

				BeforeEach(func() {
					metadataCache.Backup.Store(api.BackupMetadata{
						Name:           backupOptions.BackupName,
						StartTimestamp: backupOptions.BackupStartTime,
						JobStarted:     true,
					})
					mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
					mockDomain.EXPECT().Free().MaxTimes(1).Return(nil)
				})

				It("should record its result instead of aborting", func() {
					mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(noJob, nil).Times(2)
					mockDomain.EXPECT().GetJobStats(libvirt.DOMAIN_JOB_STATS_COMPLETED|libvirt.DOMAIN_JOB_STATS_KEEP_COMPLETED).Return(&libvirt.DomainJobInfo{
						Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
						Type:      libvirt.DOMAIN_JOB_COMPLETED,
					}, nil)

					Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(Succeed())

					md, _ := metadataCache.Backup.Load()
					Expect(md.Completed).To(BeTrue())
					Expect(md.Failed).To(BeFalse())
				})

				It("should return an error when its result cannot be read", func() {
					queryError := libvirt.Error{Code: libvirt.ERR_INTERNAL_ERROR}
					mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(noJob, nil)
					mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(nil, queryError)

					Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(MatchError(ContainSubstring(queryError.Error())))

					md, _ := metadataCache.Backup.Load()
					Expect(md.Completed).To(BeFalse())
				})
			})

			It("should return an error when the abort job call fails and restore AbortReason", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					StartTimestamp: backupOptions.BackupStartTime,
				}
				metadataCache.Backup.Store(backupMetadata)
				backupOptions.AbortReason = backupv1.BackupAbortCancel

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

				md, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(md.AbortReason).To(BeEmpty())
			})

			It("should keep the first AbortReason when a later Finalize abort is requested", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					Mode:           string(backupv1.PullMode),
					StartTimestamp: backupOptions.BackupStartTime,
					AbortReason:    string(backupv1.BackupAbortCancel),
				}
				metadataCache.Backup.Store(backupMetadata)
				backupOptions.AbortReason = backupv1.BackupAbortFinalize

				validJob := &libvirt.DomainJobInfo{
					Operation: libvirt.DOMAIN_JOB_OPERATION_BACKUP,
					Type:      libvirt.DOMAIN_JOB_UNBOUNDED,
				}
				mockConn.EXPECT().LookupDomainByName(gomock.Any()).Return(mockDomain, nil)
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(validJob, nil)
				mockDomain.EXPECT().AbortJob().Return(nil)
				mockDomain.EXPECT().Free().MaxTimes(1).Return(nil)

				Expect(manager.AbortVirtualMachineBackup(vmi, backupOptions)).To(Succeed())

				md, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(md.AbortReason).To(Equal(string(backupv1.BackupAbortCancel)))
			})

			It("should restore a previous AbortReason when a later AbortJob fails", func() {
				backupMetadata := api.BackupMetadata{
					Name:           backupOptions.BackupName,
					Mode:           string(backupv1.PullMode),
					StartTimestamp: backupOptions.BackupStartTime,
					AbortReason:    string(backupv1.BackupAbortCancel),
				}
				metadataCache.Backup.Store(backupMetadata)
				backupOptions.AbortReason = backupv1.BackupAbortFinalize

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
					MatchError(ContainSubstring(abortError.Error())),
				)

				md, exists := metadataCache.Backup.Load()
				Expect(exists).To(BeTrue())
				Expect(md.AbortReason).To(Equal(string(backupv1.BackupAbortCancel)))
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
})
