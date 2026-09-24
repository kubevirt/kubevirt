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

package export

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1 "kubevirt.io/api/backup/v1alpha1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("Offline backup source", func() {
	const namespace = "default"

	dataPVC := func(name string, mode corev1.PersistentVolumeMode) *corev1.PersistentVolumeClaim {
		return &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec:       corev1.PersistentVolumeClaimSpec{VolumeMode: &mode},
		}
	}

	statePVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "persistent-state-testvm", Namespace: namespace},
	}

	targetPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "backup-target", Namespace: namespace},
	}

	offlineBackup := func(backupType backupv1.BackupType) *backupv1.VirtualMachineBackup {
		return &backupv1.VirtualMachineBackup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup", Namespace: namespace, UID: "backup-uid"},
			Status: &backupv1.VirtualMachineBackupStatus{
				Offline:        pointer.P(true),
				Type:           backupType,
				CheckpointName: pointer.P("backup-backup"),
				IncludedVolumes: []backupv1.BackupVolumeInfo{
					{VolumeName: "rootdisk"},
				},
			},
		}
	}

	pushBackup := func(backupType backupv1.BackupType) *backupv1.VirtualMachineBackup {
		b := offlineBackup(backupType)
		b.Spec.Mode = pointer.P(backupv1.PushMode)
		b.Spec.PvcName = pointer.P("backup-target")
		return b
	}

	newPod := func() *corev1.Pod {
		return &corev1.Pod{
			Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "exporter"}}},
		}
	}

	envValue := func(container *corev1.Container, name string) string {
		for _, e := range container.Env {
			if e.Name == name {
				return e.Value
			}
		}
		return ""
	}

	Describe("isOfflineBackup", func() {
		It("is true only when status.offline is set", func() {
			Expect(isOfflineBackup(offlineBackup(backupv1.Full))).To(BeTrue())

			b := offlineBackup(backupv1.Full)
			b.Status.Offline = nil
			Expect(isOfflineBackup(b)).To(BeFalse())

			b.Status = nil
			Expect(isOfflineBackup(b)).To(BeFalse())
		})
	})

	Describe("offlineDataMountPath", func() {
		It("uses the filesystem mount root for filesystem PVCs", func() {
			pvc := dataPVC("data", corev1.PersistentVolumeFilesystem)
			Expect(offlineDataMountPath(pvc)).To(HavePrefix(fileSystemMountPath))
		})

		It("uses the block mount root for block PVCs", func() {
			pvc := dataPVC("data", corev1.PersistentVolumeBlock)
			Expect(offlineDataMountPath(pvc)).To(HavePrefix(blockVolumeMountPath))
		})
	})

	Describe("ConfigurePod", func() {
		dataPVCs := func() map[string]*corev1.PersistentVolumeClaim {
			return map[string]*corev1.PersistentVolumeClaim{"rootdisk": dataPVC("rootdisk-pvc", corev1.PersistentVolumeFilesystem)}
		}

		It("mounts the state PVC read-write and a sockets emptyDir", func() {
			source := NewOfflineVMBackupSource(offlineBackup(backupv1.Full), "testvm", dataPVCs(), statePVC, nil, "")
			pod := newPod()
			source.ConfigurePod(pod)

			container := &pod.Spec.Containers[0]
			Expect(container.VolumeMounts).To(ContainElement(HaveField("MountPath", offlineStateMountPath)))
			Expect(container.VolumeMounts).To(ContainElement(HaveField("MountPath", offlineSocketDir)))
			Expect(pod.Spec.Volumes).To(ContainElement(HaveField("Name", offlineSocketVolume)))
		})

		It("sets the offline environment for a full backup", func() {
			source := NewOfflineVMBackupSource(offlineBackup(backupv1.Full), "testvm", dataPVCs(), statePVC, nil, "")
			pod := newPod()
			source.ConfigurePod(pod)

			container := &pod.Spec.Containers[0]
			Expect(envValue(container, "OFFLINE_BACKUP")).To(Equal("true"))
			Expect(envValue(container, "BACKUP_STATE_PATH")).To(Equal(offlineStateMountPath))
			Expect(envValue(container, "SOCKET_DIR")).To(Equal(offlineSocketDir))
			Expect(envValue(container, "BACKUP_TYPE")).To(Equal(string(backupv1.Full)))
			Expect(envValue(container, "BACKUP_CHECKPOINT")).To(Equal("backup-backup"))
			Expect(envValue(container, "BACKUP_UID")).To(Equal("backup-uid"))
			Expect(envValue(container, "BACKUP_BASE_CHECKPOINT")).To(BeEmpty())
			Expect(envValue(container, "BACKUP_MODE")).To(BeEmpty())
			Expect(envValue(container, "BACKUP0_BACKUP_PATH")).To(Equal("rootdisk"))
			Expect(envValue(container, "BACKUP0_DISK_PATH")).ToNot(BeEmpty())
		})

		It("sets the base checkpoint for an incremental backup", func() {
			source := NewOfflineVMBackupSource(offlineBackup(backupv1.Incremental), "testvm", dataPVCs(), statePVC, nil, "backup-previous")
			pod := newPod()
			source.ConfigurePod(pod)

			Expect(envValue(&pod.Spec.Containers[0], "BACKUP_BASE_CHECKPOINT")).To(Equal("backup-previous"))
		})

		It("skips volumes without a resolved data PVC", func() {
			source := NewOfflineVMBackupSource(offlineBackup(backupv1.Full), "testvm", map[string]*corev1.PersistentVolumeClaim{}, statePVC, nil, "")
			pod := newPod()
			source.ConfigurePod(pod)

			Expect(envValue(&pod.Spec.Containers[0], "BACKUP0_BACKUP_PATH")).To(BeEmpty())
		})

		It("mounts the target PVC and sets push env in push mode", func() {
			backup := pushBackup(backupv1.Full)
			backup.CreationTimestamp = metav1.Date(2026, 9, 18, 10, 30, 0, 0, metav1.Now().Location())
			source := NewOfflineVMBackupSource(backup, "testvm", dataPVCs(), statePVC, targetPVC, "")
			pod := newPod()
			source.ConfigurePod(pod)

			container := &pod.Spec.Containers[0]
			Expect(container.VolumeMounts).To(ContainElement(HaveField("MountPath", offlineTargetMountPath)))
			Expect(envValue(container, "BACKUP_MODE")).To(Equal(string(backupv1.PushMode)))
			Expect(envValue(container, "BACKUP_NAME")).To(Equal("backup"))
			Expect(envValue(container, "BACKUP_TARGET_DIR")).To(HavePrefix(offlineTargetMountPath + "/testvm/backup-"))
		})

		It("does not mount the target PVC in pull mode", func() {
			source := NewOfflineVMBackupSource(offlineBackup(backupv1.Full), "testvm", dataPVCs(), statePVC, targetPVC, "")
			pod := newPod()
			source.ConfigurePod(pod)

			Expect(pod.Spec.Volumes).ToNot(ContainElement(HaveField("Name", offlineTargetVolume)))
			Expect(envValue(&pod.Spec.Containers[0], "BACKUP_TARGET_DIR")).To(BeEmpty())
		})
	})
})
