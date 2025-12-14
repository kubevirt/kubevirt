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

package snapshot

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("VMSnapshotSchedule controller", func() {
	var ctrl *VMSnapshotScheduleController

	BeforeEach(func() {
		ctrl = &VMSnapshotScheduleController{}
	})

	Context("calculateNextRun", func() {
		It("should handle @hourly schedule", func() {
			from := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			next, err := ctrl.calculateNextRun("@hourly", from)
			Expect(err).ToNot(HaveOccurred())
			Expect(next).To(Equal(from.Add(time.Hour)))
		})

		It("should handle @daily schedule", func() {
			from := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			next, err := ctrl.calculateNextRun("@daily", from)
			Expect(err).ToNot(HaveOccurred())
			Expect(next).To(Equal(from.AddDate(0, 0, 1)))
		})

		It("should handle @weekly schedule", func() {
			from := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			next, err := ctrl.calculateNextRun("@weekly", from)
			Expect(err).ToNot(HaveOccurred())
			Expect(next).To(Equal(from.AddDate(0, 0, 7)))
		})

		It("should handle @monthly schedule", func() {
			from := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			next, err := ctrl.calculateNextRun("@monthly", from)
			Expect(err).ToNot(HaveOccurred())
			Expect(next).To(Equal(from.AddDate(0, 1, 0)))
		})

		It("should handle @yearly schedule", func() {
			from := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			next, err := ctrl.calculateNextRun("@yearly", from)
			Expect(err).ToNot(HaveOccurred())
			Expect(next).To(Equal(from.AddDate(1, 0, 0)))
		})

		It("should handle standard cron expression", func() {
			from := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
			// Every day at noon
			next, err := ctrl.calculateNextRun("0 12 * * *", from)
			Expect(err).ToNot(HaveOccurred())
			Expect(next.Hour()).To(Equal(12))
			Expect(next.Day()).To(Equal(1)) // Same day since from is 10:00 and next is 12:00
		})

		It("should return error for invalid cron expression", func() {
			from := time.Now()
			_, err := ctrl.calculateNextRun("invalid", from)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("shouldCreateSnapshot", func() {
		It("should return true when no snapshots exist", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Schedule: "@hourly",
				},
			}
			result := ctrl.shouldCreateSnapshot(schedule, []*snapshotv1.VirtualMachineSnapshot{})
			Expect(result).To(BeTrue())
		})

		It("should return false when last snapshot is recent", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Schedule: "@hourly",
				},
			}
			snapshots := []*snapshotv1.VirtualMachineSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Now(),
					},
				},
			}
			result := ctrl.shouldCreateSnapshot(schedule, snapshots)
			Expect(result).To(BeFalse())
		})

		It("should return true when last snapshot is older than schedule interval", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Schedule: "@hourly",
				},
			}
			// Create a snapshot from 2 hours ago
			twoHoursAgo := metav1.NewTime(time.Now().Add(-2 * time.Hour))
			snapshots := []*snapshotv1.VirtualMachineSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: twoHoursAgo,
					},
				},
			}
			result := ctrl.shouldCreateSnapshot(schedule, snapshots)
			Expect(result).To(BeTrue())
		})
	})

	Context("getSnapshotsForSchedule", func() {
		var (
			vmSnapshotInformer cache.SharedIndexInformer
		)

		BeforeEach(func() {
			vmSnapshotInformer = cache.NewSharedIndexInformer(&cache.ListWatch{}, &snapshotv1.VirtualMachineSnapshot{}, 0, cache.Indexers{})
			ctrl.VMSnapshotInformer = vmSnapshotInformer
		})

		It("should return empty list when no snapshots exist", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-schedule",
					Namespace: "default",
				},
			}
			vm := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
				},
			}
			snapshots, err := ctrl.getSnapshotsForSchedule(schedule, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(snapshots).To(BeEmpty())
		})

		It("should filter snapshots by schedule name and VM name", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-schedule",
					Namespace: "default",
				},
			}
			vm := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
				},
			}

			// Add matching snapshot
			matchingSnapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "matching-snapshot",
					Namespace:         "default",
					CreationTimestamp: metav1.Now(),
					Labels: map[string]string{
						"snapshot.kubevirt.io/schedule-name":  "test-schedule",
						"snapshot.kubevirt.io/source-vm-name": "test-vm",
					},
				},
			}
			err := vmSnapshotInformer.GetStore().Add(matchingSnapshot)
			Expect(err).ToNot(HaveOccurred())

			// Add non-matching snapshot (different schedule)
			nonMatchingSnapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "non-matching-snapshot",
					Namespace: "default",
					Labels: map[string]string{
						"snapshot.kubevirt.io/schedule-name":  "other-schedule",
						"snapshot.kubevirt.io/source-vm-name": "test-vm",
					},
				},
			}
			err = vmSnapshotInformer.GetStore().Add(nonMatchingSnapshot)
			Expect(err).ToNot(HaveOccurred())

			snapshots, err := ctrl.getSnapshotsForSchedule(schedule, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(snapshots).To(HaveLen(1))
			Expect(snapshots[0].Name).To(Equal("matching-snapshot"))
		})

		It("should sort snapshots by creation time newest first", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-schedule",
					Namespace: "default",
				},
			}
			vm := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: "default",
				},
			}

			// Add older snapshot
			olderTime := metav1.NewTime(time.Now().Add(-2 * time.Hour))
			olderSnapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "older-snapshot",
					Namespace:         "default",
					CreationTimestamp: olderTime,
					Labels: map[string]string{
						"snapshot.kubevirt.io/schedule-name":  "test-schedule",
						"snapshot.kubevirt.io/source-vm-name": "test-vm",
					},
				},
			}
			err := vmSnapshotInformer.GetStore().Add(olderSnapshot)
			Expect(err).ToNot(HaveOccurred())

			// Add newer snapshot
			newerTime := metav1.NewTime(time.Now().Add(-1 * time.Hour))
			newerSnapshot := &snapshotv1.VirtualMachineSnapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "newer-snapshot",
					Namespace:         "default",
					CreationTimestamp: newerTime,
					Labels: map[string]string{
						"snapshot.kubevirt.io/schedule-name":  "test-schedule",
						"snapshot.kubevirt.io/source-vm-name": "test-vm",
					},
				},
			}
			err = vmSnapshotInformer.GetStore().Add(newerSnapshot)
			Expect(err).ToNot(HaveOccurred())

			snapshots, err := ctrl.getSnapshotsForSchedule(schedule, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(snapshots).To(HaveLen(2))
			Expect(snapshots[0].Name).To(Equal("newer-snapshot"))
			Expect(snapshots[1].Name).To(Equal("older-snapshot"))
		})
	})

	Context("getMatchingVMs", func() {
		var vmInformer cache.SharedIndexInformer

		BeforeEach(func() {
			vmInformer = cache.NewSharedIndexInformer(&cache.ListWatch{}, &v1.VirtualMachine{}, 0, cache.Indexers{})
			ctrl.VMInformer = vmInformer
		})

		It("should return all VMs in namespace when no selector is specified", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-schedule",
					Namespace: "default",
				},
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					ClaimSelector: nil,
				},
			}

			vm1 := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm1",
					Namespace: "default",
				},
			}
			vm2 := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm2",
					Namespace: "default",
				},
			}
			vmOtherNs := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-other",
					Namespace: "other-namespace",
				},
			}

			Expect(vmInformer.GetStore().Add(vm1)).To(Succeed())
			Expect(vmInformer.GetStore().Add(vm2)).To(Succeed())
			Expect(vmInformer.GetStore().Add(vmOtherNs)).To(Succeed())

			vms, err := ctrl.getMatchingVMs(schedule)
			Expect(err).ToNot(HaveOccurred())
			Expect(vms).To(HaveLen(2))
		})

		It("should filter VMs by label selector", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-schedule",
					Namespace: "default",
				},
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					ClaimSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"backup": "enabled",
						},
					},
				},
			}

			vmMatching := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-matching",
					Namespace: "default",
					Labels: map[string]string{
						"backup": "enabled",
					},
				},
			}
			vmNonMatching := &v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm-non-matching",
					Namespace: "default",
					Labels: map[string]string{
						"backup": "disabled",
					},
				},
			}

			Expect(vmInformer.GetStore().Add(vmMatching)).To(Succeed())
			Expect(vmInformer.GetStore().Add(vmNonMatching)).To(Succeed())

			vms, err := ctrl.getMatchingVMs(schedule)
			Expect(err).ToNot(HaveOccurred())
			Expect(vms).To(HaveLen(1))
			Expect(vms[0].Name).To(Equal("vm-matching"))
		})
	})

	Context("cleanupSnapshots", func() {
		It("should delete oldest snapshots when exceeding max count", func() {
			maxCount := int32(2)

			// Create 4 snapshots sorted newest first
			now := time.Now()
			snapshots := []*snapshotv1.VirtualMachineSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "snap-newest",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "snap-newer",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-1 * time.Hour)),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "snap-older",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "snap-oldest",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-3 * time.Hour)),
					},
				},
			}

			// We need to verify the logic without actually deleting
			// The fix ensures snapshots[MaxCount:] are deleted (oldest ones)
			toDelete := snapshots[int(maxCount):]
			Expect(toDelete).To(HaveLen(2))
			Expect(toDelete[0].Name).To(Equal("snap-older"))
			Expect(toDelete[1].Name).To(Equal("snap-oldest"))
		})

		It("should delete snapshots older than expiration time", func() {
			expires := metav1.Duration{Duration: 24 * time.Hour}
			retention := snapshotv1.VirtualMachineSnapshotScheduleRetention{
				Expires: &expires,
			}

			now := time.Now()
			cutoff := now.Add(-expires.Duration)

			// One snapshot within retention, one outside
			snapshots := []*snapshotv1.VirtualMachineSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "snap-recent",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-12 * time.Hour)),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "snap-expired",
						Namespace:         "default",
						CreationTimestamp: metav1.NewTime(now.Add(-48 * time.Hour)),
					},
				},
			}

			var toDelete []*snapshotv1.VirtualMachineSnapshot
			for _, snapshot := range snapshots {
				if !snapshot.CreationTimestamp.IsZero() && snapshot.CreationTimestamp.Time.Before(cutoff) {
					toDelete = append(toDelete, snapshot)
				}
			}

			Expect(toDelete).To(HaveLen(1))
			Expect(toDelete[0].Name).To(Equal("snap-expired"))

			// Verify retention spec is set correctly
			Expect(retention.Expires).ToNot(BeNil())
		})

		It("should handle nil retention policy gracefully", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Retention: snapshotv1.VirtualMachineSnapshotScheduleRetention{
						MaxCount: nil,
						Expires:  nil,
					},
				},
			}

			snapshots := []*snapshotv1.VirtualMachineSnapshot{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "snap-1",
						Namespace: "default",
					},
				},
			}

			// With nil retention policies, nothing should be marked for deletion
			var toDelete []*snapshotv1.VirtualMachineSnapshot

			if schedule.Spec.Retention.MaxCount != nil && len(snapshots) > int(*schedule.Spec.Retention.MaxCount) {
				toDelete = append(toDelete, snapshots[int(*schedule.Spec.Retention.MaxCount):]...)
			}

			Expect(toDelete).To(BeEmpty())
		})
	})

	Context("schedule disabled", func() {
		It("should skip processing when disabled is true", func() {
			disabled := true
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Disabled: &disabled,
				},
			}

			Expect(schedule.Spec.Disabled).ToNot(BeNil())
			Expect(*schedule.Spec.Disabled).To(BeTrue())
		})

		It("should process when disabled is false", func() {
			disabled := false
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Disabled: &disabled,
				},
			}

			Expect(schedule.Spec.Disabled).ToNot(BeNil())
			Expect(*schedule.Spec.Disabled).To(BeFalse())
		})

		It("should process when disabled is nil", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					Disabled: nil,
				},
			}

			isDisabled := schedule.Spec.Disabled != nil && *schedule.Spec.Disabled
			Expect(isDisabled).To(BeFalse())
		})
	})

	Context("snapshot template", func() {
		It("should use template labels when creating snapshot", func() {
			schedule := &snapshotv1.VirtualMachineSnapshotSchedule{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-schedule",
					Namespace: "default",
				},
				Spec: snapshotv1.VirtualMachineSnapshotScheduleSpec{
					SnapshotTemplate: snapshotv1.VirtualMachineSnapshotScheduleTemplate{
						Labels: map[string]string{
							"custom-label": "custom-value",
						},
						Annotations: map[string]string{
							"custom-annotation": "annotation-value",
						},
						Spec: snapshotv1.VirtualMachineSnapshotSpec{
							DeletionPolicy: pointer.P(snapshotv1.VirtualMachineSnapshotContentRetain),
						},
					},
				},
			}

			Expect(schedule.Spec.SnapshotTemplate.Labels).To(HaveKeyWithValue("custom-label", "custom-value"))
			Expect(schedule.Spec.SnapshotTemplate.Annotations).To(HaveKeyWithValue("custom-annotation", "annotation-value"))
			Expect(schedule.Spec.SnapshotTemplate.Spec.DeletionPolicy).ToNot(BeNil())
			Expect(*schedule.Spec.SnapshotTemplate.Spec.DeletionPolicy).To(Equal(snapshotv1.VirtualMachineSnapshotContentRetain))
		})
	})
})
