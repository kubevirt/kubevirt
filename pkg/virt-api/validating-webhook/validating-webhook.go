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

func toAdmissionResponseError(err error) *v1beta1.AdmissionResponse {
	log.Log.Reason(err).Error("admitting vms with generic error")

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		},
	}
}
func toAdmissionResponse(causes []metav1.StatusCause) *v1beta1.AdmissionResponse {
	log.Log.Infof("rejected vm admission")

	globalMessage := ""
	for _, cause := range causes {
		globalMessage = fmt.Sprintf("%s %s", globalMessage, cause.Message)
	}

	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: globalMessage,
			Code:    http.StatusUnprocessableEntity,
			Details: &metav1.StatusDetails{
				Causes: causes,
			},
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

func validateDisks(disks []v1.Disk) []metav1.StatusCause {
	var causes []metav1.StatusCause
	nameMap := make(map[string]int)

	if len(disks) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("spec.domain.devices.disks list exceeds the %d element limit in length", arrayLenMax),
			Field:   "spec.domain.devices.disks",
		})
		// We won't process anything over the limit
		return causes
	}

	for idx, disk := range disks {
		// verify name is unique
		otherIdx, ok := nameMap[disk.Name]
		if !ok {
			nameMap[disk.Name] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.domain.devices.disks[%d] and spec.domain.devices.disks[%d] must not have the same Name.", idx, otherIdx),
				Field:   fmt.Sprintf("spec.domain.devices.disks[%d]", idx),
			})
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
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.domain.devices.disks[%d] can only have a single target type defined", idx),
				Field:   fmt.Sprintf("spec.domain.devices.disks[%d]", idx),
			})
		}

	}

	return causes
}

func validateVolumes(volumes []v1.Volume) []metav1.StatusCause {
	var causes []metav1.StatusCause
	nameMap := make(map[string]int)

	if len(volumes) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("spec.volumes list exceeds the %d element limit in length", arrayLenMax),
			Field:   "spec.volumes",
		})
		// We won't process anything over the limit
		return causes
	}
	for idx, volume := range volumes {
		// verify name is unique
		otherIdx, ok := nameMap[volume.Name]
		if !ok {
			nameMap[volume.Name] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.volumes[%d] and spec.volumes[%d] must not have the same Name.", idx, otherIdx),
				Field:   fmt.Sprintf("spec.volumes[%d]", idx),
			})
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
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.volumes[%d] must have exactly one source type set", idx),
				Field:   fmt.Sprintf("spec.volumes[%d]", idx),
			})
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
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("spec.volumes[%d].cloudInitNoCloud.userDataBase64 is not a valid base64 value.", idx),
						Field:   fmt.Sprintf("spec.volumes[%d].cloudInitNoCloud.userDataBase64", idx),
					})
				}
				userDataLen = len(userData)
			}
			if noCloud.UserData != "" {
				userDataSourceCount++
				userDataLen = len(noCloud.UserData)
			}

			if userDataSourceCount != 1 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("spec.volumes[%d].cloudInitNoCloud must have one exactly one userdata source set.", idx),
					Field:   fmt.Sprintf("spec.volumes[%d].cloudInitNoCloud", idx),
				})
			}

			if userDataLen > cloudInitMaxLen {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("spec.volumes[%d].cloudInitNoCloud userdata exceeds %d byte limit", idx, cloudInitMaxLen),
					Field:   fmt.Sprintf("spec.volumes[%d].cloudInitNoCloud", idx),
				})
			}
		}
	}
	return causes
}

func validateDevices(devices *v1.Devices) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDisks(devices.Disks)...)
	return causes
}

func validateDomainSpec(spec *v1.DomainSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	causes = append(causes, validateDevices(&spec.Devices)...)
	return causes
}

func validateVirtualMachineSpec(spec *v1.VirtualMachineSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause
	volumeToDiskIndexMap := make(map[string]int)
	volumeNameMap := make(map[string]*v1.Volume)

	if len(spec.Domain.Devices.Disks) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("spec.domain.devices.disks list exceeds the %d element limit in length", arrayLenMax),
			Field:   "spec.domain.devices.disks",
		})
		// We won't process anything over the limit
		return causes
	} else if len(spec.Volumes) > arrayLenMax {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("spec.volumes list exceeds the %d element limit in length", arrayLenMax),
			Field:   "spec.volumes",
		})
		// We won't process anything over the limit
		return causes
	}

	for _, volume := range spec.Volumes {
		volumeNameMap[volume.Name] = &volume
	}

	// Validate disks and VolumeNames match up correctly
	for idx, disk := range spec.Domain.Devices.Disks {
		var matchingVolume *v1.Volume

		matchingVolume, volumeExists := volumeNameMap[disk.VolumeName]

		if !volumeExists {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.domain.devices.disks[%d].volumeName '%s' not found.", idx, disk.VolumeName),
				Field:   fmt.Sprintf("spec.domain.devices.disks[%d].volumeName", idx),
			})
		}

		// verify no other disk maps to this volume
		otherIdx, ok := volumeToDiskIndexMap[disk.VolumeName]
		if !ok {
			volumeToDiskIndexMap[disk.VolumeName] = idx
		} else {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.domain.devices.disks[%d] and spec.domain.devices.disks[%d] reference the same volumeName.", idx, otherIdx),
				Field:   fmt.Sprintf("spec.domain.devices.disks[%d].volumeName", idx),
			})
		}

		// Verify Lun disks are only mapped to network/block devices.
		if disk.LUN != nil && volumeExists && matchingVolume.PersistentVolumeClaim == nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("spec.domain.devices.disks[%d].lun can only be mapped to a PersistentVolumeClaim volume.", idx),
				Field:   fmt.Sprintf("spec.domain.devices.disks[%d].lun", idx),
			})
		}
	}

	causes = append(causes, validateDomainSpec(&spec.Domain)...)
	causes = append(causes, validateVolumes(spec.Volumes)...)
	return causes
}

func validateOfflineVirtualMachineSpec(spec *v1.OfflineVirtualMachineSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   "spec.template",
		})
	}

	causes = append(causes, validateVirtualMachineSpec(&spec.Template.Spec)...)
	return causes
}

func validateVMPresetSpec(spec *v1.VirtualMachinePresetSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Domain == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing domain."),
			Field:   "spec.domain",
		})
	}

	causes = append(causes, validateDomainSpec(spec.Domain)...)
	return causes
}
func validateVMRSSpec(spec *v1.VMReplicaSetSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   "spec.template",
		})
	}

	causes = append(causes, validateVirtualMachineSpec(&spec.Template.Spec)...)
	return causes
}

func admitVMs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	vmResource := metav1.GroupVersionResource{
		Group:    v1.VirtualMachineGroupVersionKind.Group,
		Version:  v1.VirtualMachineGroupVersionKind.Version,
		Resource: "virtualmachines",
	}
	if ar.Request.Resource != vmResource {
		err := fmt.Errorf("expect resource to be '%s'", vmResource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := validateVirtualMachineSpec(&vm.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMs)
}

func admitOVMs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    v1.OfflineVirtualMachineGroupVersionKind.Group,
		Version:  v1.OfflineVirtualMachineGroupVersionKind.Version,
		Resource: "offlinevirtualmachines",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	ovm := v1.OfflineVirtualMachine{}

	err := json.Unmarshal(raw, &ovm)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := validateOfflineVirtualMachineSpec(&ovm.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeOVMs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitOVMs)
}

func admitVMRS(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    v1.VMReplicaSetGroupVersionKind.Group,
		Version:  v1.VMReplicaSetGroupVersionKind.Version,
		Resource: "virtualmachinereplicasets",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vmrs := v1.VirtualMachineReplicaSet{}

	err := json.Unmarshal(raw, &vmrs)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := validateVMRSSpec(&vmrs.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMRS(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMRS)
}
func admitVMPreset(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	resource := metav1.GroupVersionResource{
		Group:    v1.VMReplicaSetGroupVersionKind.Group,
		Version:  v1.VMReplicaSetGroupVersionKind.Version,
		Resource: "virtualmachinepresets",
	}
	if ar.Request.Resource != resource {
		err := fmt.Errorf("expect resource to be '%s'", resource.Resource)
		return toAdmissionResponseError(err)
	}

	raw := ar.Request.Object.Raw
	vmpreset := v1.VirtualMachinePreset{}

	err := json.Unmarshal(raw, &vmpreset)
	if err != nil {
		return toAdmissionResponseError(err)
	}

	causes := validateVMPresetSpec(&vmpreset.Spec)
	if len(causes) > 0 {
		return toAdmissionResponse(causes)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func ServeVMPreset(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, admitVMPreset)
}
