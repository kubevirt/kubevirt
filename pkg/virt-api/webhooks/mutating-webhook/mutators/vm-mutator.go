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
 * Copyright 2019 Red Hat, Inc.
 */

package mutators

import (
	"encoding/json"

	admissionv1 "k8s.io/api/admission/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/flavor"

	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMsMutator struct {
	ClusterConfig *virtconfig.ClusterConfig
	FlavorMethods flavor.Methods
}

// until the minimum supported version is kubernetes 1.15 (see https://github.com/kubernetes/kubernetes/commit/c2fcdc818be1441dd788cae22648c04b1650d3af#diff-e057ec5b2ec27b4ba1e1a3915f715262)
// the mtuating webhook must pass silently on errors instead of returning errors
func emptyValidResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}

func (mutator *VMsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		log.Log.V(1).Warningf("vm-mutator: received invalid request")
		return emptyValidResponse()
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		log.Log.V(1).Warningf("vm-mutator: received invalid object in request")
		return emptyValidResponse()
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		log.Log.V(1).Warningf("vm-mutator: unable to unmarshal object in request")
		return emptyValidResponse()
	}

	// Set VM defaults
	log.Log.Object(&vm).V(4).Info("Apply defaults")
	mutator.setDefaultMachineType(&vm)

	if ar.Request.Operation == admissionv1.Create && vm.Spec.Flavor != nil {
		causes := mutator.applyFlavorToVm(&vm, ar)
		if len(causes) > 0 {
			return webhookutils.ToAdmissionResponse(causes)
		}
	}

	var patch []utiltypes.PatchOperation
	var value interface{}
	value = vm.Spec
	patch = append(patch, utiltypes.PatchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	value = vm.ObjectMeta
	patch = append(patch, utiltypes.PatchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		log.Log.V(1).Warningf("vm-mutator: unable to marshal object in request")
		return emptyValidResponse()
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func (mutator *VMsMutator) applyFlavorToVm(vm *v1.VirtualMachine, ar *admissionv1.AdmissionReview) []metav1.StatusCause {
	flavorProfile, err := mutator.FlavorMethods.FindProfile(vm)
	if err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueNotFound,
			Message: "Could not find flavor profile:",
			Field:   "spec.flavor",
		}}
	}

	if flavorProfile == nil {
		return nil
	}

	conflicts := mutator.FlavorMethods.ApplyToVmi(k8sfield.NewPath("spec", "template", "spec"), flavorProfile, &vm.Spec.Template.Spec)
	causes := []metav1.StatusCause{}
	for _, conflict := range conflicts {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "VM field conflicts with selected Flavor profile",
			Field:   conflict.String(),
		})
	}

	return causes
}

func (mutator *VMsMutator) setDefaultMachineType(vm *v1.VirtualMachine) {
	if vm.Spec.Template == nil {
		// nothing to do, let's the validating webhook fail later
		return
	}
	machineType := mutator.ClusterConfig.GetMachineType()

	if machine := vm.Spec.Template.Spec.Domain.Machine; machine != nil {
		if machine.Type == "" {
			machine.Type = machineType
		}
	} else {
		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{Type: machineType}
	}
}
