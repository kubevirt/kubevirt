/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package mutators

import (
	"encoding/json"
	"fmt"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	admissionv1 "k8s.io/api/admission/v1"

	v1 "kubevirt.io/api/core/v1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

type MigrationCreateMutator struct {
}

func (mutator *MigrationCreateMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
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

	addMigrationSelectorLabel(&migration)
	addMigrationFinalizer(&migration)

	patchBytes, err := patch.New(patch.WithReplace("/metadata", migration.ObjectMeta)).GeneratePayload()
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

func addMigrationSelectorLabel(migration *v1.VirtualMachineInstanceMigration) {
	if migration.Labels == nil {
		migration.Labels = make(map[string]string)
	}

	migration.Labels[v1.MigrationSelectorLabel] = migration.Spec.VMIName
}

func addMigrationFinalizer(migration *v1.VirtualMachineInstanceMigration) {
	migration.Finalizers = append(migration.Finalizers, v1.VirtualMachineInstanceMigrationFinalizer)
}
