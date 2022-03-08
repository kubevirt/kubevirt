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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package mutators

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"

	"kubevirt.io/api/clone"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type CloneCreateMutator struct {
}

func (mutator *CloneCreateMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, clone.GroupName, clone.ResourceVMClonePlural) {
		err := fmt.Errorf("expect resource to be '%s'", clone.ResourceVMClonePlural)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(clonev1alpha1.VirtualMachineCloneKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vmClone := &clonev1alpha1.VirtualMachineClone{}

	err := json.Unmarshal(raw, &vmClone)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	mutateClone(vmClone)

	var patch []utiltypes.PatchOperation
	var value interface{}
	value = vmClone.Spec
	patch = append(patch, utiltypes.PatchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func mutateClone(vmClone *clonev1alpha1.VirtualMachineClone) {
}
