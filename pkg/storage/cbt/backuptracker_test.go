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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/controller"
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

	Context("handleTrackerDeletion", func() {
		var (
			ctrl           *VMBackupController
			backupInformer cache.SharedIndexInformer
		)

		BeforeEach(func() {
			backupInformer, _ = testutils.NewFakeInformerWithIndexersFor(
				&backupv1.VirtualMachineBackup{},
				controller.GetVirtualMachineBackupInformerIndexers(),
			)

			ctrl = &VMBackupController{
				client:         virtClient,
				backupInformer: backupInformer,
			}
		})

		It("should remove finalizer when no backups exist", func() {
			tracker := createTracker("tracker1", "test-vmi")
			tracker.DeletionTimestamp = new(metav1.Now())
			tracker.Finalizers = []string{backupv1.VirtualMachineBackupTrackerFinalizer}
			_, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace))

			err = ctrl.handleTrackerDeletion(tracker)
			Expect(err).ToNot(HaveOccurred())

			updated, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace).Get(
				context.Background(), "tracker1", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Finalizers).To(BeEmpty())
		})

		DescribeTable("should remove finalizer only when all backups are terminal",
			func(backup *backupv1.VirtualMachineBackup, shouldRemove bool) {
				tracker := createTracker("tracker1", "test-vmi")
				tracker.DeletionTimestamp = new(metav1.Now())
				tracker.Finalizers = []string{backupv1.VirtualMachineBackupTrackerFinalizer}
				_, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace).Create(
					context.Background(), tracker, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(backupInformer.GetStore().Add(backup)).To(Succeed())

				if shouldRemove {
					virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
						Return(kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace))
				}

				err = ctrl.handleTrackerDeletion(tracker)
				if shouldRemove {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(HaveOccurred())
				}

				updated, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace).Get(
					context.Background(), "tracker1", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if shouldRemove {
					Expect(updated.Finalizers).To(BeEmpty())
				} else {
					Expect(updated.Finalizers).To(ContainElement(backupv1.VirtualMachineBackupTrackerFinalizer))
				}
			},
			Entry("active backup blocks removal",
				&backupv1.VirtualMachineBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "active-backup",
						Namespace: testNamespace,
					},
					Spec: backupv1.VirtualMachineBackupSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: new(backupv1.SchemeGroupVersion.Group),
							Kind:     backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind,
							Name:     "tracker1",
						},
					},
				}, false),
			Entry("completed backup allows removal",
				&backupv1.VirtualMachineBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "completed-backup",
						Namespace: testNamespace,
					},
					Spec: backupv1.VirtualMachineBackupSpec{
						Source: corev1.TypedLocalObjectReference{
							APIGroup: new(backupv1.SchemeGroupVersion.Group),
							Kind:     backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind,
							Name:     "tracker1",
						},
					},
					Status: &backupv1.VirtualMachineBackupStatus{
						Conditions: []metav1.Condition{
							{
								Type:   string(backupv1.ConditionComplete),
								Status: metav1.ConditionTrue,
							},
						},
					},
				}, true),
		)
	})

	Context("executeTracker", func() {
		It("should return nil when tracker does not exist", func() {
			trackerInformer, _ := testutils.NewFakeInformerWithIndexersFor(
				&backupv1.VirtualMachineBackupTracker{},
				controller.GetVirtualMachineBackupTrackerInformerIndexers(),
			)
			ctrl := &VMBackupController{backupTrackerInformer: trackerInformer}

			Expect(ctrl.executeTracker(testNamespace + "/nonexistent")).To(Succeed())
		})
	})
})

func createTracker(name, vmName string) *backupv1.VirtualMachineBackupTracker {
	return &backupv1.VirtualMachineBackupTracker{
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
}
