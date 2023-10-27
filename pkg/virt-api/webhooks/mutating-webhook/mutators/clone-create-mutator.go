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
 * Copyright the KubeVirt Authors.
 *
 */

package mutators

import (
	"encoding/json"
	"fmt"

	"kubevirt.io/client-go/log"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/api/clone"
	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type CloneCreateMutator struct {
}

func (mutator *CloneCreateMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resource := ar.Request.Resource; resource.Group != clone.GroupName || resource.Resource != clone.ResourceVMClonePlural {
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

	var patchOps []patch.PatchOperation
	var value interface{}
	value = vmClone.Spec
	patchOps = append(patchOps, patch.PatchOperation{
		Op:    "replace",
		Path:  "/spec",
		Value: value,
	})

	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	log.Log.Object(vmClone).V(4).Info(fmt.Sprintf("Mutating clone %s. Patch: %s", vmClone.Name, string(patchBytes)))

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

func mutateClone(vmClone *clonev1alpha1.VirtualMachineClone) {
	if vmClone.Spec.Target == nil {
		vmClone.Spec.Target = generateDefaultTarget(&vmClone.Spec)
	} else if vmClone.Spec.Target.Name == "" {
		vmClone.Spec.Target.Name = generateTargetName(vmClone.Spec.Source.Name)
	}
}

func generateTargetName(sourceName string) string {
	const randomSuffixLength = 5
	return fmt.Sprintf("clone-%s-%s", sourceName, rand.String(randomSuffixLength))
}

func generateDefaultTarget(cloneSpec *clonev1alpha1.VirtualMachineCloneSpec) (target *k8sv1.TypedLocalObjectReference) {
	const defaultTargetKind = "VirtualMachine"

	source := cloneSpec.Source

	target = &k8sv1.TypedLocalObjectReference{
		APIGroup: source.APIGroup,
		Kind:     defaultTargetKind,
		Name:     generateTargetName(source.Name),
	}

	return target
}
