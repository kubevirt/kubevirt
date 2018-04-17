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

func validateDisks(vm *v1.VirtualMachine) []error {
	errors := []error{}
	for idx, disk := range vm.Spec.Domain.Devices.Disks {
		var matchingVolume *v1.Volume

		// Verify disks and volume names line up.
		for _, volume := range vm.Spec.Volumes {
			if disk.VolumeName == volume.Name {
				matchingVolume = &volume
				break
			}
		}

		if matchingVolume == nil {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d].volumeName '%s' not found.", idx, disk.VolumeName))
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

		// Verify Lun disks are only mapped to network/block devices.
		if disk.LUN != nil && (matchingVolume == nil || matchingVolume.PersistentVolumeClaim == nil) {
			errors = append(errors, fmt.Errorf("spec.domain.devices.disks[%d].lun can only be mapped to a PersistentVolumeClaim volume.", idx))
		}
	}

	return errors
}

func validateVolumes(vm *v1.VirtualMachine) []error {
	errors := []error{}

	for idx, volume := range vm.Spec.Volumes {
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

func admitVMs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	errors := []error{}

	log.Log.Info("admitting vms")

	vmResource := metav1.GroupVersionResource{Group: v1.VirtualMachineGroupVersionKind.Group, Version: v1.VirtualMachineGroupVersionKind.Version, Resource: "virtualmachines"}
	if ar.Request.Resource != vmResource {
		err := fmt.Errorf("expect resource to be '%s'", vmResource)
		return toAdmissionResponse(err)
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return toAdmissionResponse(err)
	}

	errors = append(errors, validateDisks(&vm)...)
	errors = append(errors, validateVolumes(&vm)...)

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
