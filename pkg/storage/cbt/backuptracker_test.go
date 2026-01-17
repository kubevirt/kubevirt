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

package cbt

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Checkpoints", func() {
	var (
		mockCtrl    *gomock.Controller
		vmi         *v1.VirtualMachineInstance
		virtClient  *kubecli.MockKubevirtClient
		kubevirtCli *kubevirtfake.Clientset
		informer    cache.SharedIndexInformer
	)

	const testNamespace = "test-ns"

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		vmi = libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
		kubevirtCli = kubevirtfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(mockCtrl)
		informer, _ = testutils.NewFakeInformerWithIndexersFor(&backupv1.VirtualMachineBackupTracker{}, controller.GetVirtualMachineBackupTrackerInformerIndexers())
	})

	createTracker := func(name, vmName string, hasCheckpoint bool, alreadyMarked bool) *backupv1.VirtualMachineBackupTracker {
		tracker := &backupv1.VirtualMachineBackupTracker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: backupv1.VirtualMachineBackupTrackerSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: pointer.P("kubevirt.io"),
					Kind:     "VirtualMachine",
					Name:     vmName,
				},
			},
		}
		if hasCheckpoint {
			tracker.Status = &backupv1.VirtualMachineBackupTrackerStatus{
				LatestCheckpoint: &backupv1.BackupCheckpoint{
					Name: "checkpoint-1",
				},
				CheckpointRedefinitionRequired: pointer.P(alreadyMarked),
			}
		}
		return tracker
	}

	addTracker := func(tracker *backupv1.VirtualMachineBackupTracker) {
		_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
			context.Background(), tracker, metav1.CreateOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		ExpectWithOffset(1, informer.GetStore().Add(tracker)).To(Succeed())
	}

	Context("markTrackersForRedefinition", func() {
		It("should return error when informer is nil", func() {
			err := markTrackersForRedefinition(vmi, nil, virtClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tracker informer or clientset is nil"))
		})

		It("should return error when clientset is nil", func() {
			err := markTrackersForRedefinition(vmi, informer, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("tracker informer or clientset is nil"))
		})

		DescribeTable("should skip trackers that don't need redefinition",
			func(modifyTracker func(*backupv1.VirtualMachineBackupTracker)) {
				if modifyTracker != nil {
					tracker := createTracker("tracker1", "test-vmi", false, false)
					modifyTracker(tracker)
					addTracker(tracker)
				}

				err := markTrackersForRedefinition(vmi, informer, virtClient)
				Expect(err).ToNot(HaveOccurred())
			},
			Entry("when no trackers exist", nil),
			Entry("without status", func(t *backupv1.VirtualMachineBackupTracker) {
				t.Status = nil
			}),
			Entry("without latest checkpoint", func(t *backupv1.VirtualMachineBackupTracker) {
				t.Status = &backupv1.VirtualMachineBackupTrackerStatus{
					LatestCheckpoint: nil,
				}
			}),
			Entry("already marked for redefinition", func(t *backupv1.VirtualMachineBackupTracker) {
				t.Status = &backupv1.VirtualMachineBackupTrackerStatus{
					LatestCheckpoint: &backupv1.BackupCheckpoint{
						Name: "checkpoint-1",
					},
					CheckpointRedefinitionRequired: pointer.P(true),
				}
			}),
		)

		It("should mark tracker with checkpoint for redefinition", func() {
			tracker := createTracker("tracker1", "test-vmi", true, false)
			addTracker(tracker)

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err := markTrackersForRedefinition(vmi, informer, virtClient)
			Expect(err).ToNot(HaveOccurred())

			// Verify tracker was marked
			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated.Status.CheckpointRedefinitionRequired).To(BeTrue())
		})

		It("should mark multiple trackers for redefinition", func() {
			tracker1 := createTracker("tracker1", "test-vmi", true, false)
			tracker2 := createTracker("tracker2", "test-vmi", true, false)
			addTracker(tracker1)
			addTracker(tracker2)

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace)).Times(2)

			err := markTrackersForRedefinition(vmi, informer, virtClient)
			Expect(err).ToNot(HaveOccurred())

			// Verify both trackers were marked
			updated1, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated1.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated1.Status.CheckpointRedefinitionRequired).To(BeTrue())

			updated2, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker2", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated2.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated2.Status.CheckpointRedefinitionRequired).To(BeTrue())
		})

		It("should only mark trackers that need redefinition", func() {
			// tracker1 has checkpoint and needs marking
			tracker1 := createTracker("tracker1", "test-vmi", true, false)
			// tracker2 is already marked, should be skipped
			tracker2 := createTracker("tracker2", "test-vmi", true, true)
			// tracker3 has no checkpoint, should be skipped
			tracker3 := createTracker("tracker3", "test-vmi", false, false)
			addTracker(tracker1)
			addTracker(tracker2)
			addTracker(tracker3)

			// Only one call expected since tracker2 and tracker3 should be skipped
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace)).Times(1)

			err := markTrackersForRedefinition(vmi, informer, virtClient)
			Expect(err).ToNot(HaveOccurred())

			// Verify tracker1 was marked
			updated1, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated1.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated1.Status.CheckpointRedefinitionRequired).To(BeTrue())

			// Verify tracker2 was not updated (still has its original state)
			updated2, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker2", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated2.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated2.Status.CheckpointRedefinitionRequired).To(BeTrue())

			// Verify tracker3 has no status (was skipped)
			updated3, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker3", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated3.Status).To(BeNil())
		})
	})

	Context("getBackupTrackersForVMI", func() {
		It("should return empty list when no trackers exist", func() {
			trackers := getBackupTrackersForVMI(vmi, informer)
			Expect(trackers).To(BeEmpty())
		})

		It("should return trackers matching VMI namespace/name", func() {
			tracker := createTracker("tracker1", "test-vmi", false, false)
			Expect(informer.GetStore().Add(tracker)).To(Succeed())

			trackers := getBackupTrackersForVMI(vmi, informer)
			Expect(trackers).To(HaveLen(1))
			Expect(trackers[0].Name).To(Equal("tracker1"))
		})

		It("should not return trackers for different VMI", func() {
			tracker := createTracker("tracker1", "different-vmi", false, false)
			Expect(informer.GetStore().Add(tracker)).To(Succeed())

			trackers := getBackupTrackersForVMI(vmi, informer)
			Expect(trackers).To(BeEmpty())
		})
	})

	Context("trackerNeedsCheckpointRedefinition", func() {
		DescribeTable("should correctly identify trackers needing redefinition",
			func(tracker *backupv1.VirtualMachineBackupTracker, expected bool) {
				Expect(trackerNeedsCheckpointRedefinition(tracker)).To(Equal(expected))
			},
			Entry("needs redefinition when all conditions met",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						CheckpointRedefinitionRequired: pointer.P(true),
						LatestCheckpoint: &backupv1.BackupCheckpoint{
							Name: "checkpoint-1",
						},
					},
				},
				true,
			),
			Entry("does not need redefinition when tracker is nil",
				nil,
				false,
			),
			Entry("does not need redefinition when status is nil",
				&backupv1.VirtualMachineBackupTracker{
					Status: nil,
				},
				false,
			),
			Entry("does not need redefinition when flag is nil",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						CheckpointRedefinitionRequired: nil,
						LatestCheckpoint: &backupv1.BackupCheckpoint{
							Name: "checkpoint-1",
						},
					},
				},
				false,
			),
			Entry("does not need redefinition when flag is false",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						CheckpointRedefinitionRequired: pointer.P(false),
						LatestCheckpoint: &backupv1.BackupCheckpoint{
							Name: "checkpoint-1",
						},
					},
				},
				false,
			),
			Entry("does not need redefinition when checkpoint is nil",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						CheckpointRedefinitionRequired: pointer.P(true),
						LatestCheckpoint:               nil,
					},
				},
				false,
			),
			Entry("does not need redefinition when checkpoint name is empty",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						CheckpointRedefinitionRequired: pointer.P(true),
						LatestCheckpoint: &backupv1.BackupCheckpoint{
							Name: "",
						},
					},
				},
				false,
			),
		)
	})

	Context("clearRedefinitionFlag", func() {
		var (
			ctrl    *VMBackupController
			tracker *backupv1.VirtualMachineBackupTracker
		)

		BeforeEach(func() {
			ctrl = &VMBackupController{
				client: virtClient,
			}

			tracker = createTracker("tracker1", "test-vmi", true, true)
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should clear only the redefinition flag", func() {
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err := ctrl.clearRedefinitionFlag(tracker)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Flag should be cleared
			Expect(updated.Status.CheckpointRedefinitionRequired).To(BeNil())
			// Checkpoint should still exist
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())
			Expect(updated.Status.LatestCheckpoint.Name).To(Equal("checkpoint-1"))
		})
	})

	Context("clearCheckpointAndFlag", func() {
		var (
			ctrl    *VMBackupController
			tracker *backupv1.VirtualMachineBackupTracker
		)

		BeforeEach(func() {
			ctrl = &VMBackupController{
				client: virtClient,
			}

			tracker = createTracker("tracker1", "test-vmi", true, true)
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should clear both checkpoint and redefinition flag", func() {
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err := ctrl.clearCheckpointAndFlag(tracker)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.CheckpointRedefinitionRequired).To(BeNil())
			Expect(updated.Status.LatestCheckpoint).To(BeNil())
		})
	})

	Context("executeTracker", func() {
		var (
			ctrl            *VMBackupController
			trackerInformer cache.SharedIndexInformer
			vmiInformer     cache.SharedIndexInformer
			recorder        *record.FakeRecorder
			vmiInterface    *kubecli.MockVirtualMachineInstanceInterface
		)

		BeforeEach(func() {
			trackerInformer, _ = testutils.NewFakeInformerWithIndexersFor(
				&backupv1.VirtualMachineBackupTracker{},
				controller.GetVirtualMachineBackupTrackerInformerIndexers(),
			)
			vmiInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(mockCtrl)

			ctrl = &VMBackupController{
				client:                virtClient,
				backupTrackerInformer: trackerInformer,
				vmiStore:              vmiInformer.GetStore(),
				recorder:              recorder,
			}
		})

		It("should return nil when tracker does not exist", func() {
			err := ctrl.executeTracker("test-ns/nonexistent")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return nil when tracker no longer needs redefinition", func() {
			// Tracker without redefinition flag set
			tracker := createTracker("tracker1", "test-vmi", false, false)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			err := ctrl.executeTracker("test-ns/tracker1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error when VMI not found", func() {
			tracker := createTracker("tracker1", "test-vmi", true, true)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			// Don't add VMI to store

			err := ctrl.executeTracker("test-ns/tracker1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should call RedefineCheckpoint and clear flag on success", func() {
			tracker := createTracker("tracker1", "test-vmi", true, true)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(nil)
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.executeTracker("test-ns/tracker1")
			Expect(err).ToNot(HaveOccurred())

			// Verify flag was cleared
			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.CheckpointRedefinitionRequired).To(BeNil())
			// Checkpoint should still exist
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())
		})

		It("should clear checkpoint on permanent error (HTTP 422)", func() {
			tracker := createTracker("tracker1", "test-vmi", true, true)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			invalidErr := errors.New("unexpected return code 422 (422 Unprocessable Entity), message: RedefineCheckpoint failed: virError(Code=109, Domain=10, Message='checkpoint inconsistent: missing or broken bitmap')")

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(invalidErr)
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.executeTracker("test-ns/tracker1")
			Expect(err).ToNot(HaveOccurred())

			// Verify both checkpoint and flag were cleared
			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.CheckpointRedefinitionRequired).To(BeNil())
			Expect(updated.Status.LatestCheckpoint).To(BeNil())

			// Verify event was emitted
			Eventually(recorder.Events).Should(Receive(ContainSubstring("CheckpointRedefinitionFailed")))
		})

		It("should return error for requeue on transient error (HTTP 503)", func() {
			tracker := createTracker("tracker1", "test-vmi", true, true)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			transientErr := apierrors.NewServiceUnavailable("service temporarily unavailable")

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(transientErr)

			err = ctrl.executeTracker("test-ns/tracker1")
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsServiceUnavailable(err)).To(BeTrue())

			// Verify tracker was NOT modified
			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.CheckpointRedefinitionRequired).ToNot(BeNil())
			Expect(*updated.Status.CheckpointRedefinitionRequired).To(BeTrue())
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())

			// Verify no event was emitted
			Consistently(recorder.Events).ShouldNot(Receive())
		})
	})

	Context("handleRedefinitionError", func() {
		var (
			ctrl     *VMBackupController
			recorder *record.FakeRecorder
			tracker  *backupv1.VirtualMachineBackupTracker
		)

		BeforeEach(func() {
			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			ctrl = &VMBackupController{
				client:   virtClient,
				recorder: recorder,
			}

			tracker = createTracker("tracker1", "test-vmi", true, true)
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return transient error for requeue when error is ServiceUnavailable (HTTP 503)", func() {
			transientErr := apierrors.NewServiceUnavailable("service temporarily unavailable")

			err := ctrl.handleRedefinitionError(tracker, transientErr)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsServiceUnavailable(err)).To(BeTrue())

			// Verify checkpoint was NOT cleared (tracker unchanged in fake client)
			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())

			// Verify no event was emitted
			Consistently(recorder.Events).ShouldNot(Receive())
		})

		It("should return generic error for requeue when error is not a known API error", func() {
			genericErr := errors.New("some transient network error")

			err := ctrl.handleRedefinitionError(tracker, genericErr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("some transient network error"))

			// Verify checkpoint was NOT cleared
			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())

			// Verify no event was emitted
			Consistently(recorder.Events).ShouldNot(Receive())
		})

		It("should clear checkpoint when checkpoint is invalid/corrupt", func() {
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))
			virtHandlerErr := errors.New("unexpected return code 422 (422 Unprocessable Entity), message: RedefineCheckpoint failed: virError(Code=109, Domain=10, Message='checkpoint inconsistent: missing or broken bitmap')")

			err := ctrl.handleRedefinitionError(tracker, virtHandlerErr)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LatestCheckpoint).To(BeNil())
			Expect(updated.Status.CheckpointRedefinitionRequired).To(BeNil())

			Eventually(recorder.Events).Should(Receive(ContainSubstring("CheckpointRedefinitionFailed")))
		})
	})
})
