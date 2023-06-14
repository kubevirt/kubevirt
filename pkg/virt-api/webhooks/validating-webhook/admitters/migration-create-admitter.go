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
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type MigrationCreateAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
	VirtClient    kubecli.KubevirtClient
}

func isMigratable(vmi *v1.VirtualMachineInstance) error {
	for _, c := range vmi.Status.Conditions {
		if c.Type == v1.VirtualMachineInstanceIsMigratable &&
			c.Status == k8sv1.ConditionFalse {
			return fmt.Errorf("Cannot migrate VMI, Reason: %s, Message: %s", c.Reason, c.Message)
		}
	}
	return nil
}

func EnsureNoMigrationConflict(virtClient kubecli.KubevirtClient, vmiName string, namespace string) error {
	labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationSelectorLabel, vmiName))
	if err != nil {
		return err
	}
	list, err := virtClient.VirtualMachineInstanceMigration(namespace).List(&metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return err
	}
	if len(list.Items) > 0 {
		for _, mig := range list.Items {
			if mig.Status.Phase == v1.MigrationSucceeded || mig.Status.Phase == v1.MigrationFailed {
				continue
			}
			return fmt.Errorf("in-flight migration detected. Active migration job (%s) is currently already in progress for VMI %s.", string(mig.UID), mig.Spec.VMIName)
		}
	}

	return nil
}

func (admitter *MigrationCreateAdmitter) Admit(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	migration, _, err := getAdmissionReviewMigration(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceMigrationGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	causes := ValidateVirtualMachineInstanceMigrationSpec(k8sfield.NewPath("spec"), &migration.Spec)
	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	vmi, err := admitter.VirtClient.VirtualMachineInstance(migration.Namespace).Get(context.Background(), migration.Spec.VMIName, &metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// ensure VMI exists for the migration
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("the VMI \"%s/%s\" does not exist", migration.Namespace, migration.Spec.VMIName))
	} else if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Don't allow introducing a migration job for a VMI that has already finalized
	if vmi.IsFinal() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("Cannot migrate VMI in finalized state."))
	}

	// Reject migration jobs for non-migratable VMIs
	err = isMigratable(vmi)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Don't allow new migration jobs to be introduced when previous migration jobs
	// are already in flight.
	err = EnsureNoMigrationConflict(admitter.VirtClient, migration.Spec.VMIName, migration.Namespace)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func getAdmissionReviewMigration(ar *admissionv1.AdmissionReview) (new *v1.VirtualMachineInstanceMigration, old *v1.VirtualMachineInstanceMigration, err error) {

	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.MigrationGroupVersionResource.Group, webhooks.MigrationGroupVersionResource.Resource) {
		return nil, nil, fmt.Errorf("expect resource to be '%s'", webhooks.MigrationGroupVersionResource)
	}

	raw := ar.Request.Object.Raw
	newMigration := v1.VirtualMachineInstanceMigration{}

	err = json.Unmarshal(raw, &newMigration)
	if err != nil {
		return nil, nil, err
	}

	if ar.Request.Operation == admissionv1.Update {
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
