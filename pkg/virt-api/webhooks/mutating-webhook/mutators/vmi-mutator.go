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

package mutators

import (
	"encoding/json"
	"fmt"
	"net/http"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMIsMutator struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (mutator *VMIsMutator) Mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
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
	mutator.setDefaultCPUModel(&vmi)
	mutator.setDefaultMachineType(&vmi)
	mutator.setDefaultResourceRequests(&vmi)
	v1.SetObjectDefaults_VirtualMachineInstance(&vmi)

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

func (mutator *VMIsMutator) setDefaultCPUModel(vmi *v1.VirtualMachineInstance) {
	//if vmi doesn't have cpu topology or cpu model set
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		if defaultCPUModel := mutator.ClusterConfig.GetCPUModel(); defaultCPUModel != "" {
			// create cpu topology struct
			if vmi.Spec.Domain.CPU == nil {
				vmi.Spec.Domain.CPU = &v1.CPU{}
			}
			//set is as vmi cpu model
			vmi.Spec.Domain.CPU.Model = defaultCPUModel
		}
	}
}

func (mutator *VMIsMutator) setDefaultMachineType(vmi *v1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Machine.Type == "" {
		vmi.Spec.Domain.Machine.Type = mutator.ClusterConfig.GetMachineType()
	}
}

func (mutator *VMIsMutator) setDefaultResourceRequests(vmi *v1.VirtualMachineInstance) {
	if _, exists := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]; !exists {
		if vmi.Spec.Domain.Resources.Requests == nil {
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = mutator.ClusterConfig.GetMemoryRequest()
	}

	if _, exists := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU]; !exists {
		if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.DedicatedCPUPlacement {
			return
		}
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = mutator.ClusterConfig.GetCPURequest()
	}
}
