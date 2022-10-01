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
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMsMutator struct {
	ClusterConfig       *virtconfig.ClusterConfig
	InstancetypeMethods instancetype.Methods
}

func (mutator *VMsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vm := v1.VirtualMachine{}

	err := json.Unmarshal(raw, &vm)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Set VM defaults
	log.Log.Object(&vm).V(4).Info("Apply defaults")
	mutator.setDefaultInstancetypeKind(&vm)
	mutator.setDefaultPreferenceKind(&vm)
	mutator.setDefaultMachineType(&vm)

	patchBytes, err := utiltypes.GeneratePatchPayload(
		utiltypes.PatchOperation{
			Op:    utiltypes.PatchReplaceOp,
			Path:  "/spec",
			Value: vm.Spec,
		},
		utiltypes.PatchOperation{
			Op:    utiltypes.PatchReplaceOp,
			Path:  "/metadata",
			Value: vm.ObjectMeta,
		},
	)

	if err != nil {
		log.Log.Reason(err).Error("admission failed to marshall patch to JSON")
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
				Code:    http.StatusInternalServerError,
			},
		}
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func (mutator *VMsMutator) setDefaultMachineType(vm *v1.VirtualMachine) {
	// Nothing to do, let's the validating webhook fail later
	if vm.Spec.Template == nil {
		return
	}

	if machine := vm.Spec.Template.Spec.Domain.Machine; machine != nil && machine.Type != "" {
		return
	}

	if vm.Spec.Template.Spec.Domain.Machine == nil {
		vm.Spec.Template.Spec.Domain.Machine = &v1.Machine{}
	}

	vm.Spec.Template.Spec.Domain.Machine.Type = mutator.getPreferenceMachineType(vm)

	// Only use the cluster default if the user hasn't provided a machine type or referenced a preference with PreferredMachineType
	if vm.Spec.Template.Spec.Domain.Machine.Type == "" {
		vm.Spec.Template.Spec.Domain.Machine.Type = mutator.ClusterConfig.GetMachineType()
	}
}

func (mutator *VMsMutator) getPreferenceMachineType(vm *v1.VirtualMachine) string {
	preferenceSpec, err := mutator.InstancetypeMethods.FindPreferenceSpec(vm)
	if err != nil {
		// Log but ultimately swallow any preference lookup errors here and let the validating webhook handle them
		log.Log.Reason(err).Error("Ignoring error attempting to lookup PreferredMachineType.")
		return ""
	}
	if preferenceSpec != nil && preferenceSpec.Machine != nil {
		return preferenceSpec.Machine.PreferredMachineType
	}
	return ""
}

func (mutator *VMsMutator) setDefaultInstancetypeKind(vm *v1.VirtualMachine) {
	if vm.Spec.Instancetype == nil {
		return
	}

	if vm.Spec.Instancetype.Kind == "" {
		vm.Spec.Instancetype.Kind = apiinstancetype.ClusterSingularResourceName
	}
}

func (mutator *VMsMutator) setDefaultPreferenceKind(vm *v1.VirtualMachine) {
	if vm.Spec.Preference == nil {
		return
	}

	if vm.Spec.Preference.Kind == "" {
		vm.Spec.Preference.Kind = apiinstancetype.ClusterSingularPreferenceResourceName
	}
}
