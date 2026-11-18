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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	backupv1 "kubevirt.io/api/backup/v1beta1"
	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Offline backup", func() {
	const (
		namespace = "default"
		vmName    = "testvm"
	)

	offlineBackup := func() *backupv1.VirtualMachineBackup {
		return &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: namespace},
			Spec: backupv1.VirtualMachineBackupSpec{
				Source: corev1.TypedLocalObjectReference{Kind: "VirtualMachine", Name: vmName},
			},
			Status: &backupv1.VirtualMachineBackupStatus{},
		}
	}

	vmWithVolumes := func() *v1.VirtualMachine {
		return &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: namespace},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: []v1.Volume{
							{Name: "rootdisk", VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{ClaimName: "rootdisk"},
							}}},
							{Name: "cloudinit", VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}},
							{Name: "hostdisk", VolumeSource: v1.VolumeSource{HostDisk: &v1.HostDisk{}}},
						},
					},
				},
			},
		}
	}

	Describe("offlineEligibleVolumes", func() {
		It("returns only PVC-backed CBT-eligible volumes, excluding HostDisk", func() {
			volumes := offlineEligibleVolumes(vmWithVolumes())
			Expect(volumes).To(ConsistOf(backupv1.BackupVolumeInfo{VolumeName: "rootdisk"}),
				"cloudinit isn't CBT-eligible, and hostdisk has no PVC the offline exporter pod can mount")
		})

		It("returns nil when the template is absent", func() {
			Expect(offlineEligibleVolumes(&v1.VirtualMachine{})).To(BeNil())
		})
	})

	Describe("startOfflineBackup", func() {
		It("fails fast if the VM never had ChangedBlockTracking enabled, instead of scheduling a doomed export", func() {
			vm := vmWithVolumes()
			podInformer, _ := testutils.NewFakeInformerFor(&corev1.Pod{})
			ctrl := &VMBackupController{
				podIndexer: podInformer.GetIndexer(),
				recorder:   record.NewFakeRecorder(100),
			}

			backup := offlineBackup()
			backup.Spec.Mode = pointer.P(backupv1.PullMode)

			Expect(ctrl.startOfflineBackup(backup, vm, nil)).To(Succeed())
			Expect(isBackupFailed(backup)).To(BeTrue())
			failed := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionFailed))
			Expect(failed).ToNot(BeNil())
			Expect(failed.Message).To(ContainSubstring("ChangedBlockTracking"))
		})

		It("adds the tracker finalizer alongside the backup finalizer when the backup belongs to a tracker", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			defer mockCtrl.Finish()
			kubevirtCli := kubevirtfake.NewSimpleClientset()
			virtClient := kubecli.NewMockKubevirtClient(mockCtrl)
			virtClient.EXPECT().VirtualMachineBackup(namespace).
				Return(kubevirtCli.BackupV1beta1().VirtualMachineBackups(namespace)).AnyTimes()
			virtClient.EXPECT().VirtualMachineBackupTracker(namespace).
				Return(kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(namespace)).AnyTimes()

			vm := vmWithVolumes()
			vm.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{State: v1.ChangedBlockTrackingEnabled}
			podInformer, _ := testutils.NewFakeInformerFor(&corev1.Pod{})
			ctrl := &VMBackupController{
				client:     virtClient,
				podIndexer: podInformer.GetIndexer(),
				recorder:   record.NewFakeRecorder(100),
			}

			backup := offlineBackup()
			backup.Spec.Mode = pointer.P(backupv1.PullMode)
			_, err := kubevirtCli.BackupV1beta1().VirtualMachineBackups(namespace).Create(
				context.Background(), backup, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			tracker := createTracker("tracker1", vmName)
			_, err = kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(namespace).Create(
				context.Background(), tracker, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(ctrl.startOfflineBackup(backup, vm, tracker)).To(Succeed())
			// The export controller refuses to build an export until StartTimestamp is set.
			Expect(backup.Status.StartTimestamp).ToNot(BeNil())

			updated, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(namespace).Get(
				context.Background(), tracker.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Finalizers).To(ContainElement(backupv1.VirtualMachineBackupTrackerFinalizer))
		})
	})

	Describe("VM hold lifecycle", func() {
		stoppedVM := func() *v1.VirtualMachine {
			vm := vmWithVolumes()
			vm.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{State: v1.ChangedBlockTrackingEnabled}
			return vm
		}

		newCtrl := func(vm *v1.VirtualMachine) (*VMBackupController, *kubevirtfake.Clientset) {
			mockCtrl := gomock.NewController(GinkgoT())
			kubevirtCli := kubevirtfake.NewSimpleClientset()
			virtClient := kubecli.NewMockKubevirtClient(mockCtrl)
			virtClient.EXPECT().VirtualMachine(namespace).
				Return(kubevirtCli.KubevirtV1().VirtualMachines(namespace)).AnyTimes()
			vmStore, _ := testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			trackerInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackupTracker{})
			if vm != nil {
				Expect(vmStore.GetStore().Add(vm)).To(Succeed())
				_, err := kubevirtCli.KubevirtV1().VirtualMachines(namespace).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			return &VMBackupController{
				client:                virtClient,
				vmStore:               vmStore.GetStore(),
				backupTrackerInformer: trackerInformer,
			}, kubevirtCli
		}

		It("holds a stopped eligible VM by stamping BackupInProgress", func() {
			vm := stoppedVM()
			ctrl, cli := newCtrl(vm)
			held, err := ctrl.holdSourceVM(offlineBackup(), vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(held).To(BeTrue())
			updated, err := cli.KubevirtV1().VirtualMachines(namespace).Get(context.Background(), vmName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.BackupInProgress).To(HaveValue(Equal("backup")))
		})

		It("refuses to hold a VM that is set to start", func() {
			vm := stoppedVM()
			vm.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
			ctrl, _ := newCtrl(vm)
			held, err := ctrl.holdSourceVM(offlineBackup(), vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(held).To(BeFalse())
		})

		It("is idempotent when the VM is already held by this backup and refuses a VM held by another", func() {
			vm := stoppedVM()
			vm.Status.BackupInProgress = pointer.P("backup")
			ctrl, _ := newCtrl(vm)
			held, err := ctrl.holdSourceVM(offlineBackup(), vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(held).To(BeTrue())

			vm.Status.BackupInProgress = pointer.P("other-backup")
			ctrl2, _ := newCtrl(vm)
			held, err = ctrl2.holdSourceVM(offlineBackup(), vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(held).To(BeFalse())
		})

		It("releases only the VM it holds", func() {
			vm := stoppedVM()
			vm.Status.BackupInProgress = pointer.P("backup")
			ctrl, cli := newCtrl(vm)
			Expect(ctrl.releaseSourceVM(offlineBackup())).To(Succeed())
			updated, err := cli.KubevirtV1().VirtualMachines(namespace).Get(context.Background(), vmName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated.Status.BackupInProgress).To(BeNil())

			vm2 := stoppedVM()
			vm2.Status.BackupInProgress = pointer.P("other-backup")
			ctrl2, cli2 := newCtrl(vm2)
			Expect(ctrl2.releaseSourceVM(offlineBackup())).To(Succeed())
			updated2, err := cli2.KubevirtV1().VirtualMachines(namespace).Get(context.Background(), vmName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updated2.Status.BackupInProgress).To(HaveValue(Equal("other-backup")))
		})

		It("heldSourceVM reports whether this backup holds the VM", func() {
			vm := stoppedVM()
			vm.Status.BackupInProgress = pointer.P("backup")
			ctrl, _ := newCtrl(vm)
			_, held, err := ctrl.heldSourceVM(offlineBackup(), vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(held).To(BeTrue())
		})
	})

	Describe("offlineCheckpointName", func() {
		It("derives the checkpoint from the backup name and creation time", func() {
			b := offlineBackup()
			b.CreationTimestamp = metav1.Date(2026, 9, 18, 10, 30, 0, 0, time.UTC)
			Expect(offlineCheckpointName(b)).To(Equal("backup-backup-2026-09-18_10-30-00"))
		})

		It("gives a recreated backup with the same name a different checkpoint", func() {
			b := offlineBackup()
			b.CreationTimestamp = metav1.Date(2026, 9, 18, 10, 30, 0, 0, time.UTC)
			recreated := offlineBackup()
			recreated.CreationTimestamp = metav1.Date(2026, 9, 18, 11, 0, 0, 0, time.UTC)
			Expect(offlineCheckpointName(b)).ToNot(Equal(offlineCheckpointName(recreated)))
		})
	})

	Describe("hasNonTerminalLauncherPods", func() {
		newController := func(pods ...*corev1.Pod) *VMBackupController {
			informer, _ := testutils.NewFakeInformerFor(&corev1.Pod{})
			for _, p := range pods {
				Expect(informer.GetIndexer().Add(p)).To(Succeed())
			}
			return &VMBackupController{podIndexer: informer.GetIndexer()}
		}

		launcherPod := func(phase corev1.PodPhase) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "virt-launcher-testvm",
					Namespace:   namespace,
					Labels:      map[string]string{v1.AppLabel: "virt-launcher"},
					Annotations: map[string]string{v1.DomainAnnotation: vmName},
				},
				Status: corev1.PodStatus{Phase: phase},
			}
		}

		It("detects a running launcher pod", func() {
			ctrl := newController(launcherPod(corev1.PodRunning))
			stale, err := ctrl.hasNonTerminalLauncherPods(namespace, vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(stale).To(BeTrue())
		})

		It("ignores terminal launcher pods", func() {
			ctrl := newController(launcherPod(corev1.PodSucceeded))
			stale, err := ctrl.hasNonTerminalLauncherPods(namespace, vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(stale).To(BeFalse())
		})

		It("ignores pods for other VMs", func() {
			pod := launcherPod(corev1.PodRunning)
			pod.Annotations[v1.DomainAnnotation] = "othervm"
			ctrl := newController(pod)
			stale, err := ctrl.hasNonTerminalLauncherPods(namespace, vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(stale).To(BeFalse())
		})

		It("detects a running launcher pod even when the VM has a custom hostname", func() {
			pod := launcherPod(corev1.PodRunning)
			pod.Labels[v1.DeprecatedVirtualMachineNameLabel] = "custom-hostname"
			ctrl := newController(pod)
			stale, err := ctrl.hasNonTerminalLauncherPods(namespace, vmName)
			Expect(err).ToNot(HaveOccurred())
			Expect(stale).To(BeTrue())
		})
	})

	Describe("applyOfflineVolumeBackupTypes", func() {
		exporterPod := func(backupName, message string) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-export-" + backupName,
					Namespace: namespace,
					Labels:    map[string]string{v1.AppLabel: exportv1.App},
					OwnerReferences: []metav1.OwnerReference{{
						Kind:       virtualMachineExportKind,
						Name:       backupName,
						Controller: pointer.P(true),
					}},
				},
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{{
						Name:  backupTypeProbeContainerName,
						State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Message: message}},
					}},
				},
			}
		}

		newController := func(pod *corev1.Pod) *VMBackupController {
			informer, _ := testutils.NewFakeInformerFor(&corev1.Pod{})
			if pod != nil {
				Expect(informer.GetIndexer().Add(pod)).To(Succeed())
			}
			return &VMBackupController{podIndexer: informer.GetIndexer()}
		}

		backupWith := func(volumes ...backupv1.BackupVolumeInfo) *backupv1.VirtualMachineBackup {
			b := offlineBackup()
			b.Status.IncludedVolumes = volumes
			return b
		}

		It("relays a per-volume fallback from the probe into status", func() {
			backup := backupWith(
				backupv1.BackupVolumeInfo{VolumeName: "rootdisk", Type: backupv1.Incremental},
				backupv1.BackupVolumeInfo{VolumeName: "datadisk", Type: backupv1.Incremental},
			)
			ctrl := newController(exporterPod(backup.Name, `[{"volumeName":"datadisk","type":"Full"}]`))
			Expect(ctrl.applyOfflineVolumeBackupTypes(backup)).To(Succeed())
			Expect(backup.Status.IncludedVolumes).To(ConsistOf(
				backupv1.BackupVolumeInfo{VolumeName: "rootdisk", Type: backupv1.Incremental},
				backupv1.BackupVolumeInfo{VolumeName: "datadisk", Type: backupv1.Full},
			))
		})

		It("leaves status unchanged on an unparseable probe message", func() {
			backup := backupWith(backupv1.BackupVolumeInfo{VolumeName: "rootdisk", Type: backupv1.Incremental})
			ctrl := newController(exporterPod(backup.Name, "not json"))
			Expect(ctrl.applyOfflineVolumeBackupTypes(backup)).To(Succeed())
			Expect(backup.Status.IncludedVolumes).To(ConsistOf(
				backupv1.BackupVolumeInfo{VolumeName: "rootdisk", Type: backupv1.Incremental},
			))
		})

		It("is a no-op when no exporter pod exists", func() {
			backup := backupWith(backupv1.BackupVolumeInfo{VolumeName: "rootdisk", Type: backupv1.Incremental})
			ctrl := newController(nil)
			Expect(ctrl.applyOfflineVolumeBackupTypes(backup)).To(Succeed())
			Expect(backup.Status.IncludedVolumes[0].Type).To(Equal(backupv1.Incremental))
		})
	})
})

var _ = Describe("Offline tracker advance gating", func() {
	const trackerName = "tracker1"
	var (
		mockCtrl       *gomock.Controller
		virtClient     *kubecli.MockKubevirtClient
		kubevirtCli    *kubevirtfake.Clientset
		exportInformer cache.SharedIndexInformer
		vmInformer     cache.SharedIndexInformer
		ctrl           *VMBackupController
	)

	pullBackup := func() *backupv1.VirtualMachineBackup {
		return &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: testNamespace},
			Spec: backupv1.VirtualMachineBackupSpec{
				Mode:   pointer.P(backupv1.PullMode),
				Source: corev1.TypedLocalObjectReference{Kind: "VirtualMachine", Name: "testvm"},
			},
			Status: &backupv1.VirtualMachineBackupStatus{
				CheckpointName:  pointer.P("backup-backup"),
				IncludedVolumes: []backupv1.BackupVolumeInfo{{VolumeName: "rootdisk"}},
			},
		}
	}

	// The advance path makes exactly one client call, a tracker patch, so its
	// presence or absence tells us whether the tracker advanced.
	expectTrackerPatch := func() {
		virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
			Return(kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace))
	}

	latestCheckpointName := func() string {
		updated, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace).Get(
			context.Background(), trackerName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if updated.Status == nil || updated.Status.LatestCheckpoint == nil {
			return ""
		}
		return updated.Status.LatestCheckpoint.Name
	}

	// Seeded with an existing checkpoint so an advance is observable as a name change.
	seedTracker := func() *backupv1.VirtualMachineBackupTracker {
		tracker := createTracker(trackerName, "testvm")
		tracker.Status = &backupv1.VirtualMachineBackupTrackerStatus{
			LatestCheckpoint: &backupv1.BackupCheckpoint{Name: "checkpoint-1"},
		}
		_, err := kubevirtCli.BackupV1beta1().VirtualMachineBackupTrackers(testNamespace).Create(
			context.Background(), tracker, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return tracker
	}

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		kubevirtCli = kubevirtfake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(mockCtrl)
		exportInformer, _ = testutils.NewFakeInformerFor(&exportv1.VirtualMachineExport{})
		vmInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		podInformer, _ := testutils.NewFakeInformerFor(&corev1.Pod{})
		trackerInformer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackupTracker{})
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(kubevirtCli.ExportV1().VirtualMachineExports(testNamespace)).AnyTimes()
		ctrl = &VMBackupController{
			client:                virtClient,
			vmStore:               vmInformer.GetStore(),
			vmExportStore:         exportInformer.GetStore(),
			podIndexer:            podInformer.GetIndexer(),
			backupTrackerInformer: trackerInformer,
			recorder:              record.NewFakeRecorder(100),
		}
	})

	Describe("reconcileOfflineDeleting (client-driven completion)", func() {
		It("advances the tracker when the export served", func() {
			tracker := seedTracker()
			backup := pullBackup()
			setExportReady(backup)
			expectTrackerPatch()

			Expect(ctrl.reconcileOfflineDeleting(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("backup-backup"))
		})

		It("does not advance when the export never became ready", func() {
			tracker := seedTracker()
			backup := pullBackup()
			setPreparingExport(backup)

			Expect(ctrl.reconcileOfflineDeleting(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
		})

		It("advances a push backup deleted after the export terminated but before completeOfflineBackup ran", func() {
			tracker := seedTracker()
			backup := pullBackup()
			backup.Spec.Mode = pointer.P(backupv1.PushMode)
			Expect(exportInformer.GetStore().Add(&exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{Name: backup.Name, Namespace: backup.Namespace},
				Status:     &exportv1.VirtualMachineExportStatus{Phase: exportv1.Terminated},
			})).To(Succeed())
			expectTrackerPatch()

			Expect(ctrl.reconcileOfflineDeleting(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("backup-backup"))
		})

		It("does not advance a push backup whose export never reached Terminated", func() {
			tracker := seedTracker()
			backup := pullBackup()
			backup.Spec.Mode = pointer.P(backupv1.PushMode)
			Expect(exportInformer.GetStore().Add(&exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{Name: backup.Name, Namespace: backup.Namespace},
				Status:     &exportv1.VirtualMachineExportStatus{Phase: exportv1.Ready},
			})).To(Succeed())

			Expect(ctrl.reconcileOfflineDeleting(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
		})
	})

	Describe("reconcileOfflinePull TTL expiry", func() {
		expiredBackup := func() *backupv1.VirtualMachineBackup {
			b := pullBackup()
			b.CreationTimestamp = metav1.NewTime(time.Now().Add(-3 * time.Hour))
			return b
		}

		It("fails the backup and does not advance even if the export had served", func() {
			tracker := seedTracker()
			backup := expiredBackup()
			setExportReady(backup)

			Expect(ctrl.reconcileOfflinePull(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
			Expect(isBackupFailed(backup)).To(BeTrue())
			Expect(isBackupComplete(backup)).To(BeFalse())
		})

		It("fails the backup and does not advance when it expired while still preparing", func() {
			tracker := seedTracker()
			backup := expiredBackup()
			setPreparingExport(backup)

			Expect(ctrl.reconcileOfflinePull(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
			Expect(isBackupFailed(backup)).To(BeTrue())
		})
	})

	Describe("reconcileOfflinePush (export-phase completion)", func() {
		pushBackup := func() *backupv1.VirtualMachineBackup {
			b := pullBackup()
			b.Spec.Mode = pointer.P(backupv1.PushMode)
			return b
		}

		seedExport := func(backup *backupv1.VirtualMachineBackup, phase exportv1.VirtualMachineExportPhase) {
			export := &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backup.Name,
					Namespace: backup.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(backup, backupv1.VirtualMachineBackupGroupVersionKind),
					},
				},
				Status: &exportv1.VirtualMachineExportStatus{Phase: phase},
			}
			Expect(exportInformer.GetStore().Add(export)).To(Succeed())
		}

		It("completes and advances the tracker once the export is Terminated", func() {
			tracker := seedTracker()
			backup := pushBackup()
			seedExport(backup, exportv1.Terminated)
			expectTrackerPatch()

			Expect(ctrl.reconcileOfflinePush(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("backup-backup"))
			Expect(isBackupComplete(backup)).To(BeTrue())

			quiesced := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionQuiesced))
			Expect(quiesced).ToNot(BeNil())
			Expect(quiesced.Reason).To(Equal(backupv1.ReasonQuiesceSkipped))
		})

		It("does not complete while the export is still running", func() {
			tracker := seedTracker()
			backup := pushBackup()
			seedExport(backup, exportv1.Ready)

			Expect(ctrl.reconcileOfflinePush(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
			Expect(isBackupComplete(backup)).To(BeFalse())
		})

		It("does not mark the backup complete while its exporter pod still exists, and completes once it's gone", func() {
			tracker := seedTracker()
			backup := pushBackup()
			seedExport(backup, exportv1.Terminated)

			exporterPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-exporter-" + backup.Name,
					Namespace: backup.Namespace,
					Labels:    map[string]string{v1.AppLabel: exportv1.App},
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "VirtualMachineExport", Name: backup.Name, Controller: pointer.P(true)},
					},
				},
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
			}
			Expect(ctrl.podIndexer.Add(exporterPod)).To(Succeed())

			expectTrackerPatch()
			err := ctrl.reconcileOfflinePush(backup, tracker)
			Expect(err).To(MatchError(errCleanupPending))
			Expect(isBackupComplete(backup)).To(BeFalse())

			Expect(ctrl.podIndexer.Delete(exporterPod)).To(Succeed())

			expectTrackerPatch()
			Expect(ctrl.reconcileOfflinePush(backup, tracker)).To(Succeed())
			Expect(isBackupComplete(backup)).To(BeTrue())
		})

		It("fails the backup when the export reports a failed pod", func() {
			tracker := seedTracker()
			backup := pushBackup()
			export := &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backup.Name,
					Namespace: backup.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(backup, backupv1.VirtualMachineBackupGroupVersionKind),
					},
				},
				Status: &exportv1.VirtualMachineExportStatus{
					Phase: exportv1.Pending,
					Conditions: []exportv1.Condition{
						{Type: exportv1.ConditionReady, Status: corev1.ConditionFalse, Reason: "PodFailed"},
					},
				},
			}
			Expect(exportInformer.GetStore().Add(export)).To(Succeed())

			Expect(ctrl.reconcileOfflinePush(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
			Expect(isBackupFailed(backup)).To(BeTrue())
			Expect(isBackupComplete(backup)).To(BeFalse())
		})
	})

	Describe("reconcileOfflinePull (export-phase failure)", func() {
		It("fails the backup when the export reports a failed pod, instead of retrying until the TTL expires", func() {
			tracker := seedTracker()
			backup := pullBackup()
			export := &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backup.Name,
					Namespace: backup.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(backup, backupv1.VirtualMachineBackupGroupVersionKind),
					},
				},
				Status: &exportv1.VirtualMachineExportStatus{
					Phase: exportv1.Pending,
					Conditions: []exportv1.Condition{
						{Type: exportv1.ConditionReady, Status: corev1.ConditionFalse, Reason: "PodFailed"},
					},
				},
			}
			Expect(exportInformer.GetStore().Add(export)).To(Succeed())

			Expect(ctrl.reconcileOfflinePull(backup, tracker)).To(Succeed())
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
			Expect(isBackupFailed(backup)).To(BeTrue())
			Expect(isBackupComplete(backup)).To(BeFalse())
		})
	})

	DescribeTable("isBackupExportReady",
		func(setup func(*backupv1.VirtualMachineBackup), expected bool) {
			backup := pullBackup()
			setup(backup)
			Expect(isBackupExportReady(backup)).To(Equal(expected))
		},
		Entry("true once the export is ready", setExportReady, true),
		Entry("false while still preparing", setPreparingExport, false),
		Entry("false with no progressing condition", func(*backupv1.VirtualMachineBackup) {}, false),
	)
})
