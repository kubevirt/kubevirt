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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
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
	vmUID     k8stypes.UID = "vm-uid"
	backupUID k8stypes.UID = "backup-uid"
)

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
		controller            *VMBackupController
		recorder              *record.FakeRecorder
		mockBackupQueue       *testutils.MockWorkQueue[string]

		kubevirtClient *kubevirtfake.Clientset
		k8sClient      *fake.Clientset
	)

	createBackup := func(name, vmName, pvcName string) *backupv1.VirtualMachineBackup {
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
				Mode:    pointer.P(backupv1.PushMode),
			},
		}
	}
	createBackupWithTracker := func(name, vmName, pvcName string) *backupv1.VirtualMachineBackup {
		backup := createBackup(name, vmName, pvcName)
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
		key := fmt.Sprintf("%s/%s", backup.Namespace, backup.Name)
		controller.backupQueue.Add(key)
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
						return []string{fmt.Sprintf("%s/%s", backup.Namespace, backup.Spec.Source.Name)}, nil
					}
					key := fmt.Sprintf("%s/%s", backup.Namespace, backup.Spec.Source.Name)
					return []string{key}, nil
				},
				"backupTracker": func(obj interface{}) ([]string, error) {
					backup := obj.(*backupv1.VirtualMachineBackup)
					if backup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
						return []string{fmt.Sprintf("%s/%s", backup.Namespace, backup.Spec.Source.Name)}, nil
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
					return []string{fmt.Sprintf("%s/%s", tracker.Namespace, tracker.Spec.Source.Name)}, nil
				},
			},
		)
		vmInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		vmiInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		pvcInformer, _ = testutils.NewFakeInformerFor(&corev1.PersistentVolumeClaim{})

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		controller = &VMBackupController{
			client:                virtClient,
			backupInformer:        backupInformer,
			backupTrackerInformer: backupTrackerInformer,
			vmStore:               vmInformer.GetStore(),
			vmiStore:              vmiInformer.GetStore(),
			pvcStore:              pvcInformer.GetStore(),
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

		k8sClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
	})

	Context("Verify source name", func() {
		It("should fail when source name is empty", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Spec.Source.Name = ""

			addBackup(backup)
			err := controller.execute(fmt.Sprintf("%s/%s", testNamespace, backupName))
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

			backupCalled := false
			vmiInterface.EXPECT().
				Backup(gomock.Any(), vmName, gomock.Any()).
				DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
					backupCalled = true
					Expect(options.BackupName).To(Equal(backupName))
					return nil
				})

			syncInfo := controller.sync(backup)
			Expect(syncInfo).ToNot(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
			Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
			Expect(syncInfo.backupType).To(Equal(backupv1.Full))
			Expect(backupCalled).To(BeTrue())
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

			hasInitializing := false
			for _, cond := range updateObj.Status.Conditions {
				if cond.Type == backupv1.ConditionInitializing &&
					cond.Status == corev1.ConditionTrue {
					hasInitializing = true
					Expect(cond.Reason).To(ContainSubstring(fmt.Sprintf(backupTrackerNotFoundMsg, backupTrackerName)))
				}
			}
			Expect(hasInitializing).To(BeTrue(), "Should have Initializing condition")
			return true, updateObj, nil
		})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitializingEvent))
		Expect(syncInfo.reason).To(ContainSubstring(fmt.Sprintf(backupTrackerNotFoundMsg, backupTrackerName)))

		err := controller.updateStatus(backup, syncInfo, log.DefaultLogger())
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

			hasInitializing := false
			for _, cond := range updateObj.Status.Conditions {
				if cond.Type == backupv1.ConditionInitializing &&
					cond.Status == corev1.ConditionTrue {
					hasInitializing = true
					Expect(cond.Reason).To(ContainSubstring(fmt.Sprintf(trackerCheckpointRedefinitionPending, backupTrackerName)))
				}
			}
			Expect(hasInitializing).To(BeTrue(), "Should have Initializing condition")
			return true, updateObj, nil
		})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitializingEvent))
		Expect(syncInfo.reason).To(ContainSubstring(fmt.Sprintf(trackerCheckpointRedefinitionPending, backupTrackerName)))

		err := controller.updateStatus(backup, syncInfo, log.DefaultLogger())
		Expect(err).ToNot(HaveOccurred())
		Expect(statusUpdated).To(BeTrue())
	})

	Context("source verification", func() {
		Context("sourceVMExists", func() {
			It("should return false when VM doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName)
				exists, err := controller.sourceVMExists(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})

			It("should return true when VM exists", func() {
				backup := createBackup(backupName, vmName, pvcName)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				exists, err := controller.sourceVMExists(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})
		})

		Context("vmiFromSource", func() {
			It("should return false when VMI doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName)
				vmi, exists, err := controller.vmiFromSource(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
				Expect(vmi).To(BeNil())
			})

			It("should return VMI when it exists", func() {
				backup := createBackup(backupName, vmName, pvcName)
				expectedVMI := createVMI()
				controller.vmiStore.Add(expectedVMI)
				vmi, exists, err := controller.vmiFromSource(backup, vmName)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
				Expect(vmi).To(Equal(expectedVMI))
			})
		})

		Context("verifyVMIEligibleForBackup", func() {
			It("should fail when VMI doesn't have CBT eligible volumes", func() {
				vmi := createVMI()
				vmi.Spec.Volumes = []v1.Volume{}
				syncInfo := controller.verifyVMIEligibleForBackup(vmi, backupName)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNoVolumesToBackupMsg, vmName)))
			})

			It("should fail when VMI doesn't have ChangedBlockTracking", func() {
				vmi := createVMI()
				vmi.Status.ChangedBlockTracking = nil
				syncInfo := controller.verifyVMIEligibleForBackup(vmi, backupName)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmName)))
			})

			It("should fail when ChangedBlockTracking is not enabled", func() {
				vmi := createVMI()
				vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{
					State: v1.ChangedBlockTrackingDisabled,
				}
				syncInfo := controller.verifyVMIEligibleForBackup(vmi, backupName)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmName)))
			})

			It("should succeed when VMI has eligible volumes and CBT enabled", func() {
				vmi := createVMI()
				syncInfo := controller.verifyVMIEligibleForBackup(vmi, backupName)
				Expect(syncInfo).To(BeNil())
			})
		})

		Context("sync during initialization", func() {
			It("should wait when VM doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName)
				// Don't add VM to store
				controller.backupInformer.GetStore().Add(backup)

				syncInfo := controller.sync(backup)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNotFoundMsg, testNamespace, vmName)))
			})

			It("should wait when VMI doesn't exist", func() {
				backup := createBackup(backupName, vmName, pvcName)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				controller.backupInformer.GetStore().Add(backup)
				// Don't add VMI to store

				syncInfo := controller.sync(backup)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNotRunningMsg, vmName)))
			})

			It("should wait when VMI doesn't have CBT eligible volumes", func() {
				backup := createBackup(backupName, vmName, pvcName)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				vmi := createVMI()
				vmi.Spec.Volumes = []v1.Volume{}
				controller.vmiStore.Add(vmi)
				controller.backupInformer.GetStore().Add(backup)

				syncInfo := controller.sync(backup)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNoVolumesToBackupMsg, vmName)))
			})

			It("should wait when ChangedBlockTracking is not enabled", func() {
				backup := createBackup(backupName, vmName, pvcName)
				vm := createVM(vmName)
				controller.vmStore.Add(vm)
				vmi := createVMI()
				vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{
					State: v1.ChangedBlockTrackingDisabled,
				}
				controller.vmiStore.Add(vmi)
				controller.backupInformer.GetStore().Add(backup)

				syncInfo := controller.sync(backup)
				Expect(syncInfo).ToNot(BeNil())
				Expect(syncInfo.event).To(Equal(backupInitializingEvent))
				Expect(syncInfo.reason).To(Equal(fmt.Sprintf(vmNoChangedBlockTrackingMsg, vmName)))
			})
		})
	})

	Context("addBackupFinalizer", func() {
		It("should add finalizer when not present", func() {
			backup := createBackup(backupName, vmName, pvcName)
			Expect(backup.Finalizers).To(BeEmpty())

			addBackup(backup)

			patched := false
			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				patchAction := action.(testing.PatchAction)
				Expect(patchAction.GetPatchType()).To(Equal(k8stypes.JSONPatchType))
				patched = true

				updatedBackup := backup.DeepCopy()
				updatedBackup.Finalizers = []string{vmBackupFinalizer}
				return true, updatedBackup, nil
			})

			result, err := controller.addBackupFinalizer(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(patched).To(BeTrue())
			Expect(result.Finalizers).To(ContainElement(vmBackupFinalizer))
		})

		It("should not re-add finalizer if already present", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}

			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Fail("Should not patch when finalizer already exists")
				return true, nil, fmt.Errorf("unexpected patch call")
			})

			result, err := controller.addBackupFinalizer(backup)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(backup))
			Expect(result.Finalizers).To(ContainElement(vmBackupFinalizer))
		})
	})

	It("should do nothing when backup already done", func() {
		backup := createBackup(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []backupv1.Condition{
				{Type: backupv1.ConditionProgressing, Status: corev1.ConditionFalse},
				{Type: backupv1.ConditionDone, Status: corev1.ConditionTrue},
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
		syncInfo := controller.sync(backup)
		Expect(syncInfo).To(BeNil())
	})

	It("should cleanup when VMI backup status is missing", func() {
		backup := createBackup(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []backupv1.Condition{
				{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
			},
		}

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

		syncInfo := controller.sync(backup)
		Expect(syncInfo).To(BeNil())
	})

	Context("Backup deletion cleanup", func() {
		It("should wait when backup deleted but still in progress on VMI", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			// VMI with backup still in progress (not completed)
			vmi := createInitializedVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus.Completed = false
			controller.vmiStore.Add(vmi)

			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			// Should not call any patches - waiting for backup to complete
			vmiInterface.EXPECT().
				Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Times(0)

			syncInfo := controller.sync(backup)
			// Returns nil - waiting for completion
			Expect(syncInfo).To(BeNil())
		})

		It("should populate includedVolumes early when backup in progress and volumes available", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
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

			syncInfo := controller.sync(backup)
			Expect(syncInfo).ToNot(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
			// Should return includedVolumes without a completion event
			Expect(syncInfo.event).To(BeEmpty())
			Expect(syncInfo.includedVolumes).To(HaveLen(2))
			Expect(syncInfo.includedVolumes[0].VolumeName).To(Equal("rootdisk"))
			Expect(syncInfo.includedVolumes[1].VolumeName).To(Equal("datadisk"))
		})

		It("should not update includedVolumes when already set in backup status", func() {
			existingVolumes := []backupv1.BackupVolumeInfo{
				{VolumeName: "rootdisk", DiskTarget: "vda"},
			}
			backup := createBackup(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
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

			syncInfo := controller.sync(backup)
			// Should return nil since volumes already set - no update needed
			Expect(syncInfo).To(BeNil())
		})

		It("should proceed with cleanup when backup deleting and completed", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionFalse},
					{Type: backupv1.ConditionDone, Status: corev1.ConditionTrue},
				},
			}

			vm := createVM(vmName)
			controller.vmStore.Add(vm)

			// VMI with backup completed and PVC already detached
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName:     backupName,
				Completed:      true,
				CheckpointName: pointer.P(checkpointName),
			}
			controller.vmiStore.Add(vmi)

			pvc := createPVC(pvcName)
			controller.pvcStore.Add(pvc)

			// Expect patch to remove backup status
			vmiInterface.EXPECT().
				Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
				Return(vmi, nil)

			// Expect patch to remove finalizer
			finalizerPatched := false
			kubevirtClient.Fake.PrependReactor("patch", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				finalizerPatched = true
				updatedBackup := backup.DeepCopy()
				updatedBackup.Finalizers = []string{}
				return true, updatedBackup, nil
			})

			syncInfo := controller.sync(backup)
			Expect(syncInfo).To(BeNil())
			Expect(finalizerPatched).To(BeTrue())
		})
	})

	Context("updateStatus", func() {
		It("should initialize status when nil", func() {
			backup := createBackup(backupName, vmName, pvcName)
			Expect(backup.Status).To(BeNil())

			addBackup(backup)

			statusUpdated := false
			kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update := action.(testing.UpdateAction)
				if update.GetSubresource() != "status" {
					return false, nil, nil
				}
				statusUpdated = true
				updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)
				Expect(updateObj.Status).ToNot(BeNil())
				Expect(updateObj.Status.Conditions).To(HaveLen(2))
				return true, updateObj, nil
			})

			err := controller.updateStatus(backup, nil, log.DefaultLogger())
			Expect(err).ToNot(HaveOccurred())
			Expect(statusUpdated).To(BeTrue())
		})

		DescribeTable("should update to Progressing and set Type when backup initiated",
			func(backupType backupv1.BackupType) {
				backup := createBackup(backupName, vmName, pvcName)
				backup.Status = &backupv1.VirtualMachineBackupStatus{
					Conditions: []backupv1.Condition{
						{Type: backupv1.ConditionInitializing, Status: corev1.ConditionTrue},
					},
				}

				addBackup(backup)

				syncInfo := &SyncInfo{
					event:      backupInitiatedEvent,
					reason:     backupInProgress,
					backupType: backupType,
				}

				statusUpdated := false
				kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					update := action.(testing.UpdateAction)
					if update.GetSubresource() != "status" {
						return false, nil, nil
					}
					statusUpdated = true
					updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)
					hasProgressing := false
					for _, cond := range updateObj.Status.Conditions {
						if cond.Type == backupv1.ConditionProgressing && cond.Status == corev1.ConditionTrue {
							hasProgressing = true
						}
					}
					Expect(hasProgressing).To(BeTrue())
					Expect(updateObj.Status.Type).To(Equal(backupType))
					return true, updateObj, nil
				})

				err := controller.updateStatus(backup, syncInfo, log.DefaultLogger())
				Expect(err).ToNot(HaveOccurred())
				Expect(statusUpdated).To(BeTrue())
			},
			Entry("Full backup type", backupv1.Full),
			Entry("Incremental backup type", backupv1.Incremental),
		)

		DescribeTable("should update to Done when backup completed and preserve existing Type",
			func(backupType backupv1.BackupType) {
				backup := createBackup(backupName, vmName, pvcName)
				backup.Status = &backupv1.VirtualMachineBackupStatus{
					Type: backupType, // Type was already set when backup was initiated
					Conditions: []backupv1.Condition{
						{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
					},
				}

				addBackup(backup)

				syncInfo := &SyncInfo{
					event:  backupCompletedEvent,
					reason: backupCompleted,
				}

				statusUpdated := false
				kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					update := action.(testing.UpdateAction)
					if update.GetSubresource() != "status" {
						return false, nil, nil
					}
					statusUpdated = true
					updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)
					hasDone := false
					for _, cond := range updateObj.Status.Conditions {
						if cond.Type == backupv1.ConditionDone && cond.Status == corev1.ConditionTrue {
							hasDone = true
						}
					}
					Expect(hasDone).To(BeTrue())
					// Type should be preserved from when it was set at initiation
					Expect(updateObj.Status.Type).To(Equal(backupType))
					return true, updateObj, nil
				})

				err := controller.updateStatus(backup, syncInfo, log.DefaultLogger())
				Expect(err).ToNot(HaveOccurred())
				Expect(statusUpdated).To(BeTrue())
			},
			Entry("Full backup type", backupv1.Full),
			Entry("Incremental backup type", backupv1.Incremental),
		)

		It("should record warning event when backup completed with warning", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
				},
			}

			addBackup(backup)

			warningMsg := "Some volumes could not be quiesced"
			syncInfo := &SyncInfo{
				event:  backupCompletedWithWarningEvent,
				reason: fmt.Sprintf(backupCompletedWithWarningMsg, warningMsg),
			}

			statusUpdated := false
			kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update := action.(testing.UpdateAction)
				if update.GetSubresource() != "status" {
					return false, nil, nil
				}
				statusUpdated = true
				return true, update.GetObject(), nil
			})

			err := controller.updateStatus(backup, syncInfo, log.DefaultLogger())
			Expect(err).ToNot(HaveOccurred())
			Expect(statusUpdated).To(BeTrue())

			Eventually(recorder.Events).Should(Receive(ContainSubstring(backupCompletedWithWarningEvent)))
		})

		It("should add Deleting condition when backup has deletion timestamp", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.DeletionTimestamp = &metav1.Time{Time: metav1.Now().Time}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
				},
			}

			addBackup(backup)

			statusUpdated := false
			kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update := action.(testing.UpdateAction)
				if update.GetSubresource() != "status" {
					return false, nil, nil
				}
				statusUpdated = true
				updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)
				hasDeleting := false
				for _, cond := range updateObj.Status.Conditions {
					if cond.Type == backupv1.ConditionDeleting && cond.Status == corev1.ConditionTrue {
						hasDeleting = true
					}
				}
				Expect(hasDeleting).To(BeTrue())
				return true, updateObj, nil
			})

			err := controller.updateStatus(backup, nil, log.DefaultLogger())
			Expect(err).ToNot(HaveOccurred())
			Expect(statusUpdated).To(BeTrue())
		})

		It("should not update when status unchanged", func() {
			backup := createBackup(backupName, vmName, pvcName)
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
				},
			}

			addBackup(backup)

			kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Fail("Should not update when status unchanged")
				return true, nil, fmt.Errorf("unexpected update call")
			})

			err := controller.updateStatus(backup, nil, log.DefaultLogger())
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("should update IncludedVolumes in backup status",
			func(event string, reason string, expectDoneCondition bool) {
				backup := createBackup(backupName, vmName, pvcName)
				backup.Status = &backupv1.VirtualMachineBackupStatus{
					Conditions: []backupv1.Condition{
						{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
					},
				}

				addBackup(backup)

				volumesInfo := []backupv1.BackupVolumeInfo{
					{VolumeName: "rootdisk", DiskTarget: "vda"},
					{VolumeName: "datadisk", DiskTarget: "vdb"},
				}
				syncInfo := &SyncInfo{
					event:           event,
					reason:          reason,
					includedVolumes: volumesInfo,
				}

				statusUpdated := false
				kubevirtClient.Fake.PrependReactor("update", "virtualmachinebackups", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					update := action.(testing.UpdateAction)
					if update.GetSubresource() != "status" {
						return false, nil, nil
					}
					statusUpdated = true
					updateObj := update.GetObject().(*backupv1.VirtualMachineBackup)
					if expectDoneCondition {
						hasDone := false
						for _, cond := range updateObj.Status.Conditions {
							if cond.Type == backupv1.ConditionDone && cond.Status == corev1.ConditionTrue {
								hasDone = true
							}
						}
						Expect(hasDone).To(BeTrue())
					}
					Expect(updateObj.Status.IncludedVolumes).To(HaveLen(2))
					Expect(updateObj.Status.IncludedVolumes[0].VolumeName).To(Equal("rootdisk"))
					Expect(updateObj.Status.IncludedVolumes[1].VolumeName).To(Equal("datadisk"))
					return true, updateObj, nil
				})

				err := controller.updateStatus(backup, syncInfo, log.DefaultLogger())
				Expect(err).ToNot(HaveOccurred())
				Expect(statusUpdated).To(BeTrue())
			},
			Entry("when backup in progress (early update)", "", "", false),
			Entry("when backup completed", backupCompletedEvent, backupCompleted, true),
		)
	})

	Context("updateSourceBackupInProgress", func() {
		It("should fail when another backup is already in progress", func() {
			vmi := createVMI()
			vmi.Status.ChangedBlockTracking.BackupStatus = &v1.VirtualMachineInstanceBackupStatus{
				BackupName:     "other-backup",
				Completed:      false,
				CheckpointName: pointer.P("other-checkpoint"),
			}

			err := controller.updateSourceBackupInProgress(vmi, backupName)
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
				Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
				DoAndReturn(func(ctx context.Context, name string, patchType k8stypes.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
					patched = true
					Expect(string(patchBytes)).To(ContainSubstring("backupStatus"))
					Expect(string(patchBytes)).To(ContainSubstring(backupName))
					return vmi, nil
				})

			err := controller.updateSourceBackupInProgress(vmi, backupName)
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

			err := controller.updateSourceBackupInProgress(vmi, backupName)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	It("should attach PVC and return when PVC not yet attached", func() {
		backup := createBackup(backupName, vmName, pvcName)
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

		patchCalled := false
		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, patchType k8stypes.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchCalled = true
				Expect(string(patchBytes)).To(ContainSubstring("utilityVolumes"))
				Expect(string(patchBytes)).To(ContainSubstring(pvcName))
				return vmi, nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitializingEvent))
		Expect(syncInfo.reason).To(Equal(fmt.Sprintf(attachTargetPVCMsg, pvcName, vmName)))
		Expect(patchCalled).To(BeTrue())
	})

	It("should successfully initiate backup and return backupInitiatedEvent with Full type", func() {
		backup := createBackup(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}

		vm := createVM(vmName)
		controller.vmStore.Add(vm)

		vmi := createInitializedVMI()
		controller.vmiStore.Add(vmi)

		pvc := createPVC(pvcName)
		controller.pvcStore.Add(pvc)

		backupCalled := false
		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				backupCalled = true
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.PushPath).ToNot(BeNil())
				return nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
		Expect(syncInfo.reason).To(Equal(backupInProgress))
		Expect(syncInfo.backupType).To(Equal(backupv1.Full))
		Expect(backupCalled).To(BeTrue())
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

		backupCalled := false
		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				backupCalled = true
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.PushPath).ToNot(BeNil())
				Expect(options.Incremental).To(BeNil())
				return nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
		Expect(syncInfo.reason).To(Equal(backupInProgress))
		Expect(syncInfo.backupType).To(Equal(backupv1.Full))
		Expect(backupCalled).To(BeTrue())
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

		backupCalled := false
		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				backupCalled = true
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.PushPath).ToNot(BeNil())
				Expect(options.Incremental).ToNot(BeNil())
				Expect(*options.Incremental).To(Equal(checkpointName))
				return nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
		Expect(syncInfo.reason).To(Equal(backupInProgress))
		Expect(syncInfo.backupType).To(Equal(backupv1.Incremental))
		Expect(backupCalled).To(BeTrue())
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

		backupCalled := false
		vmiInterface.EXPECT().
			Backup(gomock.Any(), vmName, gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, options *backupv1.BackupOptions) error {
				backupCalled = true
				Expect(options.BackupName).To(Equal(backupName))
				Expect(options.Cmd).To(Equal(backupv1.Start))
				Expect(options.Mode).To(Equal(backupv1.PushMode))
				Expect(options.PushPath).ToNot(BeNil())
				Expect(options.Incremental).To(BeNil())
				return nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
		Expect(syncInfo.reason).To(Equal(backupInProgress))
		Expect(syncInfo.backupType).To(Equal(backupv1.Full))
		Expect(backupCalled).To(BeTrue())
	})

	It("should initiate cleanup when backup completed", func() {
		backup := createBackup(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []backupv1.Condition{
				{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
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

		// Expect detach patch
		patchCalled := false
		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, patchType k8stypes.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchCalled = true
				Expect(string(patchBytes)).To(ContainSubstring("utilityVolumes"))
				return vmi, nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
		Expect(syncInfo.reason).To(Equal(fmt.Sprintf(detachTargetPVCMsg, vmName)))
		Expect(patchCalled).To(BeTrue())
	})

	It("should remove backup status from VMI and return completed event when already detached", func() {
		backup := createBackup(backupName, vmName, pvcName)
		backup.Finalizers = []string{vmBackupFinalizer}
		backup.Status = &backupv1.VirtualMachineBackupStatus{
			Conditions: []backupv1.Condition{
				{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
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

		// Expect patch to remove backup status
		patchCalled := false
		vmiInterface.EXPECT().
			Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, name string, patchType k8stypes.PatchType, patchBytes []byte, opts metav1.PatchOptions, subresources ...string) (*v1.VirtualMachineInstance, error) {
				patchCalled = true
				Expect(string(patchBytes)).To(ContainSubstring("backupStatus"))
				Expect(string(patchBytes)).To(ContainSubstring("remove"))
				return vmi, nil
			})

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		Expect(syncInfo.event).To(Equal(backupCompletedEvent))
		Expect(syncInfo.reason).To(Equal(backupCompleted))
		Expect(patchCalled).To(BeTrue())
		Expect(syncInfo.includedVolumes).To(HaveLen(2))
		Expect(syncInfo.includedVolumes[0].VolumeName).To(Equal("rootdisk"))
		Expect(syncInfo.includedVolumes[1].VolumeName).To(Equal("datadisk"))
		// checkpointName should NOT be populated since there's no BackupTracker
		Expect(syncInfo.checkpointName).To(BeNil())
	})

	DescribeTable("should update backupTracker with checkpoint and volumes info when backup completes",
		func(existingCheckpoint string, expectedOp string) {
			backupTracker := createBackupTracker(backupTrackerName, vmName, existingCheckpoint)
			controller.backupTrackerInformer.GetStore().Add(backupTracker)

			backup := createBackupWithTracker(backupName, vmName, pvcName)
			backup.Finalizers = []string{vmBackupFinalizer}
			backup.Status = &backupv1.VirtualMachineBackupStatus{
				Conditions: []backupv1.Condition{
					{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
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
				Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
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

			syncInfo := controller.sync(backup)
			Expect(syncInfo).ToNot(BeNil())
			Expect(syncInfo.err).ToNot(HaveOccurred())
			Expect(syncInfo.event).To(Equal(backupCompletedEvent))
			Expect(syncInfo.reason).To(Equal(backupCompleted))
			Expect(trackerPatched).To(BeTrue())
			Expect(syncInfo.includedVolumes).To(HaveLen(2))
			Expect(syncInfo.includedVolumes[0].VolumeName).To(Equal("rootdisk"))
			Expect(syncInfo.includedVolumes[0].DiskTarget).To(Equal("vda"))
			Expect(syncInfo.includedVolumes[1].VolumeName).To(Equal("datadisk"))
			Expect(syncInfo.includedVolumes[1].DiskTarget).To(Equal("vdb"))
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
			Conditions: []backupv1.Condition{
				{Type: backupv1.ConditionProgressing, Status: corev1.ConditionTrue},
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
			Patch(gomock.Any(), vmName, k8stypes.JSONPatchType, gomock.Any(), gomock.Any()).
			Return(vmi, nil)

		syncInfo := controller.sync(backup)
		Expect(syncInfo).ToNot(BeNil())
		Expect(syncInfo.err).ToNot(HaveOccurred())
		// Cleanup returned early (detaching PVC)
		Expect(syncInfo.event).To(Equal(backupInitiatedEvent))
		Expect(syncInfo.reason).To(ContainSubstring("detaching"))
		// But backupTracker was still updated before cleanup
		Expect(trackerPatched).To(BeTrue())
	})
})
