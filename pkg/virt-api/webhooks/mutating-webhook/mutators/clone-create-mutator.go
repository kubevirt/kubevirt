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
 * Copyright The KubeVirt Authors.
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

	clonebase "kubevirt.io/api/clone"
	clone "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type CloneCreateMutator struct {
	targetSuffix string
}

func NewCloneCreateMutator() *CloneCreateMutator {
	const randomSuffixLength = 5
	return NewCloneCreateMutatorWithTargetSuffix(rand.String(randomSuffixLength))
}

func NewCloneCreateMutatorWithTargetSuffix(targetSuffix string) *CloneCreateMutator {
	return &CloneCreateMutator{
		targetSuffix: targetSuffix,
	}
}

func (mutator *CloneCreateMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resource := ar.Request.Resource; resource.Group != clonebase.GroupName || resource.Resource != clonebase.ResourceVMClonePlural {
		err := fmt.Errorf("expect resource to be '%s'", clonebase.ResourceVMClonePlural)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(clone.VirtualMachineCloneKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	vmCloneOrig := &clone.VirtualMachineClone{}

	if err := json.Unmarshal(ar.Request.Object.Raw, &vmCloneOrig); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	vmClone := vmCloneOrig.DeepCopy()

	mutateClone(vmClone, mutator.targetSuffix)

	if !hasTargetChanged(vmCloneOrig.Spec.Target, vmClone.Spec.Target) {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := patch.New(patch.WithReplace("/spec", vmClone.Spec)).GeneratePayload()

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

func mutateClone(vmClone *clone.VirtualMachineClone, targetSuffix string) {
	if vmClone.Spec.Target == nil {
		vmClone.Spec.Target = generateDefaultTarget(&vmClone.Spec, targetSuffix)
	} else if vmClone.Spec.Target.Name == "" {
		vmClone.Spec.Target.Name = generateTargetName(vmClone.Spec.Source.Name, targetSuffix)
	}
}

func generateTargetName(sourceName string, targetSuffix string) string {
	return fmt.Sprintf("clone-%s-%s", sourceName, targetSuffix)
}

func generateDefaultTarget(cloneSpec *clone.VirtualMachineCloneSpec, targetSuffix string) (target *k8sv1.TypedLocalObjectReference) {
	const (
		virtualMachineAPIGroup = "kubevirt.io"
		virtualMachineKind     = "VirtualMachine"
	)

	source := cloneSpec.Source

	target = &k8sv1.TypedLocalObjectReference{
		APIGroup: pointer.P(virtualMachineAPIGroup),
		Kind:     virtualMachineKind,
		Name:     generateTargetName(source.Name, targetSuffix),
	}

	return target
}

func hasTargetChanged(original, mutated *k8sv1.TypedLocalObjectReference) bool {
	if original == nil {
		return true
	}

	if original.Name != mutated.Name ||
		original.Kind != mutated.Kind {
		return true
	}

	if original.APIGroup != nil &&
		*original.APIGroup != *mutated.APIGroup {
		return true
	}

	return false
}
