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

	"k8s.io/api/admission/v1beta1"

	v1 "kubevirt.io/client-go/api/v1"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

type MigrationCreateMutator struct {
}

func (mutator *MigrationCreateMutator) Mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.MigrationGroupVersionResource.Group, webhooks.MigrationGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.MigrationGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceMigrationGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	migration := v1.VirtualMachineInstanceMigration{}

	err := json.Unmarshal(raw, &migration)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
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
		return webhookutils.ToAdmissionResponseError(err)
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}
