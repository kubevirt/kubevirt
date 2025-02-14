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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/errors"
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
				"/usr/share/OVMF",
				ephemeralDiskCreatorMock,
				metadataCache,
				nil, //stop chn
				virtconfig.DefaultDiskVerificationMemoryLimitBytes,
				fakeCpuSetGetter,
				false, // image volume enabled
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
})
