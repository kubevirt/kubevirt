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

package validating_webhook

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	v1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

const (
	cloudInitMaxLen = 2048
	arrayLenMax     = 256
)

func getAdmissionReview(r *http.Request) (*v1beta1.AdmissionReview, error) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, fmt.Errorf("contentType=%s, expect application/json", contentType)
	}

	ar := &v1beta1.AdmissionReview{}
	err := json.Unmarshal(body, ar)
	return ar, err
}

func toAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	log.Log.Reason(err).Error("admitting vms")
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
			Code:    http.StatusUnprocessableEntity,
		},
	}
}

type admitFunc func(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func serve(resp http.ResponseWriter, req *http.Request, admit admitFunc) {
	response := v1beta1.AdmissionReview{}
	review, err := getAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	reviewResponse := admit(review)
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
	review.Request.Object = runtime.RawExtension{}
	review.Request.OldObject = runtime.RawExtension{}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		log.Log.Reason(err).Errorf("failed json encode webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := resp.Write(responseBytes); err != nil {
		log.Log.Reason(err).Errorf("failed to write webhook response")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}
	resp.WriteHeader(http.StatusOK)
}

func validateDisks(disks []v1.Disk) []error {
	errors := []error{}
	nameMap := make(map[string]int)

	if len(disks) > arrayLenMax {
		errors = append(errors, fmt.Errorf("spec.domain.devices.disks list exceeds the %d element limit in length", arrayLenMax))
		// We won't process anything over the limit
		return errors
	}

	for idx, disk := range disks {
		// verify name is unique
		otherIdx, ok := nameMap[disk.Name]
		if !ok {
			nameMap[disk.Name] = idx
		} else {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d] and spec.domain.devices.disks[%d] must not have the same Name.", idx, otherIdx))
		}
		// Verify only a single device type is set.
		deviceTargetSetCount := 0
		if disk.Disk != nil {
			deviceTargetSetCount++
		}
		if disk.LUN != nil {
			deviceTargetSetCount++
		}
		if disk.Floppy != nil {
			deviceTargetSetCount++
		}
		if disk.CDRom != nil {
			deviceTargetSetCount++
		}

		// NOTE: not setting a device target is okay. We default to Disk.
		// However, only a single device target is allowed to be set at a time.
		if deviceTargetSetCount > 1 {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d] can only have a single target type defined", idx))
		}

	}

	return errors
}

func validateVolumes(volumes []v1.Volume) []error {
	errors := []error{}
	nameMap := make(map[string]int)

	if len(volumes) > arrayLenMax {
		errors = append(errors, fmt.Errorf("spec.volumes list exceeds the %d element limit in length", arrayLenMax))
		// We won't process anything over the limit
		return errors
	}
	for idx, volume := range volumes {
		// verify name is unique
		otherIdx, ok := nameMap[volume.Name]
		if !ok {
			nameMap[volume.Name] = idx
		} else {
			errors = append(errors, fmt.Errorf("spec.volumes[%d] and spec.volumes[%d] must not have the same Name.", idx, otherIdx))
		}

		// Verify exactly one source is set
		volumeSourceSetCount := 0
		if volume.PersistentVolumeClaim != nil {
			volumeSourceSetCount++
		}
		if volume.CloudInitNoCloud != nil {
			volumeSourceSetCount++
		}
		if volume.RegistryDisk != nil {
			volumeSourceSetCount++
		}
		if volume.Ephemeral != nil {
			volumeSourceSetCount++
		}
		if volume.EmptyDisk != nil {
			volumeSourceSetCount++
		}

		if volumeSourceSetCount != 1 {
			errors = append(errors, fmt.Errorf("spec.volumes[%d] must have exactly one source type set", idx))
		}

		// Verify cloud init data is within size limits
		if volume.CloudInitNoCloud != nil {
			noCloud := volume.CloudInitNoCloud
			userDataLen := 0

			userDataSourceCount := 0
			if noCloud.UserDataSecretRef != nil && noCloud.UserDataSecretRef.Name != "" {
				userDataSourceCount++
			}
			if noCloud.UserDataBase64 != "" {
				userDataSourceCount++
				userData, err := base64.StdEncoding.DecodeString(noCloud.UserDataBase64)
				if err != nil {
					errors = append(errors, fmt.Errorf("spec.volumes[%d].cloudInitNoCloud.userDataBase64 is not a valid base64 value.", idx))
				}
				userDataLen = len(userData)
			}
			if noCloud.UserData != "" {
				userDataSourceCount++
				userDataLen = len(noCloud.UserData)
			}

			if userDataSourceCount != 1 {
				errors = append(errors, fmt.Errorf("spec.volumes[%d].cloudInitNoCloud must have one exactly one userdata source set.", idx))
			}

			if userDataLen > cloudInitMaxLen {
				errors = append(errors, fmt.Errorf("spec.volumes[%d].cloudInitNoCloud userdata exceeds %d byte limit", idx, cloudInitMaxLen))
			}
		}
	}
	return errors
}

func validateDevices(devices *v1.Devices) []error {
	errors := []error{}
	errors = append(errors, validateDisks(devices.Disks)...)
	return errors
}

func validateDomainSpec(spec *v1.DomainSpec) []error {
	errors := []error{}
	errors = append(errors, validateDevices(&spec.Devices)...)
	return errors
}

func validateVirtualMachineSpec(spec *v1.VirtualMachineSpec) []error {
	errors := []error{}
	volumeToDiskIndexMap := make(map[string]int)
	volumeNameMap := make(map[string]*v1.Volume)

	if len(spec.Domain.Devices.Disks) > arrayLenMax {
		errors = append(errors, fmt.Errorf("spec.domain.devices.disks list exceeds the %d element limit in length", arrayLenMax))
		// We won't process anything over the limit
		return errors
	} else if len(spec.Volumes) > arrayLenMax {
		errors = append(errors, fmt.Errorf("spec.volumes list exceeds the %d element limit in length", arrayLenMax))
		// We won't process anything over the limit
		return errors
	}

	for _, volume := range spec.Volumes {
		volumeNameMap[volume.Name] = &volume
	}

	// Validate disks and VolumeNames match up correctly
	for idx, disk := range spec.Domain.Devices.Disks {
		var matchingVolume *v1.Volume

		matchingVolume, volumeExists := volumeNameMap[disk.VolumeName]

		if !volumeExists {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d].volumeName '%s' not found.", idx, disk.VolumeName))
		}

		// verify no other disk maps to this volume
		otherIdx, ok := volumeToDiskIndexMap[disk.VolumeName]
		if !ok {
			volumeToDiskIndexMap[disk.VolumeName] = idx
		} else {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d] and spec.domain.devices.disks[%d] reference the same volumeName.", idx, otherIdx))
		}

		// Verify Lun disks are only mapped to network/block devices.
		if disk.LUN != nil && volumeExists && matchingVolume.PersistentVolumeClaim == nil {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d].lun can only be mapped to a PersistentVolumeClaim volume.", idx))
		}
	}

	errors = append(errors, validateDomainSpec(&spec.Domain)...)
	errors = append(errors, validateVolumes(spec.Volumes)...)
	return errors
}

func validateOfflineVirtualMachineSpec(spec *v1.OfflineVirtualMachineSpec) []error {
	errors := []error{}

	if spec.Template == nil {
		return append(errors, fmt.Errorf("missing virtual machine template."))
	}

	errors = append(errors, validateVirtualMachineSpec(&spec.Template.Spec)...)
	return errors
}

func validateVMPresetSpec(spec *v1.VirtualMachinePresetSpec) []error {
	errors := []error{}

	if spec.Domain == nil {
		return append(errors, fmt.Errorf("missing domain."))
	}

	errors = append(errors, validateDomainSpec(spec.Domain)...)
	return errors
}
func validateVMRSSpec(spec *v1.VMReplicaSetSpec) []error {
	errors := []error{}

	if spec.Template == nil {
		return append(errors, fmt.Errorf("missing virtual machine template."))
	}

	errors = append(errors, validateVirtualMachineSpec(&spec.Template.Spec)...)
	return errors
}

func admitVMs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	errors := []error{}

	vmResource := metav1.GroupVersionResource{
		Group:    v1.VirtualMachineGroupVersionKind.Group,
		Version:  v1.VirtualMachineGroupVersionKind.Version,
		Resource: "virtualmachines",
	}
	if ar.Request.Resource != vmResource {
		err := fmt.Errorf("expect resource to be '%s'", vmResource.Resource)
		return toAdmissionResponse(err)
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return toAdmissionResponse(err)
	}

	errors = append(errors, validateVirtualMachineSpec(&vm.Spec)...)
	if len(errors) > 0 {
		err := utilerrors.NewAggregate(errors)
		return toAdmissionResponse(err)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMs)
}

func admitOVMs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	errors := []error{}

	resource := metav1.GroupVersionResource{
		Group:    v1.OfflineVirtualMachineGroupVersionKind.Group,
		Version:  v1.OfflineVirtualMachineGroupVersionKind.Version,
		Resource: "offlinevirtualmachines",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponse(err)
	}

	raw := ar.Request.Object.Raw
	ovm := v1.OfflineVirtualMachine{}

	err := json.Unmarshal(raw, &ovm)
	if err != nil {
		return toAdmissionResponse(err)
	}

	errors = append(errors, validateOfflineVirtualMachineSpec(&ovm.Spec)...)
	if len(errors) > 0 {
		err := utilerrors.NewAggregate(errors)
		return toAdmissionResponse(err)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeOVMs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitOVMs)
}

func admitVMRS(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	errors := []error{}

	resource := metav1.GroupVersionResource{
		Group:    v1.VMReplicaSetGroupVersionKind.Group,
		Version:  v1.VMReplicaSetGroupVersionKind.Version,
		Resource: "virtualmachinereplicasets",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponse(err)
	}

	raw := ar.Request.Object.Raw
	vmrs := v1.VirtualMachineReplicaSet{}

	err := json.Unmarshal(raw, &vmrs)
	if err != nil {
		return toAdmissionResponse(err)
	}

	errors = append(errors, validateVMRSSpec(&vmrs.Spec)...)
	if len(errors) > 0 {
		err := utilerrors.NewAggregate(errors)
		return toAdmissionResponse(err)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMRS(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMRS)
}
func admitVMPreset(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	errors := []error{}

	resource := metav1.GroupVersionResource{
		Group:    v1.VMReplicaSetGroupVersionKind.Group,
		Version:  v1.VMReplicaSetGroupVersionKind.Version,
		Resource: "virtualmachinepresets",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponse(err)
	}

	raw := ar.Request.Object.Raw
	vmpreset := v1.VirtualMachinePreset{}

	err := json.Unmarshal(raw, &vmpreset)
	if err != nil {
		return toAdmissionResponse(err)
	}

	errors = append(errors, validateVMPresetSpec(&vmpreset.Spec)...)
	if len(errors) > 0 {
		err := utilerrors.NewAggregate(errors)
		return toAdmissionResponse(err)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMPreset(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMPreset)
}
