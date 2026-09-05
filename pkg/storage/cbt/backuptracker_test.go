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
	"k8s.io/apimachinery/pkg/types"
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

	const (
		testNodeName = "test-node"
	)

	var (
		testPodUID  = types.UID("test-pod-uid-123")
		stalePodUID = types.UID("stale-pod-uid-456")
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		kubevirtCli = kubevirtfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(mockCtrl)
	})

	setActivePod := func(vmi *v1.VirtualMachineInstance, podUID types.UID) {
		vmi.Status.NodeName = testNodeName
		vmi.Status.ActivePods = map[types.UID]string{podUID: testNodeName}
	}

	Context("trackerHasCheckpoint", func() {
		DescribeTable("should correctly identify trackers with checkpoints",
			func(tracker *backupv1.VirtualMachineBackupTracker, expected bool) {
				Expect(trackerHasCheckpoint(tracker)).To(Equal(expected))
			},
			Entry("has checkpoint",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						LatestCheckpoint: &backupv1.BackupCheckpoint{
							Name: "checkpoint-1",
						},
					},
				},
				true,
			),
			Entry("tracker is nil",
				nil,
				false,
			),
			Entry("status is nil",
				&backupv1.VirtualMachineBackupTracker{
					Status: nil,
				},
				false,
			),
			Entry("checkpoint is nil",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						LatestCheckpoint: nil,
					},
				},
				false,
			),
			Entry("checkpoint name is empty",
				&backupv1.VirtualMachineBackupTracker{
					Status: &backupv1.VirtualMachineBackupTrackerStatus{
						LatestCheckpoint: &backupv1.BackupCheckpoint{
							Name: "",
						},
					},
				},
				false,
			),
		)
	})

	Context("ActivePodUID", func() {
		It("should return the UID of the pod matching NodeName", func() {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			setActivePod(vmi, testPodUID)
			Expect(ActivePodUID(vmi)).To(Equal(testPodUID))
		})

		It("should return empty when ActivePods is empty", func() {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			vmi.Status.NodeName = testNodeName
			Expect(ActivePodUID(vmi)).To(Equal(types.UID("")))
		})

		It("should return empty when no pod matches NodeName", func() {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			vmi.Status.NodeName = testNodeName
			vmi.Status.ActivePods = map[types.UID]string{
				testPodUID: "other-node",
			}
			Expect(ActivePodUID(vmi)).To(Equal(types.UID("")))
		})

		It("should return correct UID during migration with two active pods", func() {
			sourceNodeName := "source-node"
			targetNodeName := "target-node"
			sourcePodUID := types.UID("source-pod-uid")
			targetPodUID := types.UID("target-pod-uid")

			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			vmi.Status.ActivePods = map[types.UID]string{
				sourcePodUID: sourceNodeName,
				targetPodUID: targetNodeName,
			}

			vmi.Status.NodeName = sourceNodeName
			Expect(ActivePodUID(vmi)).To(Equal(sourcePodUID))

			vmi.Status.NodeName = targetNodeName
			Expect(ActivePodUID(vmi)).To(Equal(targetPodUID))
		})

		It("should return empty when NodeName is not set", func() {
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			vmi.Status.ActivePods = map[types.UID]string{testPodUID: testNodeName}
			Expect(ActivePodUID(vmi)).To(Equal(types.UID("")))
		})
	})

	Context("trackerNeedsRedefinitionForPod", func() {
		DescribeTable("with an active pod",
			func(hasCheckpoint bool, trackedPodUID *types.UID, expected bool) {
				tracker := createTracker("tracker1", "test-vmi", hasCheckpoint, trackedPodUID)
				vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
				setActivePod(vmi, testPodUID)
				Expect(trackerNeedsRedefinitionForPod(tracker, vmi)).To(Equal(expected))
			},
			Entry("nil LastTrackedPodUID", true, nil, true),
			Entry("stale LastTrackedPodUID", true, new(stalePodUID), true),
			Entry("matching LastTrackedPodUID", true, new(testPodUID), false),
			Entry("no checkpoint", false, nil, false),
		)

		It("does not need redefinition when VMI has no active pod", func() {
			tracker := createTracker("tracker1", "test-vmi", true, nil)
			vmi := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			Expect(trackerNeedsRedefinitionForPod(tracker, vmi)).To(BeFalse())
		})
	})

	Context("updateLastTrackedPodUID", func() {
		var (
			ctrl    *VMBackupController
			tracker *backupv1.VirtualMachineBackupTracker
		)

		BeforeEach(func() {
			ctrl = &VMBackupController{
				client: virtClient,
			}
		})

		It("should add LastTrackedPodUID when nil", func() {
			tracker = createTracker("tracker1", "test-vmi", true, nil)
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.updateLastTrackedPodUID(tracker, testPodUID)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).ToNot(BeNil())
			Expect(*updated.Status.LastTrackedPodUID).To(Equal(testPodUID))
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())
			Expect(updated.Status.LatestCheckpoint.Name).To(Equal("checkpoint-1"))
		})

		It("should replace LastTrackedPodUID when already set", func() {
			tracker = createTracker("tracker1", "test-vmi", true, pointer.P(stalePodUID))
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.updateLastTrackedPodUID(tracker, testPodUID)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).ToNot(BeNil())
			Expect(*updated.Status.LastTrackedPodUID).To(Equal(testPodUID))
		})
	})

	Context("clearCheckpointAndTrackedPod", func() {
		var (
			ctrl    *VMBackupController
			tracker *backupv1.VirtualMachineBackupTracker
		)

		BeforeEach(func() {
			ctrl = &VMBackupController{
				client: virtClient,
			}

			tracker = createTracker("tracker1", "test-vmi", true, pointer.P(testPodUID))
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should clear both checkpoint and LastTrackedPodUID", func() {
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err := ctrl.clearCheckpointAndTrackedPod(tracker)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).To(BeNil())
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

		It("should return nil when tracker has no checkpoint", func() {
			tracker := createTracker("tracker1", "test-vmi", false, nil)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			err := ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return nil when LastTrackedPodUID already matches", func() {
			tracker := createTracker("tracker1", "test-vmi", true, pointer.P(testPodUID))
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			setActivePod(testVMI, testPodUID)
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			err := ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return error when VMI not found", func() {
			tracker := createTracker("tracker1", "test-vmi", true, nil)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())

			err := ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should call RedefineCheckpoint and set LastTrackedPodUID on success", func() {
			tracker := createTracker("tracker1", "test-vmi", true, nil)
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			setActivePod(testVMI, testPodUID)
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(nil)
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).ToNot(BeNil())
			Expect(*updated.Status.LastTrackedPodUID).To(Equal(testPodUID))
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())
		})

		It("should redefine when LastTrackedPodUID is stale", func() {
			tracker := createTracker("tracker1", "test-vmi", true, pointer.P(stalePodUID))
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			setActivePod(testVMI, testPodUID)
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(nil)
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).ToNot(BeNil())
			Expect(*updated.Status.LastTrackedPodUID).To(Equal(testPodUID))
		})

		It("should clear checkpoint on permanent error (HTTP 422)", func() {
			tracker := createTracker("tracker1", "test-vmi", true, pointer.P(stalePodUID))
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			setActivePod(testVMI, testPodUID)
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			invalidErr := errors.New("unexpected return code 422 (422 Unprocessable Entity), message: RedefineCheckpoint failed: virError(Code=109, Domain=10, Message='checkpoint inconsistent: missing or broken bitmap')")

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(invalidErr)
			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).To(BeNil())
			Expect(updated.Status.LatestCheckpoint).To(BeNil())

			Eventually(recorder.Events).Should(Receive(ContainSubstring("CheckpointRedefinitionFailed")))
		})

		It("should return error for requeue on transient error (HTTP 503)", func() {
			tracker := createTracker("tracker1", "test-vmi", true, pointer.P(stalePodUID))
			Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			testVMI := libvmi.New(libvmi.WithNamespace(testNamespace), libvmi.WithName("test-vmi"))
			setActivePod(testVMI, testPodUID)
			Expect(vmiInformer.GetStore().Add(testVMI)).To(Succeed())

			transientErr := apierrors.NewServiceUnavailable("service temporarily unavailable")

			virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface)
			vmiInterface.EXPECT().RedefineCheckpoint(gomock.Any(), "test-vmi", gomock.Any()).Return(transientErr)

			err = ctrl.executeTracker(testNamespace + "/tracker1")
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsServiceUnavailable(err)).To(BeTrue())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LastTrackedPodUID).ToNot(BeNil())
			Expect(*updated.Status.LastTrackedPodUID).To(Equal(stalePodUID))
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())

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

			tracker = createTracker("tracker1", "test-vmi", true, pointer.P(stalePodUID))
			_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return transient error for requeue when error is ServiceUnavailable (HTTP 503)", func() {
			transientErr := apierrors.NewServiceUnavailable("service temporarily unavailable")

			err := ctrl.handleRedefinitionError(tracker, transientErr)
			Expect(err).To(HaveOccurred())
			Expect(apierrors.IsServiceUnavailable(err)).To(BeTrue())

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())

			Consistently(recorder.Events).ShouldNot(Receive())
		})

		It("should return generic error for requeue when error is not a known API error", func() {
			genericErr := errors.New("some transient network error")

			err := ctrl.handleRedefinitionError(tracker, genericErr)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("some transient network error"))

			updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.LatestCheckpoint).ToNot(BeNil())

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
			Expect(updated.Status.LastTrackedPodUID).To(BeNil())

			Eventually(recorder.Events).Should(Receive(ContainSubstring("CheckpointRedefinitionFailed")))
		})
	})
})

func createTracker(name, vmName string, hasCheckpoint bool, syncedPodUID *types.UID) *backupv1.VirtualMachineBackupTracker {
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
			LastTrackedPodUID: syncedPodUID,
		}
	}
	return tracker
}
