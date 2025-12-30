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

var _ = Describe("VMBackupController", func() {
	var (
		mockCtrl    *gomock.Controller
		virtClient  *kubecli.MockKubevirtClient
		kubevirtCli *kubevirtfake.Clientset
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		kubevirtCli = kubevirtfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(mockCtrl)
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
			err := ctrl.executeTracker(testNamespace + "/nonexistent")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return nil when tracker no longer needs redefinition", func() {
			// Tracker without redefinition flag set
			tracker := createTracker("tracker1", "test-vmi", false, false)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			err := ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error when VMI not found", func() {
			tracker := createTracker("tracker1", "test-vmi", true, true)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			// Don't add VMI to store

			err := ctrl.executeTracker(testNamespace + "/tracker1")
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

			err = ctrl.executeTracker(testNamespace + "/tracker1")
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

			err = ctrl.executeTracker(testNamespace + "/tracker1")
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

			err = ctrl.executeTracker(testNamespace + "/tracker1")
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

func createTracker(name, vmName string, hasCheckpoint bool, redefinitionRequired bool) *backupv1.VirtualMachineBackupTracker {
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
			CheckpointRedefinitionRequired: pointer.P(redefinitionRequired),
		}
	}
	return tracker
}
