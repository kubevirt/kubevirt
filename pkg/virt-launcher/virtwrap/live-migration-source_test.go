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
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
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
	})

	Context("Migration abort status", func() {
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
})
