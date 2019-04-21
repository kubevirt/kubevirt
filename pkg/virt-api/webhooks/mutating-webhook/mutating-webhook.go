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

package mutating_webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

type mutateFunc func(*v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func serve(resp http.ResponseWriter, req *http.Request, mutate mutateFunc) {
	response := v1beta1.AdmissionReview{}
	review, err := webhooks.GetAdmissionReview(req)

	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	reviewResponse := mutate(review)
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

func mutateVMIs(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if ar.Request.Resource != webhooks.VirtualMachineInstanceGroupVersionResource {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstanceGroupVersionResource.Resource)
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vmi := v1.VirtualMachineInstance{}

	err := json.Unmarshal(raw, &vmi)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	informers := webhooks.GetInformers()
	namespace, err := util.GetNamespace()
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}
	config := virtconfig.NewClusterConfig(informers.ConfigMapInformer.GetStore(), namespace)

	// Apply presets
	err = applyPresets(&vmi, informers.VMIPresetInformer)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusUnprocessableEntity,
			},
		}
	}

	// Apply namespace limits
	applyNamespaceLimitRangeValues(&vmi, informers.NamespaceLimitsInformer)

	// Set VMI defaults
	log.Log.Object(&vmi).V(4).Info("Apply defaults")
	setDefaultCPUModel(&vmi, config.GetCPUModel())
	setDefaultMachineType(&vmi, config.GetMachineType())
	v1.SetObjectDefaults_VirtualMachineInstance(&vmi)
	// Default CPU request is done after the Resources section is initialized, if needed,
	// in v1.SetObjectDefaults_VirtualMachineInstance. TODO: set default memory here
	setDefaultCPURequest(&vmi, config.GetCPURequest())

	// Add foreground finalizer
	vmi.Finalizers = append(vmi.Finalizers, v1.VirtualMachineInstanceFinalizer)

	var patch []patchOperation
	var value interface{}
	value = vmi.Spec
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	value = vmi.ObjectMeta
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}

}

func mutateMigrationCreate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if ar.Request.Resource != webhooks.MigrationGroupVersionResource {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.MigrationGroupVersionResource.Resource)
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(v1.VirtualMachineInstanceMigrationGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	migration := v1.VirtualMachineInstanceMigration{}

	err := json.Unmarshal(raw, &migration)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	// Add a finalizer
	migration.Finalizers = append(migration.Finalizers, v1.VirtualMachineInstanceMigrationFinalizer)
	var patch []patchOperation
	var value interface{}

	value = migration.Spec
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	value = migration.ObjectMeta
	patch = append(patch, patchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func setDefaultCPUModel(vmi *v1.VirtualMachineInstance, defaultCPUModel string) {
	//if vmi doesn't have cpu topology or cpu model set
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		if defaultCPUModel != "" {
			// create cpu topology struct
			if vmi.Spec.Domain.CPU == nil {
				vmi.Spec.Domain.CPU = &v1.CPU{}
			}
			//set is as vmi cpu model
			vmi.Spec.Domain.CPU.Model = defaultCPUModel
		}
	}
}

func setDefaultMachineType(vmi *v1.VirtualMachineInstance, defaultMachineType string) {
	if vmi.Spec.Domain.Machine.Type == "" {
		vmi.Spec.Domain.Machine.Type = defaultMachineType
	}
}

func setDefaultCPURequest(vmi *v1.VirtualMachineInstance, defaultCPURequest resource.Quantity) {
	if _, exists := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; !exists {
		if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.DedicatedCPUPlacement {
			return
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = defaultCPURequest
	}
}

func ServeVMIs(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, mutateVMIs)
}

func ServeMigrationCreate(resp http.ResponseWriter, req *http.Request) {
	serve(resp, req, mutateMigrationCreate)
}
