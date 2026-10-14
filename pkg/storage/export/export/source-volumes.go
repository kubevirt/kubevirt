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
	"fmt"
	"path"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	exportv1 "kubevirt.io/api/export/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/types"
)

type sourceVolumes struct {
	volumes         []sourceVolume
	inUse           bool
	isPopulated     bool
	readyCondition  exportv1.Condition
	sourceCondition exportv1.Condition
}

type sourceVolume struct {
	pvc                 *corev1.PersistentVolumeClaim
	kubevirtContentType bool
}

func (sv *sourceVolumes) isSourceAvailable() bool {
	return !sv.inUse && sv.isPopulated && len(sv.duplicatePVCNames()) == 0
}

func (sv *sourceVolumes) duplicatePVCNames() []string {
	seen, duplicates := sets.New[string](), sets.New[string]()
	for _, volume := range sv.volumes {
		if volume.pvc == nil {
			continue
		}
		if seen.Has(volume.pvc.Name) {
			duplicates.Insert(volume.pvc.Name)
		}
		seen.Insert(volume.pvc.Name)
	}
	return sets.List(duplicates)
}

func (sv *sourceVolumes) hasContent() bool {
	return len(sv.volumes) > 0
}

func (sv *sourceVolumes) ReadyCondition() exportv1.Condition {
	if duplicates := sv.duplicatePVCNames(); len(duplicates) > 0 {
		return newReadyCondition(corev1.ConditionFalse, duplicatePVCReason,
			fmt.Sprintf("Source references the same PersistentVolumeClaim from more than one volume: %s",
				strings.Join(duplicates, ", ")))
	}
	return sv.readyCondition
}

func (sv *sourceVolumes) configurePodVolumes(podManifest *corev1.Pod) {
	for i, volume := range sv.volumes {
		var mountPoint string
		pvc := volume.pvc
		volumeName := fmt.Sprintf("vol%d", i)
		if types.IsPVCBlock(pvc.Spec.VolumeMode) {
			mountPoint = fmt.Sprintf("%s/%s", blockVolumeMountPath, volumeName)
			podManifest.Spec.Containers[0].VolumeDevices = append(podManifest.Spec.Containers[0].VolumeDevices, corev1.VolumeDevice{
				Name:       volumeName,
				DevicePath: mountPoint,
			})
		} else {
			mountPoint = fmt.Sprintf("%s/%s", fileSystemMountPath, volumeName)
			podManifest.Spec.Containers[0].VolumeMounts = append(podManifest.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
				Name:      volumeName,
				ReadOnly:  true,
				MountPath: mountPoint,
			})
		}
		podManifest.Spec.Volumes = append(podManifest.Spec.Volumes, corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.Name,
				},
			},
		})
		addVolumeEnvironmentVariables(&podManifest.Spec.Containers[0], pvc, i, mountPoint, volume.kubevirtContentType)
	}
}

type volumeNamer func(pvcName string) string

func defaultVolumeNamer(pvcName string) string {
	return pvcName
}

func (sv *sourceVolumes) populateLink(
	exportLink *exportv1.VirtualMachineExportLink,
	paths *ServerPaths,
	pod *corev1.Pod,
	hostAndBase, scheme string,
	namer volumeNamer,
) {
	if pod.Status.Phase != corev1.PodRunning {
		return
	}

	for _, volume := range sv.volumes {
		pvc := volume.pvc
		if pvc == nil {
			continue
		}

		volumeInfo := paths.GetVolumeInfo(pvc.Name)
		if volumeInfo == nil {
			log.Log.Warningf("Volume %s not found in paths", pvc.Name)
			continue
		}

		ev := exportv1.VirtualMachineExportVolume{
			Name: namer(pvc.Name),
		}

		if volumeInfo.RawURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtRaw,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.RawURI),
			})
		}
		if volumeInfo.RawGzURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.KubeVirtGz,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.RawGzURI),
			})
		}
		if volumeInfo.DirURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.Dir,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.DirURI),
			})
		}
		if volumeInfo.ArchiveURI != "" {
			ev.Formats = append(ev.Formats, exportv1.VirtualMachineExportVolumeFormat{
				Format: exportv1.ArchiveGz,
				Url:    scheme + path.Join(hostAndBase, volumeInfo.ArchiveURI),
			})
		}

		if len(ev.Formats) == 0 {
			log.Log.Warningf("No formats found for volume %s", pvc.Name)
			continue
		}

		exportLink.Volumes = append(exportLink.Volumes, ev)
	}
}

func addVolumeEnvironmentVariables(exportContainer *corev1.Container, pvc *corev1.PersistentVolumeClaim, index int, mountPoint string, isKubevirt bool) {
	exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
		Name:  fmt.Sprintf("VOLUME%d_EXPORT_PATH", index),
		Value: mountPoint,
	})
	if types.IsPVCBlock(pvc.Spec.VolumeMode) {
		exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
			Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_URI", index),
			Value: rawURI(pvc),
		}, corev1.EnvVar{
			Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_GZIP_URI", index),
			Value: rawGzipURI(pvc),
		})
	} else {
		if isKubevirt {
			exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_URI", index),
				Value: rawURI(pvc),
			}, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_RAW_GZIP_URI", index),
				Value: rawGzipURI(pvc),
			})
		} else {
			exportContainer.Env = append(exportContainer.Env, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_ARCHIVE_URI", index),
				Value: archiveURI(pvc),
			}, corev1.EnvVar{
				Name:  fmt.Sprintf("VOLUME%d_EXPORT_DIR_URI", index),
				Value: dirURI(pvc),
			})
		}
	}
}
