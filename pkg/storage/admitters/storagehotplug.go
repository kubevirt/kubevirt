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

package admitters

import (
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// AdmitHotplugStorage compares the old and new volumes and disks, and ensures that they match and are valid.
func AdmitHotplugStorage(newVolumes, oldVolumes []v1.Volume, newDisks, oldDisks []v1.Disk, volumeStatuses []v1.VolumeStatus, newVMI *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig) *admissionv1.AdmissionResponse {
	if err := validateExpectedDisksAndFilesystems(newVolumes, newDisks, newVMI.Spec.Domain.Devices.Filesystems, config); err != nil {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: err.Error(),
			},
		})
	}
	newHotplugVolumeMap := getHotplugVolumes(newVolumes, volumeStatuses)
	newPermanentVolumeMap := getPermanentVolumes(newVolumes, volumeStatuses)
	oldHotplugVolumeMap := getHotplugVolumes(oldVolumes, volumeStatuses)
	oldPermanentVolumeMap := getPermanentVolumes(oldVolumes, volumeStatuses)
	migratedVolumeMap := getMigratedVolumeMaps(newVMI.Status.MigratedVolumes)

	newDiskMap := getDiskMap(newDisks)
	oldDiskMap := getDiskMap(oldDisks)

	permanentAr := verifyPermanentVolumes(newPermanentVolumeMap, oldPermanentVolumeMap, newDiskMap, oldDiskMap, migratedVolumeMap)
	if permanentAr != nil {
		return permanentAr
	}

	hotplugAr := verifyHotplugVolumes(newHotplugVolumeMap, oldHotplugVolumeMap, newDiskMap, oldDiskMap, migratedVolumeMap)
	if hotplugAr != nil {
		return hotplugAr
	}

	return nil
}

func ValidateHotplugDiskConfiguration(disk *v1.Disk, name, messagePrefix, field string) []metav1.StatusCause {
	if disk == nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s for [%s] requires the disk field to be set.", messagePrefix, name),
			Field:   field,
		}}
	}

	bus := getDiskBus(*disk)
	switch {
	case disk.DiskDevice.Disk != nil:
		if bus != v1.DiskBusSCSI && bus != v1.DiskBusVirtio {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s for disk [%s] requires bus to be 'scsi' or 'virtio'. [%s] is not permitted.", messagePrefix, name, bus),
				Field:   field,
			}}
		}
	case disk.DiskDevice.LUN != nil:
		if bus != v1.DiskBusSCSI {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("%s for LUN [%s] requires bus to be 'scsi'. [%s] is not permitted.", messagePrefix, name, bus),
				Field:   field,
			}}
		}
	default:
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s for [%s] requires diskDevice of type 'disk' or 'lun' to be used.", messagePrefix, name),
			Field:   field,
		}}
	}

	if disk.DedicatedIOThread != nil && *disk.DedicatedIOThread && bus != v1.DiskBusVirtio {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("%s for [%s] requires virtio bus for IOThreads.", messagePrefix, name),
			Field:   field,
		}}
	}

	// Validate boot order
	if disk.BootOrder != nil {
		order := *disk.BootOrder
		if order < 1 {
			return []metav1.StatusCause{{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.domain.devices.disks[1] must have a boot order > 0, if supplied"),
				Field:   fmt.Sprintf("spec.domain.devices.disks[1].bootOrder"),
			}}
		}
	}

	return nil
}

func validateExpectedDisksAndFilesystems(volumes []v1.Volume, disks []v1.Disk, filesystems []v1.Filesystem, config *virtconfig.ClusterConfig) error {
	names := make(map[string]struct{})
	for _, volume := range volumes {
		if volume.MemoryDump == nil {
			names[volume.Name] = struct{}{}
		}
	}

	requiredVolumes := len(filesystems)
	for _, disk := range disks {
		_, ok := names[disk.Name]
		// it's okay for a CDRom to not be mapped to a volume
		if !ok && disk.CDRom != nil && config.DeclarativeHotplugVolumesEnabled() {
			continue
		}
		requiredVolumes++
	}

	// Make sure volume is not mapped to multiple disks/filesystems or volume mapped to nothing
	if requiredVolumes != len(names) {
		return fmt.Errorf("mismatch between volumes declared (%d) and required (%d)", len(names), requiredVolumes)
	}

	return nil
}

func verifyHotplugVolumes(newHotplugVolumeMap, oldHotplugVolumeMap map[string]v1.Volume, newDisks, oldDisks map[string]v1.Disk,
	migratedVols map[string]bool) *admissionv1.AdmissionResponse {
	for k, v := range newHotplugVolumeMap {
		if _, ok := oldHotplugVolumeMap[k]; ok {
			_, okMigVol := migratedVols[k]
			// New and old have same volume, ensure they are the same
			if !equality.Semantic.DeepEqual(v, oldHotplugVolumeMap[k]) && !okMigVol {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("hotplug volume %s, changed", k),
					},
				})
			}
			if v.MemoryDump == nil {
				if _, ok := newDisks[k]; !ok {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("volume %s doesn't have a matching disk", k),
						},
					})
				}
				if !equality.Semantic.DeepEqual(newDisks[k], oldDisks[k]) {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("hotplug disk %s, changed", k),
						},
					})
				}
			}
		} else {
			// This is a new volume, ensure that the volume is either DV, PVC or memoryDumpVolume
			if v.DataVolume == nil && v.PersistentVolumeClaim == nil && v.MemoryDump == nil {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("volume %s is not a PVC or DataVolume", k),
					},
				})
			}
			if v.MemoryDump == nil {
				// Also ensure the matching new disk exists and has a valid bus
				if _, ok := newDisks[k]; !ok {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("disk %s does not exist", k),
						},
					})
				}
				disk := newDisks[k]
				if _, ok := oldDisks[k]; !ok {
					causes := ValidateHotplugDiskConfiguration(&disk, k, "Hotplug configuration", "")
					if len(causes) > 0 {
						return webhookutils.ToAdmissionResponse(causes)
					}
				}
			}
		}
	}
	return nil
}

func isMigratedVolume(newVol, oldVol *v1.Volume, migratedVolumeMap map[string]bool) bool {
	if newVol.Name != oldVol.Name {
		return false
	}
	_, ok := migratedVolumeMap[newVol.Name]
	return ok
}

func verifyPermanentVolumes(newPermanentVolumeMap, oldPermanentVolumeMap map[string]v1.Volume, newDisks, oldDisks map[string]v1.Disk, migratedVolumeMap map[string]bool) *admissionv1.AdmissionResponse {
	if len(newPermanentVolumeMap) != len(oldPermanentVolumeMap) {
		// Removed one of the permanent volumes, reject admission.
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Number of permanent volumes has changed",
			},
		})
	}

	// Ensure we didn't modify any permanent volumes
	for k, v := range newPermanentVolumeMap {
		// Know at this point the new old and permanent have the same count.
		if _, ok := oldPermanentVolumeMap[k]; !ok {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("permanent volume %s, not found", k),
				},
			})
		}
		oldVol := oldPermanentVolumeMap[k]
		if isMigratedVolume(&v, &oldVol, migratedVolumeMap) {
			continue
		}
		if !equality.Semantic.DeepEqual(v, oldVol) {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("permanent volume %s, changed", k),
				},
			})
		}
		if !equality.Semantic.DeepEqual(newDisks[k], oldDisks[k]) {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("permanent disk %s, changed", k),
				},
			})
		}
	}
	return nil
}

func getDiskMap(disks []v1.Disk) map[string]v1.Disk {
	newDiskMap := make(map[string]v1.Disk, 0)
	for _, disk := range disks {
		if disk.Name != "" {
			newDiskMap[disk.Name] = disk
		}
	}
	return newDiskMap
}

func getHotplugVolumes(volumes []v1.Volume, volumeStatuses []v1.VolumeStatus) map[string]v1.Volume {
	permanentVolumesFromStatus := make(map[string]v1.Volume, 0)
	for _, volume := range volumeStatuses {
		if volume.HotplugVolume == nil {
			permanentVolumesFromStatus[volume.Name] = v1.Volume{}
		}
	}
	permanentVolumes := make(map[string]v1.Volume, 0)
	for _, volume := range volumes {
		if _, ok := permanentVolumesFromStatus[volume.Name]; !ok {
			permanentVolumes[volume.Name] = volume
		}
	}
	return permanentVolumes
}

func getPermanentVolumes(volumes []v1.Volume, volumeStatuses []v1.VolumeStatus) map[string]v1.Volume {
	permanentVolumesFromStatus := make(map[string]v1.Volume, 0)
	for _, volume := range volumeStatuses {
		if volume.HotplugVolume == nil {
			permanentVolumesFromStatus[volume.Name] = v1.Volume{}
		}
	}
	permanentVolumes := make(map[string]v1.Volume, 0)
	for _, volume := range volumes {
		if _, ok := permanentVolumesFromStatus[volume.Name]; ok {
			permanentVolumes[volume.Name] = volume
		}
	}
	return permanentVolumes
}

func getMigratedVolumeMaps(migratedDisks []v1.StorageMigratedVolumeInfo) map[string]bool {
	volumes := make(map[string]bool)
	for _, v := range migratedDisks {
		volumes[v.VolumeName] = true
	}
	return volumes
}

func validateUtilityVolumes(field *field.Path, spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if len(spec.UtilityVolumes) > 0 && !config.UtilityVolumesEnabled() {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "UtilityVolumes feature gate is not enabled",
			Field:   field.Child("utilityVolumes").String(),
		})
		return causes
	}

	volumeNameMap := make(map[string]int)
	for idx, volume := range spec.Volumes {
		volumeNameMap[volume.Name] = idx
	}

	utilityVolumeNameMap := make(map[string]int)
	for idx, utilityVolume := range spec.UtilityVolumes {

		if utilityVolume.Name == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "UtilityVolume 'name' must be set",
				Field:   field.Child("utilityVolumes").Index(idx).Child("name").String(),
			})
		}

		if otherIdx, exists := utilityVolumeNameMap[utilityVolume.Name]; exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("%s and %s must not have the same Name.", field.Child("utilityVolumes").Index(idx).String(), field.Child("utilityVolumes").Index(otherIdx).String()),
				Field:   field.Child("utilityVolumes").Index(idx).Child("name").String(),
			})
		} else {
			utilityVolumeNameMap[utilityVolume.Name] = idx
		}

		// Check for conflicts with regular volume names
		if regularVolumeIdx, exists := volumeNameMap[utilityVolume.Name]; exists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueDuplicate,
				Message: fmt.Sprintf("%s conflicts with %s. UtilityVolume names must be unique across both volumes and utilityVolumes.", field.Child("utilityVolumes").Index(idx).String(), field.Child("volumes").Index(regularVolumeIdx).String()),
				Field:   field.Child("utilityVolumes").Index(idx).Child("name").String(),
			})
		}

		// Validate PVC claim name is set
		if utilityVolume.PersistentVolumeClaimVolumeSource.ClaimName == "" {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueRequired,
				Message: "UtilityVolume PVC 'claimName' must be set",
				Field:   field.Child("utilityVolumes").Index(idx).Child("claimName").String(),
			})
		}
	}

	return causes
}

func AdmitUtilityVolumes(newSpec, oldSpec *v1.VirtualMachineInstanceSpec, volumeStatuses []v1.VolumeStatus, config *virtconfig.ClusterConfig) *admissionv1.AdmissionResponse {
	if causes := validateUtilityVolumes(field.NewPath("spec"), newSpec, config); len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	if oldSpec == nil {
		return nil
	}

	// Ensure utility volumes are never treated as permanent volumes
	// They should always be hotplug volumes
	permanentVolumesFromStatus := make(map[string]bool)
	for _, volumeStatus := range volumeStatuses {
		if volumeStatus.HotplugVolume == nil {
			permanentVolumesFromStatus[volumeStatus.Name] = true
		}
	}

	// Check that no utility volumes are marked as permanent
	for _, utilityVolume := range newSpec.UtilityVolumes {
		if permanentVolumesFromStatus[utilityVolume.Name] {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("utility volume %s cannot be a permanent volume, utility volumes must always be hotplug", utilityVolume.Name),
				},
			})
		}
	}

	newUtilityVolumeMap := getUtilityVolumeMap(newSpec.UtilityVolumes)
	oldUtilityVolumeMap := getUtilityVolumeMap(oldSpec.UtilityVolumes)

	for name, newVolume := range newUtilityVolumeMap {
		if oldVolume, exists := oldUtilityVolumeMap[name]; exists {
			// Existing utility volume, ensure it hasn't changed
			if !equality.Semantic.DeepEqual(newVolume, oldVolume) {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("utility volume %s has changed", name),
					},
				})
			}
		}
	}

	return nil
}

func getUtilityVolumeMap(utilityVolumes []v1.UtilityVolume) map[string]v1.UtilityVolume {
	utilityVolumeMap := make(map[string]v1.UtilityVolume)
	for _, volume := range utilityVolumes {
		utilityVolumeMap[volume.Name] = volume
	}
	return utilityVolumeMap
}

func ValidateUtilityVolumesNotPresentOnCreation(field *field.Path, spec *v1.VirtualMachineInstanceSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.UtilityVolumes != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "cannot create VMI with utility volumes in spec, utility volumes can only be added via hotplug",
			Field:   field.Child("utilityVolumes").String(),
		})
	}

	return causes
}
