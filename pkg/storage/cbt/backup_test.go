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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	testNamespace     = "default"
	vmName            = "test-vm"
	backupName        = "test-backup"
	backupTrackerName = "test-backup-tracker"
	checkpointName    = "test-checkpoint"
	pvcName           = "backup-target"
)

var (
	vmUID     types.UID = "vm-uid"
	backupUID types.UID = "backup-uid"
)

func newCondition(condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    condType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

var _ = Describe("Backup Controller", func() {
	var (
		ctrl                  *gomock.Controller
		virtClient            *kubecli.MockKubevirtClient
		vmInterface           *kubecli.MockVirtualMachineInterface
		vmiInterface          *kubecli.MockVirtualMachineInstanceInterface
		backupInformer        cache.SharedIndexInformer
		backupSource          *framework.FakeControllerSource
		backupTrackerInformer cache.SharedIndexInformer
		vmInformer            cache.SharedIndexInformer
		vmiInformer           cache.SharedIndexInformer
		pvcInformer           cache.SharedIndexInformer
		vmExportInformer      cache.SharedIndexInformer
		controller            *VMBackupController
		recorder              *record.FakeRecorder
		mockBackupQueue       *testutils.MockWorkQueue[string]

		kubevirtClient *kubevirtfake.Clientset
		k8sClient      *fake.Clientset
	)

	createBackup := func(name, vmName, pvcName string, mode backupv1.BackupMode) *backupv1.VirtualMachineBackup {
		return &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
				UID:       backupUID,
			},
			Spec: backupv1.VirtualMachineBackupSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: pointer.P("kubevirt.io"),
					Kind:     "VirtualMachine",
					Name:     vmName,
				},
				PvcName: pointer.P(pvcName),
				Mode:    pointer.P(mode),
			},
		}
	}

	createBackupWithTracker := func(name, vmName, pvcName string) *backupv1.VirtualMachineBackup {
		backup := createBackup(name, vmName, pvcName, backupv1.PushMode)
		backup.Spec.Source = corev1.TypedLocalObjectReference{
			APIGroup: pointer.P("backup.kubevirt.io"),
			Kind:     backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind,
			Name:     backupTrackerName,
		}
		return backup
	}

	createVMI := func() *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmName,
				Namespace: testNamespace,
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: []v1.Volume{
					{
						Name: "disk0",
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-disk",
								},
							},
						},
					},
				},
			},
			Status: v1.VirtualMachineInstanceStatus{
				ChangedBlockTracking: &v1.ChangedBlockTrackingStatus{
					State: v1.ChangedBlockTrackingEnabled,
				},
				Phase: v1.Running,
			},
		}
	}

	createInitializedVMI := func() *v1.VirtualMachineInstance {
		vmi := createVMI()
		volumeName := backupTargetVolumeName(backupName)
		vmi.Spec.UtilityVolumes = []v1.UtilityVolume{
			{
				Name: volumeName,
				PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
				Type: pointer.P(v1.Backup),
			},
		}
		vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
			BackupName:     backupName,
			Completed:      false,
			CheckpointName: pointer.P(checkpointName),
		}
		vmi.Status.VolumeStatus = []v1.VolumeStatus{
			{
				Name:          volumeName,
				Phase:         v1.HotplugVolumeMounted,
				HotplugVolume: &v1.HotplugVolumeStatus{},
			},
		}
		return vmi
	}

	createBackupTracker := func(name, vmName, checkpointName string) *backupv1.VirtualMachineBackupTracker {
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
			Status: &backupv1.VirtualMachineBackupTrackerStatus{},
		}
		if checkpointName != "" {
			tracker.Status.LatestCheckpoint = &backupv1.BackupCheckpoint{
				Name:         checkpointName,
				CreationTime: &metav1.Time{Time: metav1.Now().Time},
			}
		}
		return tracker
	}

	createVM := func(name string) *v1.VirtualMachine {
		return &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
				UID:       vmUID,
			},
		}
	}

	createPVC := func(name string) *corev1.PersistentVolumeClaim {
		return &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: testNamespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeMode: pointer.P(corev1.PersistentVolumeFilesystem),
			},
		}
	}

	addBackup := func(backup *backupv1.VirtualMachineBackup) {
		backupSource.Add(backup)
		backupInformer.GetStore().Add(backup)
		_, err := kubevirtClient.BackupV1alpha1().VirtualMachineBackups(backup.Namespace).Get(context.Background(), backup.Name, metav1.GetOptions{})
		if err != nil {
			_, err = kubevirtClient.BackupV1alpha1().VirtualMachineBackups(backup.Namespace).Create(context.Background(), backup, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}
		key := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Name}.String()
		controller.backupQueue.Add(key)
	}

	createBackupVMExport := func(backup *backupv1.VirtualMachineBackup) *exportv1.VirtualMachineExport {
		return &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      backup.Name,
				Namespace: testNamespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(backup, backupv1.SchemeGroupVersion.WithKind(
						backupv1.VirtualMachineBackupGroupVersionKind.Kind)),
				},
			},
			Spec: exportv1.VirtualMachineExportSpec{
				Source: corev1.TypedLocalObjectReference{
					APIGroup: pointer.P(backupv1.VirtualMachineBackupGroupVersionKind.Group),
					Kind:     backupv1.VirtualMachineBackupGroupVersionKind.Kind,
					Name:     backup.Name,
				},
			},
		}
	}

	syncBackup := func(backup *backupv1.VirtualMachineBackup) (*backupv1.VirtualMachineBackup, error) {
		backupCopy := backup.DeepCopy()
		if backupCopy.Status == nil {
			backupCopy.Status = &backupv1.VirtualMachineBackupStatus{}
		}
		err := controller.sync(backupCopy)
		return backupCopy, err
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		backupInformer, backupSource = testutils.NewFakeInformerWithIndexersFor(
			&backupv1.VirtualMachineBackup{},
			cache.Indexers{
				"vmi": func(obj interface{}) ([]string, error) {
					backup := obj.(*backupv1.VirtualMachineBackup)
					if backup.Spec.Source.Kind == v1.VirtualMachineGroupVersionKind.Kind {
						return []string{types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()}, nil
					}
					key := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()
					return []string{key}, nil
				},
				"backupTracker": func(obj interface{}) ([]string, error) {
					backup := obj.(*backupv1.VirtualMachineBackup)
					if backup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
						return []string{types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()}, nil
					}
					return nil, nil
				},
			},
		)
		backupTrackerInformer, _ = testutils.NewFakeInformerWithIndexersFor(
			&backupv1.VirtualMachineBackupTracker{},
			cache.Indexers{
				"vmi": func(obj interface{}) ([]string, error) {
					tracker := obj.(*backupv1.VirtualMachineBackupTracker)
					return []string{types.NamespacedName{Namespace: tracker.Namespace, Name: tracker.Spec.Source.Name}.String()}, nil
				},
			},
		)
		vmInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		vmiInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		pvcInformer, _ = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})
		vmExportInformer, _ = testutils.NewFakeInformerFor(&exportv1.VirtualMachineExport{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		controller = &VMBackupController{
			client:                virtClient,
			backupInformer:        backupInformer,
			backupTrackerInformer: backupTrackerInformer,
			vmStore:               vmInformer.GetStore(),
			vmiStore:              vmiInformer.GetStore(),
			pvcStore:              pvcInformer.GetStore(),
			vmExportStore:         vmExportInformer.GetStore(),
			recorder:              recorder,
			backupQueue: workqueue.NewTypedRateLimitingQueueWithConfig(
				workqueue.DefaultTypedControllerRateLimiter[string](),
				workqueue.TypedRateLimitingQueueConfig[string]{Name: "test-backup-queue"},
			),
		}
		controller.hasSynced = func() bool {
			return backupInformer.HasSynced() && backupTrackerInformer.HasSynced() && vmInformer.HasSynced() && vmiInformer.HasSynced() && pvcInformer.HasSynced()
		}

		mockBackupQueue = testutils.NewMockWorkQueue(controller.backupQueue)
		controller.backupQueue = mockBackupQueue

		virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(testNamespace).Return(vmiInterface).AnyTimes()

		kubevirtClient = kubevirtfake.NewSimpleClientset()
		virtClient.EXPECT().VirtualMachineBackup(testNamespace).
			Return(kubevirtClient.BackupV1alpha1().VirtualMachineBackups(testNamespace)).AnyTimes()
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(kubevirtClient.ExportV1().VirtualMachineExports(testNamespace)).AnyTimes()

		k8sClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
	})

	Context("Verify source name", func() {
		It("should fail when source name is empty", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Spec.Source.Name = ""

			addBackup(backup)
			err := controller.execute(types.NamespacedName{Namespace: testNamespace, Name: backup.Name}.String())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errSourceNameEmpty))
		})

		It("should get source name from backupTracker when source kind is VirtualMachineBackupTracker", func() {
			backupTracker := createBackupTracker(backupTrackerName, vmName, "")
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			vmi := createInitializedVMI()
			controller.vmiStore.Add(vmi)

			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			vmiInterface.EXPECT().
				Backup(gomock.Any(), vmName, gomock.Any()).
				DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
					Expect(options.BackupName).To(Equal(backupName))
					return nil
				})

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionProgressing))).To(BeTrue())
			Expect(backupCopy.Status.Type).To(Equal(backupv1.Full))
		})
	})

	It("should wait when backupTracker does not exist yet", func() {
		backup := createBackupWithTracker(backupName, vmName, pvcName)
		addBackup(backup)
		// Don't add backupTracker

		statusUpdated := false
		kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update := action.(testing.UpdateAction)
			if update.GetSubresource() != "status" {
				return false, nil, nil
			}
			statusUpdated = true
			updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)

			Expect(meta.IsStatusConditionTrue(updateObj.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
			cond := meta.FindStatusCondition(updateObj.Status.Conditions, string(backupv1.ConditionInitializing))
			Expect(cond.Message).To(ContainSubstring(fmt.Sprintf(backupTrackerNotFoundMsg, backupTrackerName)))
			return true, updateObj, nil
		})

		err := controller.execute(types.NamespacedName{Namespace: testNamespace, Name: backupName}.String())
		Expect(err).ToNot(HaveOccurred())
		Expect(statusUpdated).To(BeTrue())
	})

	It("should wait when backupTracker needs checkpoint redefinition", func() {
		backupTracker := createBackupTracker(backupTrackerName, vmName, "existing-checkpoint")
		backupTracker.Status.CheckpointRedefinitionRequired = pointer.P(true)
		controller.backupTrackerInformer.GetStore().Add(backupTracker)

		backup := createBackupWithTracker(backupName, vmName, pvcName)
		addBackup(backup)

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createInitializedVMI()
		controller.vmiStore.Add(vmi)

		statusUpdated := false
		kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update := action.(testing.UpdateAction)
			if update.GetSubresource() != "status" {
				return false, nil, nil
			}
			statusUpdated = true
			updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)

			Expect(meta.IsStatusConditionTrue(updateObj.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
			cond := meta.FindStatusCondition(updateObj.Status.Conditions, string(backupv1.ConditionInitializing))
			Expect(cond.Message).To(ContainSubstring(fmt.Sprintf(trackerCheckpointRedefinitionPending, backupTrackerName)))
			return true, updateObj, nil
		})

		err := controller.execute(types.NamespacedName{Namespace: testNamespace, Name: backupName}.String())
		Expect(err).ToNot(HaveOccurred())
		Expect(statusUpdated).To(BeTrue())
	})

	Context("source verification", func() {
		Context("sourceVMExists", func() {
			It("should return false when VM doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				exists, err := controller.sourceVMExists(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})

			It("should return true when VM exists", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				exists, err := controller.sourceVMExists(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("vmiFromSource", func() {
			It("should return false when VMI doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				vmi, exists, err := controller.vmiFromSource(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(vmi).To(BeNil())
			})

			It("should return VMI when it exists", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				expectedVMI := createVMI()
				controller.vmiStore.Add(expectedVMI)
				vmi, exists, err := controller.vmiFromSource(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
				Expect(vmi).To(Equal(expectedVMI))
			})
		})

		Context("verifyVMIEligibleForBackup", func() {
			It("should return reason when VMI doesn't have CBT eligible volumes", func() {
				vmi := createVMI()
				vmi.Spec.Volumes = []v1.Volume{}
				reason := controller.verifyVMIEligibleForBackup(vmi)
				Expect(reason).To(Equal(fmt.Sprintf(vmNoVolumesToBackupMsg, vmName)))
			})

			It("should return reason when VMI doesn't have ChangedBlockTracking", func() {
				vmi := createVMI()
				vmi.Status.ChangedBlockTracking = nil
				reason := controller.verifyVMIEligibleForBackup(vmi)
				Expect(reason).To(Equal(fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmName)))
			})

			It("should return reason when ChangedBlockTracking is not enabled", func() {
				vmi := createVMI()
				vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{
					State: v1.ChangedBlockTrackingDisabled,
				}
				reason := controller.verifyVMIEligibleForBackup(vmi)
				Expect(reason).To(Equal(fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmName)))
			})

			It("should return empty string when VMI has eligible volumes and CBT enabled", func() {
				vmi := createVMI()
				reason := controller.verifyVMIEligibleForBackup(vmi)
				Expect(reason).To(BeEmpty())
			})
		})

		Context("sync during initialization", func() {
			It("should wait when VM doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				controller.backupInformer.GetStore().Add(backup)

				backupCopy, err := syncBackup(backup)
				Expect(err).ToNot(HaveOccurred())
				Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
				cond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))
				Expect(cond.Message).To(Equal(fmt.Sprintf(vmNotFoundMsg, testNamespace, vmName)))
			})

			It("should wait when VMI doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				controller.backupInformer.GetStore().Add(backup)

				backupCopy, err := syncBackup(backup)
				Expect(err).ToNot(HaveOccurred())
				Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
				cond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))
				Expect(cond.Message).To(Equal(fmt.Sprintf(vmNotRunningMsg, vmName)))
			})

			It("should wait when VMI doesn't have CBT eligible volumes", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				vmi := createVMI()
				vmi.Spec.Volumes = []v1.Volume{}
				controller.vmiStore.Add(vmi)
				controller.backupInformer.GetStore().Add(backup)

				backupCopy, err := syncBackup(backup)
				Expect(err).ToNot(HaveOccurred())
				Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
				cond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))
				Expect(cond.Message).To(Equal(fmt.Sprintf(vmNoVolumesToBackupMsg, vmName)))
			})

			It("should wait when ChangedBlockTracking is not enabled", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				vmi := createVMI()
				vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{
					State: v1.ChangedBlockTrackingDisabled,
				}
				controller.vmiStore.Add(vmi)
				controller.backupInformer.GetStore().Add(backup)

				backupCopy, err := syncBackup(backup)
				Expect(err).ToNot(HaveOccurred())
				Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
				cond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionInitializing))
				Expect(cond.Message).To(Equal(fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmName)))
			})

			It("should wait when VMI is migrating and update initializing condition", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				vmi := createVMI()
				now := metav1.Now()
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					StartTimestamp: &now,
				}
				controller.vmiStore.Add(vmi)
				addBackup(backup)

				statusUpdated := false
				kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					update := action.(testing.UpdateAction)
					if update.GetSubresource() != "status" {
						return false, nil, nil
					}
					statusUpdated = true
					updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)

					Expect(meta.IsStatusConditionTrue(updateObj.Status.Conditions, string(backupv1.ConditionInitializing))).To(BeTrue())
					cond := meta.FindStatusCondition(updateObj.Status.Conditions, string(backupv1.ConditionInitializing))
					Expect(cond.Message).To(ContainSubstring(fmt.Sprintf(vmMigrationInProgressMsg, vmName)))
					return true, updateObj, nil
				})

				err := controller.execute(types.NamespacedName{Namespace: testNamespace, Name: backupName}.String())
				Expect(err).ToNot(HaveOccurred())
				Expect(statusUpdated).To(BeTrue())
			})

			It("should proceed when VMI migration has completed", func() {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
				backup.Finalizers = []string{vmBackupFinalizer}
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				vmi := createInitializedVMI()
				now := metav1.Now()
				vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
					StartTimestamp: &now,
					EndTimestamp:   &now,
				}
				controller.vmiStore.Add(vmi)
				pvc := createPVC(pvcName)
				controller.pvcStore.Add(pvc)
				controller.backupInformer.GetStore().Add(backup)

				vmiInterface.EXPECT().
					Backup(gomock.Any(), vmName, gomock.Any()).
					Return(nil)

				backupCopy, err := syncBackup(backup)
				Expect(err).ToNot(HaveOccurred())
				Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionProgressing))).To(BeTrue())
			})
		})
	})

	Context("addBackupFinalizer", func() {
		It("should add finalizer when not present", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			Expect(backup.Finalizers).To(BeEmpty())

			addBackup(backup)

			patched := false
			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patchAction := action.(testing.PatchAction)
				Expect(patchAction.GetPatchType()).To(Equal(types.JSONPatchType))
				patched = true

				updatedBackup := backup.DeepCopy()
				updatedBackup.Finalizers = []string{vmBackupFinalizer}
				return true, updatedBackup, nil
			})

			err := controller.addBackupFinalizer(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(patched).To(BeTrue())
		})

		It("should not re-add finalizer if already present", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}

			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Fail("Should not patch when finalizer already exists")
				return true, nil, fmt.Errorf("unexpected patch call")
			})

			err := controller.addBackupFinalizer(backup)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	It("should do nothing when backup already done", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []metav1.Condition{
				newCondition(string(backupv1.ConditionProgressing), metav1.ConditionFalse, "Progressing", ""),
				newCondition(string(backupv1.ConditionDone), metav1.ConditionTrue, "Done", ""),
			},
		}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		// VMI should have no backup status and no utility volumes when backup is done
		vmi := createVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		// No patch should be called - backup is done and cleanup is complete
		_, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should cleanup when VMI backup status is missing", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []metav1.Condition{
				newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
			},
		}
		backup.Finalizers = []string{vmBackupFinalizer}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		// VMI with no backup status (backup status is nil)
		vmi := createVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		// Should not call any patches since nothing to cleanup
		vmiInterface.EXPECT().
			Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)

		backupCopy, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
		doneCond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionDone))
		Expect(doneCond.Message).To(ContainSubstring("VMI backup status was lost"))
	})

	Context("Backup deletion cleanup", func() {
		It("should populate includedVolumes early when backup in progress and volumes available", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
				// IncludedVolumes not yet set
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			// VMI with backup in progress but volumes already populated by virt-launcher
			volumesInfo := []backupv1.BackupVolumeInfo{
				{VolumeName: "rootdisk", DiskTarget: "vda"},
				{VolumeName: "datadisk", DiskTarget: "vdb"},
			}
			vmi := createInitializedVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus.Completed = false
			vmi.Status.ChangedBlockTracking.BackupStatus.Volumes = volumesInfo
			controller.vmiStore.Add(vmi)

			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(2))
			Expect(backupCopy.Status.IncludedVolumes[0].VolumeName).To(Equal("rootdisk"))
			Expect(backupCopy.Status.IncludedVolumes[1].VolumeName).To(Equal("datadisk"))
		})

		It("should not update includedVolumes when already set in backup status", func() {
			existingVolumes := []backupv1.BackupVolumeInfo{
				{VolumeName: "rootdisk", DiskTarget: "vda"},
			}
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
				IncludedVolumes: existingVolumes, // Already set
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			// VMI with backup in progress and volumes available
			vmi := createInitializedVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus.Completed = false
			vmi.Status.ChangedBlockTracking.BackupStatus.Volumes = existingVolumes
			controller.vmiStore.Add(vmi)

			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(1))
		})

		It("should patch VMI to remove backup status when backup is completed", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
					newCondition(string(backupv1.ConditionDone), metav1.ConditionFalse, "Done", ""),
				},
			}
			vm := createVM(vmName)
			controller.vmStore.Add(vm)
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName:     backupName,
				Completed:      true,
				CheckpointName: pointer.P(checkpointName),
			}
			controller.vmiStore.Add(vmi)
			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
		})

		It("should remove finalizer when a completed backup is being deleted", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionFalse, "Progressing", ""),
					newCondition(string(backupv1.ConditionDone), metav1.ConditionTrue, "Done", ""),
				},
			}
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}

			finalizerPatched := false
			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				finalizerPatched = true
				updatedBackup := backup.DeepCopy()
				updatedBackup.Finalizers = []string{}
				return true, updatedBackup, nil
			})

			_, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(finalizerPatched).To(BeTrue())
		})
	})

	Context("initialization failures", func() {
		It("should handle backup deletion during initialization when VMI is already gone", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Finalizers = []string{vmBackupFinalizer}

			patched := false
			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patchAction := action.(testing.PatchAction)
				if patchAction.GetName() == backupName {
					patched = true
				}
				return true, backup, nil
			})

			_, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(patched).To(BeTrue())
		})

		It("should handle backup deletion during initialization when the VMI exists", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Finalizers = []string{vmBackupFinalizer}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName: backupName,
			}
			controller.vmiStore.Add(vmi)

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
			doneCond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionDone))
			Expect(doneCond.Message).To(ContainSubstring("backup was deleted during initialization"))
		})

		It("should retry cleanup if it fails when backup is deleted during initialization", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Finalizers = []string{vmBackupFinalizer}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName: backupName,
			}
			controller.vmiStore.Add(vmi)

			conflictErr := errors.NewApplyConflict([]metav1.StatusCause{}, "conflict error")
			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(nil, conflictErr)

			_, err := syncBackup(backup)
			Expect(err).To(MatchError(conflictErr))
		})
	})

	Context("progressing failures", func() {
		It("should fail backup if VMI is deleted while backup is progressing", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
			doneCond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionDone))
			Expect(doneCond.Message).To(ContainSubstring("VMI was deleted during backup"))
			Eventually(recorder.Events).Should(Receive(ContainSubstring(backupFailedEvent)))
		})

		It("should initiate abort if backup is deleted while progressing", func() {
			backupTracker := createBackupTracker(backupTrackerName, vmName, "new-checkpoint")
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)
			vmi := createInitializedVMI()
			controller.vmiStore.Add(vmi)

			vmiInterface.EXPECT().
				Backup(gomock.Any(), vmName, gomock.Any()).
				DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
					Expect(options.Cmd).To(Equal(backupv1.Abort))
					return nil
				})

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionAborting))).To(BeTrue())
			Eventually(recorder.Events).Should(Receive(ContainSubstring(backupAbortingEvent)))
		})

		It("should record warning event when aborting a push-mode backup", func() {
			backupTracker := createBackupTracker(backupTrackerName, vmName, "new-checkpoint")
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.Spec.Mode = pointer.P(backupv1.PushMode)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)
			vmi := createInitializedVMI()
			controller.vmiStore.Add(vmi)

			vmiInterface.EXPECT().
				Backup(gomock.Any(), vmName, gomock.Any()).
				Return(nil)

			syncBackup(backup)
			var event string
			Eventually(recorder.Events).Should(Receive(&event))
			Expect(event).To(ContainSubstring(backupAbortingEvent))
			Expect(event).To(ContainSubstring(string(corev1.EventTypeWarning)))
		})

		It("should wait if backup is already marked as aborting", func() {
			backupTracker := createBackupTracker(backupTrackerName, vmName, "new-checkpoint")
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
					newCondition(string(backupv1.ConditionAborting), metav1.ConditionTrue, "Aborting", backupAborting),
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)
			vmi := createInitializedVMI()
			controller.vmiStore.Add(vmi)

			_, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should finalize backup as failed when abort completes", func() {
			backupTracker := createBackupTracker(backupTrackerName, vmName, "new-checkpoint")
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
					newCondition(string(backupv1.ConditionAborting), metav1.ConditionTrue, "Aborting", backupAborting),
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			vmiCanceled := createInitializedVMI()
			vmiCanceled.Spec.UtilityVolumes = nil
			vmiCanceled.Status.VolumeStatus = nil
			vmiCanceled.Status.ChangedBlockTracking.BackupStatus.Completed = true
			vmiCanceled.Status.ChangedBlockTracking.BackupStatus.Failed = true
			vmiCanceled.Status.ChangedBlockTracking.BackupStatus.BackupMsg = pointer.P("backup aborted")
			controller.vmiStore.Add(vmiCanceled)

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmiCanceled, nil)

			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackuptrackers", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Fail("Backup was canceled and failed, should not update the tracker")
				return true, nil, nil
			})

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
			doneCond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionDone))
			Expect(doneCond.Message).To(ContainSubstring("backup aborted"))
			Expect(meta.IsStatusConditionFalse(backupCopy.Status.Conditions, string(backupv1.ConditionAborting))).To(BeTrue())
			Eventually(recorder.Events).Should(Receive(ContainSubstring(backupFailedEvent)))
		})

		It("should initiate cleanup if VMI stops running while progressing", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}
			addBackup(backup)

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			vmi := createInitializedVMI()
			vmi.Status.Phase = v1.Failed
			controller.vmiStore.Add(vmi)

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			_, err := syncBackup(backup)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not done cleaning"))
		})

		It("should fail backup when cleanup completes for a stopped VMI", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}
			addBackup(backup)

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			vmiDetached := createInitializedVMI()
			vmiDetached.Status.Phase = v1.Failed
			vmiDetached.Spec.UtilityVolumes = nil
			vmiDetached.Status.VolumeStatus = nil
			controller.vmiStore.Add(vmiDetached)

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmiDetached, nil)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
			doneCond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionDone))
			Expect(doneCond.Message).To(ContainSubstring("VMI is not in a running state"))
		})
	})

	It("should fail backup when VMI is nil during completion check", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Status = &backupv1.VirtualMachineBackupStatus{}
		err := controller.checkBackupCompletion(backup, nil, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backup.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
		doneCond := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionDone))
		Expect(doneCond.Message).To(ContainSubstring("unexpected state: VMI is nil"))
	})

	Context("handleBackupInitiation", func() {
		It("should return error if updateSourceBackupInProgress fails", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)
			vmi := createVMI()
			controller.vmiStore.Add(vmi)

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("patch failed"))

			err := controller.handleBackupInitiation(backup, vmi, nil, log.DefaultLogger())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to update source backup in progress"))
		})

		It("should return error if Start backup command fails", func() {
			backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)
			vmi := createInitializedVMI()
			controller.vmiStore.Add(vmi)
			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			vmiInterface.EXPECT().
				Backup(gomock.Any(), vmName, gomock.Any()).
				Return(fmt.Errorf("api error"))

			err := controller.handleBackupInitiation(backup, vmi, nil, log.DefaultLogger())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to send Start backup command"))
		})
	})

	Context("updateSourceBackupInProgress", func() {
		It("should fail when another backup is already in progress", func() {
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName:     "other-backup",
				Completed:      false,
				CheckpointName: pointer.P("other-checkpoint"),
			}

			err := controller.updateSourceBackupInProgress(vmi, backupName, metav1.Now())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("another backup"))
			Expect(err.Error()).To(ContainSubstring("other-backup"))
			Expect(err.Error()).To(ContainSubstring("already in progress"))
		})

		It("should successfully patch VMI to add backup status", func() {
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = nil

			patched := false
			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, name string, patchType types.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
					patched = true
					Expect(string(patchBytes)).To(ContainSubstring("backupStatus"))
					Expect(string(patchBytes)).To(ContainSubstring(backupName))
					return vmi, nil
				})

			err := controller.updateSourceBackupInProgress(vmi, backupName, metav1.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(patched).To(BeTrue())
		})

		It("should return nil when same backup already in progress", func() {
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName:     backupName,
				Completed:      false,
				CheckpointName: pointer.P(checkpointName),
			}

			vmiInterface.EXPECT().
				Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)

			err := controller.updateSourceBackupInProgress(vmi, backupName, metav1.Now())
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("resolveCompletion", func() {
		DescribeTable("should correctly resolve completion status and record event",
			func(isFailed bool, msg *string, expectedDoneReason string, expectedMessageContains string, expectedEvent string) {
				backup := createBackup(backupName, vmName, pvcName, backupv1.PullMode)
				backup.Status = &backupv1.VirtualMachineBackupStatus{}

				backupStatus := &v1.VirtualMachineInstanceBackupStatus{
					Failed:    isFailed,
					BackupMsg: msg,
				}

				controller.resolveCompletion(backup, backupStatus)

				Expect(meta.IsStatusConditionTrue(backup.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
				doneCond := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionDone))
				Expect(doneCond.Reason).To(Equal(expectedDoneReason))
				Expect(doneCond.Message).To(ContainSubstring(expectedMessageContains))
				Eventually(recorder.Events).Should(Receive(ContainSubstring(expectedEvent)))
			},
			Entry("failure with a message",
				true, pointer.P("disk error"),
				"Failed", "disk error", backupFailedEvent,
			),
			Entry("failure without a message (nil check)",
				true, nil,
				"Failed", "unknown, no completion message", backupFailedEvent,
			),
			Entry("success with a warning message",
				false, pointer.P("quiesce failed"),
				"CompletedWithWarning", "quiesce failed", backupCompletedWithWarningEvent,
			),
			Entry("success",
				false, nil,
				"Completed", backupCompleted, backupCompletedEvent,
			),
		)
	})

	It("should attach PVC and return when PVC not yet attached", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Finalizers = []string{vmBackupFinalizer}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createVMI()
		vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
			BackupName:     backupName,
			Completed:      false,
			CheckpointName: pointer.P(checkpointName),
		}
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, patchType types.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				Expect(string(patchBytes)).To(ContainSubstring("utilityVolumes"))
				Expect(string(patchBytes)).To(ContainSubstring(pvcName))
				return vmi, nil
			})

		_, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should successfully initiate backup with Full type", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Finalizers = []string{vmBackupFinalizer}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createInitializedVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.TargetPath).ToNot(BeNil())
				return nil
			})

		backupCopy, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionProgressing))).To(BeTrue())
		Expect(backupCopy.Status.Type).To(Equal(backupv1.Full))
	})

	It("should initiate full backup when backupTracker exists but has no LatestCheckpoint", func() {
		backupTracker := createBackupTracker(backupTrackerName, vmName, "")
		controller.backupTrackerInformer.GetStore().Add(backupTracker)

		backup := createBackupWithTracker(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createInitializedVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.TargetPath).ToNot(BeNil())
				Expect(options.Incremental).To(BeNil())
				return nil
			})

		backupCopy, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionProgressing))).To(BeTrue())
		Expect(backupCopy.Status.Type).To(Equal(backupv1.Full))
	})

	It("should initiate incremental backup when backupTracker has LatestCheckpoint", func() {
		backupTracker := createBackupTracker(backupTrackerName, vmName, checkpointName)
		controller.backupTrackerInformer.GetStore().Add(backupTracker)

		backup := createBackupWithTracker(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createInitializedVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.TargetPath).ToNot(BeNil())
				Expect(options.Incremental).ToNot(BeNil())
				Expect(*options.Incremental).To(Equal(checkpointName))
				return nil
			})

		backupCopy, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionProgressing))).To(BeTrue())
		Expect(backupCopy.Status.Type).To(Equal(backupv1.Incremental))
	})

	It("should initiate full backup with ForceFullBackup even with LatestCheckpoint", func() {
		backupTracker := createBackupTracker(backupTrackerName, vmName, checkpointName)
		controller.backupTrackerInformer.GetStore().Add(backupTracker)

		backup := createBackupWithTracker(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Spec.ForceFullBackup = true

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createInitializedVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.TargetPath).ToNot(BeNil())
				Expect(options.Incremental).To(BeNil())
				return nil
			})

		backupCopy, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionProgressing))).To(BeTrue())
		Expect(backupCopy.Status.Type).To(Equal(backupv1.Full))
	})

	It("should initiate cleanup when backup completed", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []metav1.Condition{
				newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
			},
		}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		// VMI with backup completed and PVC still attached
		vmi := createInitializedVMI()
		vmi.Status.ChangedBlockTracking.BackupStatus.Completed = true
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, patchType types.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				Expect(string(patchBytes)).To(ContainSubstring("utilityVolumes"))
				return vmi, nil
			})

		_, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should remove backup status from VMI and return completed event when already detached", func() {
		backup := createBackup(backupName, vmName, pvcName, backupv1.PushMode)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []metav1.Condition{
				newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
			},
		}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		// VMI with backup completed and PVC already detached
		volumesInfo := []backupv1.BackupVolumeInfo{
			{VolumeName: "rootdisk", DiskTarget: "vda"},
			{VolumeName: "datadisk", DiskTarget: "vdb"},
		}
		vmi := createVMI()
		vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
			BackupName:     backupName,
			Completed:      true,
			CheckpointName: pointer.P(checkpointName),
			Volumes:        volumesInfo,
		}
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, patchType types.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				Expect(string(patchBytes)).To(ContainSubstring("backupStatus"))
				Expect(string(patchBytes)).To(ContainSubstring("remove"))
				return vmi, nil
			})

		backupCopy, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
		Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(2))
		Expect(backupCopy.Status.IncludedVolumes[0].VolumeName).To(Equal("rootdisk"))
		Expect(backupCopy.Status.IncludedVolumes[1].VolumeName).To(Equal("datadisk"))
		// checkpointName should NOT be populated since there's no BackupTracker
		Expect(backupCopy.Status.CheckpointName).To(BeNil())
	})

	DescribeTable("should update backupTracker with checkpoint and volumes info when backup completes",
		func(existingCheckpoint string, expectedOp string) {
			backupTracker := createBackupTracker(backupTrackerName, vmName, existingCheckpoint)
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}

			addBackup(backup)

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			// VMI with backup completed, checkpoint name, and volumes info
			volumesInfo := []backupv1.BackupVolumeInfo{
				{VolumeName: "rootdisk", DiskTarget: "vda"},
				{VolumeName: "datadisk", DiskTarget: "vdb"},
			}
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName:     backupName,
				Completed:      true,
				CheckpointName: pointer.P(checkpointName),
				Volumes:        volumesInfo,
			}
			controller.vmiStore.Add(vmi)

			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			// Expect patch to remove backup status from VMI
			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			// Expect patch to update backupTracker with checkpoint and volumes info
			trackerPatched := false
			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackuptrackers", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patchAction := action.(testing.PatchAction)
				Expect(patchAction.GetName()).To(Equal(backupTrackerName))
				Expect(patchAction.GetSubresource()).To(Equal("status"))

				patchBytes := patchAction.GetPatch()
				trackerPatched = true
				Expect(string(patchBytes)).To(ContainSubstring(expectedOp))
				Expect(string(patchBytes)).To(ContainSubstring("latestCheckpoint"))
				Expect(string(patchBytes)).To(ContainSubstring(checkpointName))
				Expect(string(patchBytes)).To(ContainSubstring("volumes"))
				Expect(string(patchBytes)).To(ContainSubstring("rootdisk"))
				Expect(string(patchBytes)).To(ContainSubstring("vda"))
				Expect(string(patchBytes)).To(ContainSubstring("datadisk"))
				Expect(string(patchBytes)).To(ContainSubstring("vdb"))

				updatedTracker := backupTracker.DeepCopy()
				updatedTracker.Status = &backupv1.VirtualMachineBackupTrackerStatus{
					LatestCheckpoint: &backupv1.BackupCheckpoint{
						Name:         checkpointName,
						CreationTime: &metav1.Time{Time: metav1.Now().Time},
						Volumes:      volumesInfo,
					},
				}
				return true, updatedTracker, nil
			})

			virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
				Return(kubevirtClient.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionDone))).To(BeTrue())
			Expect(trackerPatched).To(BeTrue())
			Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(2))
			Expect(backupCopy.Status.IncludedVolumes[0].VolumeName).To(Equal("rootdisk"))
			Expect(backupCopy.Status.IncludedVolumes[0].DiskTarget).To(Equal("vda"))
			Expect(backupCopy.Status.IncludedVolumes[1].VolumeName).To(Equal("datadisk"))
			Expect(backupCopy.Status.IncludedVolumes[1].DiskTarget).To(Equal("vdb"))
		},
		Entry("when tracker has no previous checkpoint", "", "\"op\":\"add\""),
		Entry("when tracker already has a checkpoint", "old-checkpoint", "\"op\":\"replace\""),
	)

	It("should update backupTracker even when cleanup returns early", func() {
		backupTracker := createBackupTracker(backupTrackerName, vmName, "")
		controller.backupTrackerInformer.GetStore().Add(backupTracker)

		backup := createBackupWithTracker(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []metav1.Condition{
				newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
			},
		}

		addBackup(backup)

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		// VMI with backup completed but PVC still attached (cleanup will return early)
		vmi := createInitializedVMI()
		vmi.Status.ChangedBlockTracking.BackupStatus.Completed = true
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		// Expect backupTracker to be patched
		trackerPatched := false
		kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackuptrackers", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patchAction := action.(testing.PatchAction)
			Expect(patchAction.GetName()).To(Equal(backupTrackerName))
			trackerPatched = true
			return true, backupTracker, nil
		})

		virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
			Return(kubevirtClient.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))

		// Expect VMI patch for detaching PVC (cleanup returns early)
		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
			Return(vmi, nil)

		_, err := syncBackup(backup)
		Expect(err).ToNot(HaveOccurred())
		Expect(trackerPatched).To(BeTrue())
	})

	Context("Pull mode", func() {
		var (
			backup   *backupv1.VirtualMachineBackup
			vmExport *exportv1.VirtualMachineExport
			vmi      *v1.VirtualMachineInstance
		)
		BeforeEach(func() {
			backup = createBackup(backupName, vmName, pvcName, backupv1.PullMode)
			backup.CreationTimestamp = metav1.Now()
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []metav1.Condition{
					newCondition(string(backupv1.ConditionProgressing), metav1.ConditionTrue, "Progressing", ""),
				},
			}
			vmExport = createBackupVMExport(backup)
			vmi = createInitializedVMI()
			controller.vmiStore.Add(vmi)
		})

		It("should return false for a new backup", func() {
			Expect(isPullBackupTTLExpired(backup)).To(BeFalse())
		})

		It("should return true when creation time exceeds default TTL", func() {
			backup.CreationTimestamp = metav1.NewTime(time.Now().Add(-3 * time.Hour))
			Expect(isPullBackupTTLExpired(backup)).To(BeTrue())
		})

		DescribeTable("should respect a custom TTL", func(age, ttl time.Duration, expectedExpired bool) {
			backup.Spec.TTLDuration = &metav1.Duration{Duration: ttl}
			backup.CreationTimestamp = metav1.NewTime(time.Now().Add(-age))
			Expect(isPullBackupTTLExpired(backup)).To(Equal(expectedExpired))
		},
			Entry("when not yet expired", 5*time.Minute, 10*time.Minute, false),
			Entry("when just expired", 15*time.Minute, 10*time.Minute, true),
			Entry("when exactly at boundary is expired", 10*time.Minute, 10*time.Minute, true),
		)

		It("should return zero when already expired", func() {
			backup.CreationTimestamp = metav1.NewTime(time.Now().Add(-5 * time.Hour))
			Expect(getPullBackupRemainingTTL(backup).Duration).To(BeZero())
		})

		It("should return a positive duration for a new backup", func() {
			remaining := getPullBackupRemainingTTL(backup)
			Expect(remaining.Duration).To(BeNumerically(">", 0))
			Expect(remaining.Duration).To(BeNumerically("<=", defaultPullModeDurationTTL))
		})

		It("should return full TTL when CreationTimestamp is zero", func() {
			backup.CreationTimestamp = metav1.Time{}
			Expect(getPullBackupRemainingTTL(backup).Duration).To(Equal(defaultPullModeDurationTTL))
		})

		It("should account for elapsed time", func() {
			backup.CreationTimestamp = metav1.NewTime(time.Now().Add(-30 * time.Minute))
			remaining := getPullBackupRemainingTTL(backup)
			Expect(remaining.Duration).To(BeNumerically(">", 89*time.Minute))
			Expect(remaining.Duration).To(BeNumerically("<", 91*time.Minute))
		})

		It("should return nil when export is not yet in Ready phase", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))

			vmExport = createBackupVMExport(backup)
			vmExport.Status = &exportv1.VirtualMachineExportStatus{Phase: exportv1.Pending}
			controller.vmExportStore.Add(vmExport)

			_, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should update includedVolumes when export is in Ready phase but the backup has no included volumes", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))

			vmExport := createBackupVMExport(backup)
			vmExport.Status = &exportv1.VirtualMachineExportStatus{Phase: exportv1.Ready}
			controller.vmExportStore.Add(vmExport)

			volume := backupv1.BackupVolumeInfo{
				VolumeName: "datadisk",
				DiskTarget: "vda",
			}
			vmi.Status.ChangedBlockTracking.BackupStatus.Volumes = append(vmi.Status.ChangedBlockTracking.BackupStatus.Volumes, volume)
			controller.vmiStore.Update(vmi)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(backupCopy.Status.IncludedVolumes).To(ContainElement(volume))
		})

		It("should return an error when export is ready but has no links", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))
			backup.Status.IncludedVolumes = append(backup.Status.IncludedVolumes, backupv1.BackupVolumeInfo{
				VolumeName: "datadisk",
				DiskTarget: "vda",
			})
			vmExport.Status = &exportv1.VirtualMachineExportStatus{Phase: exportv1.Ready, Links: nil}
			controller.vmExportStore.Add(vmExport)

			_, err := syncBackup(backup)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no backup links")))
		})

		It("should return an error when the export is ready but the cert is empty", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))
			backup.Status.IncludedVolumes = append(backup.Status.IncludedVolumes, backupv1.BackupVolumeInfo{
				VolumeName: "datadisk",
				DiskTarget: "vda",
			})
			vmExport.Status = &exportv1.VirtualMachineExportStatus{
				Phase: exportv1.Ready,
				Links: &exportv1.VirtualMachineExportLinks{
					Internal: &exportv1.VirtualMachineExportLink{
						Cert: "",
						Backups: []exportv1.VirtualMachineExportBackup{{
							Name: pvcName,
							Endpoints: []exportv1.VirtualMachineExportBackupEndpoint{{
								Url:      "data",
								Endpoint: exportv1.Data,
							}},
						}},
					},
				},
			}
			controller.vmExportStore.Add(vmExport)

			_, err := syncBackup(backup)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no cert exposed")))
		})

		It("should return ExportReady with populated endpoints using internal links", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))
			backup.Status.IncludedVolumes = []backupv1.BackupVolumeInfo{{VolumeName: pvcName}}
			vmExport.Status = &exportv1.VirtualMachineExportStatus{
				Phase: exportv1.Ready,
				Links: &exportv1.VirtualMachineExportLinks{
					Internal: &exportv1.VirtualMachineExportLink{
						Cert: "test",
						Backups: []exportv1.VirtualMachineExportBackup{{
							Name: pvcName,
							Endpoints: []exportv1.VirtualMachineExportBackupEndpoint{
								{Url: "/data", Endpoint: exportv1.Data},
								{Url: "/map", Endpoint: exportv1.Map},
							},
						}},
					},
				},
			}
			controller.vmExportStore.Add(vmExport)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionExportReady))).To(BeTrue())
			Expect(backupCopy.Status.EndpointCert).ToNot(BeNil())
			Expect(*backupCopy.Status.EndpointCert).ToNot(BeEmpty())
			Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(1))
			Expect(backupCopy.Status.IncludedVolumes[0].DataEndpoint).To(Equal("/data"))
			Expect(backupCopy.Status.IncludedVolumes[0].MapEndpoint).To(Equal("/map"))
		})

		It("should prioritize external links over internal links", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))
			backup.Status.IncludedVolumes = []backupv1.BackupVolumeInfo{{VolumeName: pvcName}}
			vmExport.Status = &exportv1.VirtualMachineExportStatus{
				Phase: exportv1.Ready,
				Links: &exportv1.VirtualMachineExportLinks{
					Internal: &exportv1.VirtualMachineExportLink{
						Cert: "test",
						Backups: []exportv1.VirtualMachineExportBackup{{
							Name: pvcName,
							Endpoints: []exportv1.VirtualMachineExportBackupEndpoint{
								{Url: "/internal/data", Endpoint: exportv1.Data},
								{Url: "/internal/map", Endpoint: exportv1.Map},
							},
						}},
					},
					External: &exportv1.VirtualMachineExportLink{
						Cert: "test",
						Backups: []exportv1.VirtualMachineExportBackup{{
							Name: pvcName,
							Endpoints: []exportv1.VirtualMachineExportBackupEndpoint{
								{Url: "/external/data", Endpoint: exportv1.Data},
								{Url: "/external/map", Endpoint: exportv1.Map},
							},
						}},
					},
				},
			}
			controller.vmExportStore.Add(vmExport)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionExportReady))).To(BeTrue())
			Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(1))
			Expect(backupCopy.Status.IncludedVolumes[0].DataEndpoint).To(Equal("/external/data"))
			Expect(backupCopy.Status.IncludedVolumes[0].MapEndpoint).To(Equal("/external/map"))
		})

		It("should map endpoints independently for multiple volumes", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""))
			backup.Status.IncludedVolumes = []backupv1.BackupVolumeInfo{
				{VolumeName: "rootdisk"},
				{VolumeName: "datadisk"},
			}
			vmExport.Status = &exportv1.VirtualMachineExportStatus{
				Phase: exportv1.Ready,
				Links: &exportv1.VirtualMachineExportLinks{
					Internal: &exportv1.VirtualMachineExportLink{
						Cert: pvcName,
						Backups: []exportv1.VirtualMachineExportBackup{
							{
								Name: "rootdisk",
								Endpoints: []exportv1.VirtualMachineExportBackupEndpoint{
									{Url: "/rootdisk/data", Endpoint: exportv1.Data},
									{Url: "/rootdisk/map", Endpoint: exportv1.Map},
								},
							},
							{
								Name: "datadisk",
								Endpoints: []exportv1.VirtualMachineExportBackupEndpoint{
									{Url: "/datadisk/data", Endpoint: exportv1.Data},
									{Url: "/datadisk/map", Endpoint: exportv1.Map},
								},
							},
						},
					},
				},
			}
			controller.vmExportStore.Add(vmExport)

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(backupCopy.Status.IncludedVolumes).To(HaveLen(2))
			for _, vol := range backupCopy.Status.IncludedVolumes {
				Expect(vol.DataEndpoint).To(ContainSubstring(vol.VolumeName))
				Expect(vol.MapEndpoint).To(ContainSubstring(vol.VolumeName))
			}
		})

		It("should return an error when export exists but is not owned by this backup", func() {
			unownedExport := &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupName,
					Namespace: testNamespace,
				},
			}
			controller.vmExportStore.Add(unownedExport)

			errMsg := fmt.Sprintf(exportExistsWithDifferentOwner, backupName, backupName)
			_, err := syncBackup(backup)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring(errMsg)))
		})

		It("should create a new export when none exists and set preparing export conditions", func() {

			kubevirtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				createAction := action.(testing.CreateAction)

				exp := createAction.GetObject().(*exportv1.VirtualMachineExport)

				Expect(exp.Name).To(Equal(backup.Name))
				Expect(exp.Namespace).To(Equal(backup.Namespace))
				Expect(metav1.IsControlledBy(exp, backup)).To(BeTrue())
				Expect(exp.Spec.Source.Kind).To(Equal(backupv1.VirtualMachineBackupGroupVersionKind.Kind))
				Expect(exp.Spec.Source.Name).To(Equal(backup.Name))
				Expect(exp.Spec.TTLDuration).ToNot(BeNil())

				return false, nil, nil
			})

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionFalse(backupCopy.Status.Conditions, string(backupv1.ConditionExportInitiated))).To(BeTrue())
		})

		It("should set a TTL on the export that reflects elapsed time", func() {
			backup.CreationTimestamp = metav1.NewTime(time.Now().Add(-30 * time.Minute))

			kubevirtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				createAction := action.(testing.CreateAction)

				exp := createAction.GetObject().(*exportv1.VirtualMachineExport)

				Expect(exp.Spec.TTLDuration).ToNot(BeNil())
				Expect(exp.Spec.TTLDuration.Duration).To(BeNumerically(">", 89*time.Minute))
				Expect(exp.Spec.TTLDuration.Duration).To(BeNumerically("<", 91*time.Minute))

				return false, nil, nil
			})

			syncBackup(backup)
		})

		It("should reset to Progressing condition when the export has disappeared", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""),
				newCondition(string(backupv1.ConditionExportReady), metav1.ConditionTrue, "ExportReady", ""),
			)
			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionFalse(backupCopy.Status.Conditions, string(backupv1.ConditionExportInitiated))).To(BeTrue())
		})

		It("should abort when backup is still in progress at TTL expiry", func() {
			vmi.Status.ChangedBlockTracking.BackupStatus.Completed = false
			controller.vmiStore.Update(vmi)
			backup.Spec.TTLDuration = &metav1.Duration{Duration: 5 * time.Minute}
			backup.CreationTimestamp = metav1.NewTime(time.Now().Add(-5 * time.Minute))

			vmiInterface.EXPECT().
				Backup(gomock.Any(), vmName, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, opts *backupv1.BackupOptions) error {
					Expect(opts.Cmd).To(Equal(backupv1.Abort))
					return nil
				})

			backupCopy, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(meta.IsStatusConditionTrue(backupCopy.Status.Conditions, string(backupv1.ConditionAborting))).To(BeTrue())
			abortCond := meta.FindStatusCondition(backupCopy.Status.Conditions, string(backupv1.ConditionAborting))
			Expect(abortCond.Message).To(ContainSubstring(backupTTLExpiredMsg))
		})

		It("should delete the export when the backup completed", func() {
			backup.Status.Conditions = append(backup.Status.Conditions,
				newCondition(string(backupv1.ConditionExportInitiated), metav1.ConditionTrue, "ExportInitiated", ""),
				newCondition(string(backupv1.ConditionExportReady), metav1.ConditionTrue, "ExportReady", ""),
			)
			vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{
				State: v1.ChangedBlockTrackingEnabled,
				BackupStatus: &v1.VirtualMachineInstanceBackupStatus{
					BackupName: backupName,
					Completed:  true,
				},
			}
			controller.vmiStore.Update(vmi)
			controller.vmExportStore.Add(vmExport)

			deleteCalled := false
			kubevirtClient.Fake.PrependReactor("delete", "virtualmachineexports", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				deleteAction := action.(testing.DeleteAction)

				if deleteAction.GetName() == vmExport.Name && deleteAction.GetNamespace() == backup.Namespace {
					deleteCalled = true
				}
				return false, nil, nil
			})

			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, types.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			_, err := syncBackup(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(deleteCalled).To(BeTrue())
		})
	})
})
