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

package admitters

import (
	"encoding/json"
	"fmt"

	"k8s.io/api/admission/v1beta1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type MigrationCreateAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (admitter *MigrationCreateAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	migration, _, err := getAdmissionReviewMigration(ar)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	if resp := webhooks.ValidateSchema(v1.VirtualMachineInstanceMigrationGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	if !admitter.ClusterConfig.LiveMigrationEnabled() {
		return webhooks.ToAdmissionResponseError(fmt.Errorf("LiveMigration feature gate is not enabled in kubevirt-config"))
	}

	causes := ValidateVirtualMachineInstanceMigrationSpec(k8sfield.NewPath("spec"), &migration.Spec)
	if len(causes) > 0 {
		return webhooks.ToAdmissionResponse(causes)
	}

	informers := webhooks.GetInformers()
	cacheKey := fmt.Sprintf("%s/%s", migration.Namespace, migration.Spec.VMIName)
	obj, exists, err := informers.VMIInformer.GetStore().GetByKey(cacheKey)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	// ensure VMI exists for the migration
	if !exists {
		return webhooks.ToAdmissionResponseError(fmt.Errorf("the VMI %s does not exist under the cache", migration.Spec.VMIName))
	}
	vmi := obj.(*v1.VirtualMachineInstance)

	// Don't allow introducing a migration job for a VMI that has already finalized
	if vmi.IsFinal() {
		return webhooks.ToAdmissionResponseError(fmt.Errorf("Cannot migrated VMI in finalized state."))
	}

	// Reject migration jobs for non-migratable VMIs
	for _, c := range vmi.Status.Conditions {
		if c.Type == v1.VirtualMachineInstanceIsMigratable &&
			c.Status == k8sv1.ConditionFalse {
			errMsg := fmt.Errorf("Cannot migrate VMI, Reason: %s, Message: %s",
				c.Reason, c.Message)
			return webhooks.ToAdmissionResponseError(errMsg)
		}
	}

	// Don't allow new migration jobs to be introduced when previous migration jobs
	// are already in flight.
	if vmi.Status.MigrationState != nil &&
		string(vmi.Status.MigrationState.MigrationUID) != "" &&
		!vmi.Status.MigrationState.Completed &&
		!vmi.Status.MigrationState.Failed {

		return webhooks.ToAdmissionResponseError(fmt.Errorf("in-flight migration detected. Active migration job (%s) is currently already in progress for VMI %s.", string(vmi.Status.MigrationState.MigrationUID), vmi.Name))
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func getAdmissionReviewMigration(ar *v1beta1.AdmissionReview) (new *v1.VirtualMachineInstanceMigration, old *v1.VirtualMachineInstanceMigration, err error) {

	if !webhooks.ValidateRequestResource(ar.Request.Resource, webhooks.MigrationGroupVersionResource.Group, webhooks.MigrationGroupVersionResource.Resource) {
		return nil, nil, fmt.Errorf("expect resource to be '%s'", webhooks.MigrationGroupVersionResource)
	}

	raw := ar.Request.Object.Raw
	newMigration := v1.VirtualMachineInstanceMigration{}

	err = json.Unmarshal(raw, &newMigration)
	if err != nil {
		return nil, nil, err
	}

	if ar.Request.Operation == v1beta1.Update {
		raw := ar.Request.OldObject.Raw
		oldMigration := v1.VirtualMachineInstanceMigration{}
		err = json.Unmarshal(raw, &oldMigration)
		if err != nil {
			return nil, nil, err
		}
		return &newMigration, &oldMigration, nil
	}

	return &newMigration, nil, nil
}

func ValidateVirtualMachineInstanceMigrationSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstanceMigrationSpec) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.VMIName == "" {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("vmiName is missing"),
			Field:   field.Child("vmiName").String(),
		})
	}

	return causes
}
