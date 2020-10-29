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
	"reflect"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"

	v1 "kubevirt.io/client-go/api/v1"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type VMIUpdateAdmitter struct {
}

func (admitter *VMIUpdateAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Only allow the KubeVirt SA to modify the VMI spec, since that means it went through the sub resource.
	allowed := webhooks.GetAllowedServiceAccounts()
	if _, ok := allowed[ar.Request.UserInfo.Username]; ok {
		hotplugResponse := admitHotplug(newVMI.Spec.Volumes, newVMI.Spec.Domain.Devices.Disks, oldVMI.Status.VolumeStatus)
		if hotplugResponse != nil {
			return hotplugResponse
		}
		// blank out volumes and disks so we can compare the rest of the VMI to ensure it didn't change
		newVMI.Spec.Volumes = []v1.Volume{}
		oldVMI.Spec.Volumes = []v1.Volume{}
		newVMI.Spec.Domain.Devices.Disks = []v1.Disk{}
		oldVMI.Spec.Domain.Devices.Disks = []v1.Disk{}
	}
	// Reject VMI update if VMI spec changed
	if !reflect.DeepEqual(newVMI.Spec, oldVMI.Spec) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "update of VMI object is restricted",
			},
		})
	}

	if reviewResponse := admitVMILabelsUpdate(newVMI, oldVMI, ar); reviewResponse != nil {
		return reviewResponse
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

// admitHotplug compares the old and new volumes and disks, and ensures that they match and are valid.
func admitHotplug(newVolumes []v1.Volume, newDisks []v1.Disk, volumeStatuses []v1.VolumeStatus) *v1beta1.AdmissionResponse {
	if len(newVolumes) != len(newDisks) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "number of disks does not equal the number of volumes",
			},
		})
	}
	hotplugVolumeMap := getHotplugVolumes(newVolumes, volumeStatuses)
	permanentVolumeMap := getPermanentVolumes(newVolumes, hotplugVolumeMap)

	// Ensure we didn't remove non hot plugged disks and volumes.
	permanentCount := 0
	for _, volume := range newVolumes {
		if _, ok := permanentVolumeMap[volume.Name]; ok {
			permanentCount++
		}
	}
	if len(permanentVolumeMap) > permanentCount || len(permanentVolumeMap) == 0 {
		// Removed one of the permanent volumes, reject admission.
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "cannot remove permanent volume",
			},
		})
	}

	// Ensure all the disks and volumes have matching names.
	for _, disk := range newDisks {
		if _, ok := hotplugVolumeMap[disk.Name]; !ok {
			// Not a hotplug volume, check permanent volumes
			if _, ok := permanentVolumeMap[disk.Name]; !ok {
				// Not in persistent map either, this disk doesn't have a matching volume.
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("Disk %s doesn't have a matching volume", disk.Name),
					},
				})
			}
		} else {
			// Ensure the volume source is either PVC or DataVolume
			volume, _ := hotplugVolumeMap[disk.Name]
			if volume.DataVolume == nil && volume.PersistentVolumeClaim == nil {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("Disk %s has a volume that is not a PVC or DataVolume", disk.Name),
					},
				})
			}
		}
	}
	return nil
}

func getHotplugVolumes(volumes []v1.Volume, volumeStatuses []v1.VolumeStatus) map[string]v1.Volume {
	hotplugVolumes := make(map[string]v1.Volume, 0)
	for _, volume := range volumeStatuses {
		if volume.HotplugVolume != nil {
			hotplugVolumes[volume.Name] = v1.Volume{}
		}
	}
	for _, volume := range volumes {
		if _, ok := hotplugVolumes[volume.Name]; ok {
			hotplugVolumes[volume.Name] = volume
		}
	}
	// Make sure the map only contains valid volumes
	for k, v := range hotplugVolumes {
		if k != v.Name {
			delete(hotplugVolumes, k)
		}
	}
	return hotplugVolumes
}

func getPermanentVolumes(volumes []v1.Volume, hotplugVolumeMap map[string]v1.Volume) map[string]v1.Volume {
	permanentVolumes := make(map[string]v1.Volume, 0)
	for _, volume := range volumes {
		if _, ok := hotplugVolumeMap[volume.Name]; !ok {
			permanentVolumes[volume.Name] = volume
		}
	}
	return permanentVolumes
}

func admitVMILabelsUpdate(
	newVMI *v1.VirtualMachineInstance,
	oldVMI *v1.VirtualMachineInstance,
	ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	// Skip admission for internal components
	allowed := webhooks.GetAllowedServiceAccounts()
	if _, ok := allowed[ar.Request.UserInfo.Username]; ok {
		return nil
	}

	oldLabels := filterKubevirtLabels(oldVMI.ObjectMeta.Labels)
	newLabels := filterKubevirtLabels(newVMI.ObjectMeta.Labels)

	if !reflect.DeepEqual(oldLabels, newLabels) {
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
		if _, ok := restriectedVmiLabels[label]; ok {
			m[label] = value
		}
	}

	return m
}
