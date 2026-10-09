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
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
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
			Status: &backupv1.VirtualMachineBackupStatus{Offline: pointer.P(true)},
		}
	}

	vmWithVolumes := func() *v1.VirtualMachine {
		return &v1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{Name: vmName, Namespace: namespace},
			Spec: v1.VirtualMachineSpec{
				Template: &v1.VirtualMachineInstanceTemplateSpec{
					Spec: v1.VirtualMachineInstanceSpec{
						Volumes: []v1.Volume{
							{Name: "rootdisk", VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{}}},
							{Name: "cloudinit", VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{}}},
						},
					},
				},
			},
		}
	}

	Describe("isOfflineBackup", func() {
		It("is true only when status.offline is set", func() {
			Expect(isOfflineBackup(offlineBackup())).To(BeTrue())

			b := offlineBackup()
			b.Status.Offline = nil
			Expect(isOfflineBackup(b)).To(BeFalse())

			b.Status = nil
			Expect(isOfflineBackup(b)).To(BeFalse())
		})
	})

	Describe("shouldReconcileOffline", func() {
		newController := func(offlineEnabled bool) *VMBackupController {
			featureGates := []string{}
			if offlineEnabled {
				featureGates = []string{"IncrementalBackup", "OfflineIncrementalBackup"}
			}
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
				DeveloperConfiguration: &v1.DeveloperConfiguration{FeatureGates: featureGates},
			})
			return &VMBackupController{clusterConfig: config}
		}

		It("latches to offline once status.offline is set regardless of gate or VMI", func() {
			ctrl := newController(false)
			Expect(ctrl.shouldReconcileOffline(offlineBackup(), true, nil)).To(BeTrue())
		})

		It("is false when the feature gate is disabled", func() {
			ctrl := newController(false)
			b := offlineBackup()
			b.Status = &backupv1.VirtualMachineBackupStatus{}
			Expect(ctrl.shouldReconcileOffline(b, false, nil)).To(BeFalse())
		})

		It("is true for a fresh backup with no running VMI when enabled", func() {
			ctrl := newController(true)
			b := offlineBackup()
			b.Status = &backupv1.VirtualMachineBackupStatus{}
			Expect(ctrl.shouldReconcileOffline(b, false, nil)).To(BeTrue())
		})

		It("is false when a VMI exists", func() {
			ctrl := newController(true)
			b := offlineBackup()
			b.Status = &backupv1.VirtualMachineBackupStatus{}
			Expect(ctrl.shouldReconcileOffline(b, true, nil)).To(BeFalse())
		})
	})

	Describe("offlineEligibleVolumes", func() {
		It("returns only CBT-eligible volumes", func() {
			volumes := offlineEligibleVolumes(vmWithVolumes())
			Expect(volumes).To(ConsistOf(backupv1.BackupVolumeInfo{VolumeName: "rootdisk"}))
		})

		It("returns nil when the template is absent", func() {
			Expect(offlineEligibleVolumes(&v1.VirtualMachine{})).To(BeNil())
		})
	})

	Describe("offlineCheckpointName", func() {
		It("derives the checkpoint from the backup name", func() {
			b := offlineBackup()
			Expect(offlineCheckpointName(b)).To(Equal("backup-backup"))
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

	Describe("otherOfflineBackupInProgress", func() {
		It("finds another in-progress offline backup for the same VM", func() {
			informer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
			other := offlineBackup()
			other.Name = "other"
			setProgressing(other)
			Expect(informer.GetStore().Add(other)).To(Succeed())

			ctrl := &VMBackupController{backupInformer: informer}
			Expect(ctrl.otherOfflineBackupInProgress(offlineBackup(), vmName)).To(Equal("other"))
		})

		It("returns empty when no other backup is in progress", func() {
			informer, _ := testutils.NewFakeInformerFor(&backupv1.VirtualMachineBackup{})
			ctrl := &VMBackupController{backupInformer: informer}
			Expect(ctrl.otherOfflineBackupInProgress(offlineBackup(), vmName)).To(BeEmpty())
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
				Offline:         pointer.P(true),
				CheckpointName:  pointer.P("backup-backup"),
				IncludedVolumes: []backupv1.BackupVolumeInfo{{VolumeName: "rootdisk"}},
			},
		}
	}

	// The advance path makes exactly one client call, a tracker patch, so its
	// presence or absence tells us whether the tracker advanced.
	expectTrackerPatch := func() {
		virtClient.EXPECT().VirtualMachineBackupTracker(testNamespace).
			Return(kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace))
	}

	latestCheckpointName := func() string {
		updated, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Get(
			context.Background(), trackerName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		if updated.Status == nil || updated.Status.LatestCheckpoint == nil {
			return ""
		}
		return updated.Status.LatestCheckpoint.Name
	}

	// Seeded with an existing checkpoint so an advance is observable as a name change.
	seedTracker := func() *backupv1.VirtualMachineBackupTracker {
		tracker := createTracker(trackerName, "testvm", true, false)
		_, err := kubevirtCli.BackupV1alpha1().VirtualMachineBackupTrackers(testNamespace).Create(
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
		virtClient.EXPECT().VirtualMachineExport(testNamespace).
			Return(kubevirtCli.ExportV1().VirtualMachineExports(testNamespace)).AnyTimes()
		ctrl = &VMBackupController{
			client:        virtClient,
			vmStore:       vmInformer.GetStore(),
			vmExportStore: exportInformer.GetStore(),
			recorder:      record.NewFakeRecorder(100),
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

	Describe("empty incremental rejection", func() {
		emptyExport := func(backup *backupv1.VirtualMachineBackup) {
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
						{Type: exportv1.ConditionReady, Status: corev1.ConditionFalse, Reason: "EmptyIncremental"},
					},
				},
			}
			Expect(exportInformer.GetStore().Add(export)).To(Succeed())
		}

		expectEmptyIncrementalFailure := func(backup *backupv1.VirtualMachineBackup) {
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))
			Expect(isBackupFailed(backup)).To(BeTrue())
			Expect(isBackupComplete(backup)).To(BeFalse())
			failed := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionFailed))
			Expect(failed).ToNot(BeNil())
			Expect(failed.Reason).To(Equal(backupv1.ReasonEmptyIncremental))
		}

		It("fails a pull backup without advancing when the export reports an empty delta", func() {
			tracker := seedTracker()
			backup := pullBackup()
			backup.CreationTimestamp = metav1.NewTime(time.Now())
			emptyExport(backup)

			Expect(ctrl.reconcileOfflinePull(backup, tracker)).To(Succeed())
			expectEmptyIncrementalFailure(backup)
		})

		It("fails a push backup without advancing when the export reports an empty delta", func() {
			tracker := seedTracker()
			backup := pullBackup()
			backup.Spec.Mode = pointer.P(backupv1.PushMode)
			emptyExport(backup)

			Expect(ctrl.reconcileOfflinePush(backup, tracker)).To(Succeed())
			expectEmptyIncrementalFailure(backup)
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

	Describe("reconcileOffline (VM deleted mid-backup)", func() {
		It("fails the backup and cleans up the export when the VM no longer exists", func() {
			tracker := seedTracker()
			backup := pullBackup()
			setProgressing(backup)

			export := &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backup.Name,
					Namespace: backup.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(backup, backupv1.VirtualMachineBackupGroupVersionKind),
					},
				},
			}
			Expect(exportInformer.GetStore().Add(export)).To(Succeed())
			_, err := kubevirtCli.ExportV1().VirtualMachineExports(testNamespace).Create(context.Background(), export, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(ctrl.reconcileOffline(backup, tracker, "testvm", false)).To(Succeed())

			Expect(isBackupFailed(backup)).To(BeTrue())
			failed := meta.FindStatusCondition(backup.Status.Conditions, string(backupv1.ConditionFailed))
			Expect(failed).ToNot(BeNil())
			Expect(failed.Reason).To(Equal(backupv1.ReasonVMDeleted))
			Expect(latestCheckpointName()).To(Equal("checkpoint-1"))

			_, err = kubevirtCli.ExportV1().VirtualMachineExports(testNamespace).Get(context.Background(), backup.Name, metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("proceeds normally when the VM still exists", func() {
			tracker := seedTracker()
			backup := pullBackup()
			backup.CreationTimestamp = metav1.NewTime(time.Now())
			Expect(vmInformer.GetStore().Add(&v1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: testNamespace},
			})).To(Succeed())

			Expect(ctrl.reconcileOffline(backup, tracker, "testvm", false)).To(Succeed())

			Expect(isBackupFailed(backup)).To(BeFalse())
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
