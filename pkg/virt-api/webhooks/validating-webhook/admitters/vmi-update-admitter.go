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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package admitters

import (
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	v1 "kubevirt.io/api/core/v1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

const nodeNameExtraInfo = "authentication.kubernetes.io/node-name"

type VMIUpdateAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (admitter *VMIUpdateAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if admitter.ClusterConfig.NodeRestrictionEnabled() && webhooks.IsComponentServiceAccount(ar.Request.UserInfo.Username, webhooks.GetNamespace(), components.HandlerServiceAccountName) {
		values, exist := ar.Request.UserInfo.Extra[nodeNameExtraInfo]
		if exist && len(values) > 0 {
			nodeName := values[0]
			sourceNode := oldVMI.Status.NodeName
			targetNode := ""
			if oldVMI.Status.MigrationState != nil {
				targetNode = oldVMI.Status.MigrationState.TargetNode
			}

			// Check that source or target is making this request
			if nodeName != sourceNode && (targetNode == "" || nodeName != targetNode) {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "Node restriction, virt-handler is only allowed to modify VMIs it owns",
					},
				})
			}

			// Check that handler is not setting target
			if targetNode == "" && newVMI.Status.MigrationState != nil && newVMI.Status.MigrationState.TargetNode != targetNode {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "Node restriction, virt-handler is not allowed to set target node",
					},
				})
			}
		} else {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Node restriction failed, virt-handler service account is missing node name",
				},
			})
		}
	}

	// Reject VMI update if VMI spec changed
	if !equality.Semantic.DeepEqual(newVMI.Spec, oldVMI.Spec) {
		// Only allow the KubeVirt SA to modify the VMI spec, since that means it went through the sub resource.
		if webhooks.IsKubeVirtServiceAccount(ar.Request.UserInfo.Username) {
			hotplugResponse := admitHotplug(oldVMI, newVMI, admitter.ClusterConfig)
			if hotplugResponse != nil {
				return hotplugResponse
			}
		} else {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "update of VMI object is restricted",
				},
			})
		}
	}

	if reviewResponse := admitVMILabelsUpdate(newVMI, oldVMI, ar); reviewResponse != nil {
		return reviewResponse
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnDeprecatedAPIs(&newVMI.Spec, admitter.ClusterConfig),
	}
}

func getExpectedDisksAndFilesystems(newVolumes []v1.Volume) int {
	numMemoryDumpVolumes := 0
	for _, volume := range newVolumes {
		if volume.MemoryDump != nil {
			numMemoryDumpVolumes = numMemoryDumpVolumes + 1
		}
	}
	return len(newVolumes) - numMemoryDumpVolumes
}

// admitHotplugStorage compares the old and new volumes and disks, and ensures that they match and are valid.
func admitHotplugStorage(newVolumes, oldVolumes []v1.Volume, newDisks, oldDisks []v1.Disk, volumeStatuses []v1.VolumeStatus, newVMI *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig) *admissionv1.AdmissionResponse {
	expectedDisksAndFilesystems := getExpectedDisksAndFilesystems(newVolumes)
	observedDisksAndFilesystems := len(newDisks) + len(newVMI.Spec.Domain.Devices.Filesystems)
	if expectedDisksAndFilesystems != observedDisksAndFilesystems {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("number of disks and filesystems (%d) does not equal the number of volumes (%d)", observedDisksAndFilesystems, expectedDisksAndFilesystems),
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

	hotplugAr := verifyHotplugVolumes(newHotplugVolumeMap, oldHotplugVolumeMap, newDiskMap, oldDiskMap)
	if hotplugAr != nil {
		return hotplugAr
	}

	causes := ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &newVMI.Spec, config)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return nil
}

func verifyHotplugVolumes(newHotplugVolumeMap, oldHotplugVolumeMap map[string]v1.Volume, newDisks, oldDisks map[string]v1.Disk) *admissionv1.AdmissionResponse {
	for k, v := range newHotplugVolumeMap {
		if _, ok := oldHotplugVolumeMap[k]; ok {
			// New and old have same volume, ensure they are the same
			if !equality.Semantic.DeepEqual(v, oldHotplugVolumeMap[k]) {
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
							Message: fmt.Sprintf("Volume %s doesn't have a matching disk", k),
						},
					})
				}

				if !equalDiskIgnoreSerial(newDisks[k], oldDisks[k]) {
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
			if v.DataVolume == nil && v.PersistentVolumeClaim == nil && v.MemoryDump == nil && v.ContainerDisk == nil {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("volume %s is not a PVC,DataVolume,MemoryDumpVolume or ContainerDisk", k),
					},
				})
			}
			if v.MemoryDump == nil {
				// Also ensure the matching new disk exists and is of type scsi
				if _, ok := newDisks[k]; !ok {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("Disk %s does not exist", k),
						},
					})
				}
				disk := newDisks[k]
				if disk.Disk == nil && disk.LUN == nil && disk.CDRom == nil {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("Disk %s requires diskDevice of type 'disk','lun' or cdrom to be hotplugged.", k),
						},
					})
				}
				if (disk.Disk != nil && disk.Disk.Bus != v1.DiskBusSCSI) || (disk.LUN != nil && disk.LUN.Bus != v1.DiskBusSCSI) || (disk.CDRom != nil && disk.CDRom.Bus != v1.DiskBusSCSI) {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("hotplugged Disk %s does not use a %q bus", k, v1.DiskBusSCSI),
						},
					})

				}
				if disk.DedicatedIOThread != nil && *disk.DedicatedIOThread {
					return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
						{
							Type:    metav1.CauseTypeFieldValueInvalid,
							Message: fmt.Sprintf("hotplugged Disk %s can't use dedicated IOThread: scsi bus is unsupported.", k),
						},
					})
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

		if !equalDiskIgnoreSerial(newDisks[k], oldDisks[k]) {
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

func admitVMILabelsUpdate(
	newVMI *v1.VirtualMachineInstance,
	oldVMI *v1.VirtualMachineInstance,
	ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {

	if webhooks.IsKubeVirtServiceAccount(ar.Request.UserInfo.Username) {
		return nil
	}

	oldLabels := filterKubevirtLabels(oldVMI.ObjectMeta.Labels)
	newLabels := filterKubevirtLabels(newVMI.ObjectMeta.Labels)

	if !equality.Semantic.DeepEqual(oldLabels, newLabels) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "modification of the following reserved kubevirt.io/ labels on a VMI object is prohibited",
			},
		})
	}

	return nil
}

func filterKubevirtLabels(labels map[string]string) map[string]string {
	m := make(map[string]string)
	if len(labels) == 0 {
		// Return the empty map to avoid edge cases
		return m
	}
	for label, value := range labels {
		if _, ok := restrictedVmiLabels[label]; ok {
			m[label] = value
		}
	}

	return m
}

func admitHotplug(
	oldVMI, newVMI *v1.VirtualMachineInstance,
	clusterConfig *virtconfig.ClusterConfig,
) *admissionv1.AdmissionResponse {

	if response := admitHotplugCPU(oldVMI.Spec.Domain.CPU, newVMI.Spec.Domain.CPU); response != nil {
		return response
	}

	if response := admitHotplugMemory(oldVMI.Spec.Domain.Memory, newVMI.Spec.Domain.Memory); response != nil {
		return response
	}

	return admitHotplugStorage(
		newVMI.Spec.Volumes,
		oldVMI.Spec.Volumes,
		newVMI.Spec.Domain.Devices.Disks,
		oldVMI.Spec.Domain.Devices.Disks,
		oldVMI.Status.VolumeStatus,
		newVMI,
		clusterConfig)

}

func admitHotplugCPU(oldCPUTopology, newCPUTopology *v1.CPU) *admissionv1.AdmissionResponse {

	if oldCPUTopology.MaxSockets != newCPUTopology.MaxSockets {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "CPU topology maxSockets changed",
			},
		})
	}

	return nil
}

func admitHotplugMemory(oldMemory, newMemory *v1.Memory) *admissionv1.AdmissionResponse {
	if oldMemory == nil ||
		oldMemory.MaxGuest == nil ||
		newMemory == nil ||
		newMemory.MaxGuest == nil {
		return nil
	}

	if !oldMemory.MaxGuest.Equal(*newMemory.MaxGuest) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Memory maxGuest changed",
			},
		})
	}

	return nil
}
