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

package admitters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
)

type VMsDeleteAdmitter struct {
	VirtClient kubecli.KubevirtClient
}

func NewVMsDeletionAdmitter(client kubecli.KubevirtClient) *VMsDeleteAdmitter {
	return &VMsDeleteAdmitter{
		VirtClient: client,
	}
}

func (admitter *VMsDeleteAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		err := fmt.Errorf("expected resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if ar.Request.DryRun != nil && *ar.Request.DryRun {
		log.Log.V(3).Infof("Skipping patch for VMI '%s/%s' due to dry-run", ar.Request.Namespace, ar.Request.Name)
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	if ar.Request.Options.Raw == nil {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	deleteOpts := &metav1.DeleteOptions{}
	if err := json.Unmarshal(ar.Request.Options.Raw, deleteOpts); err != nil {
		log.Log.Errorf("Failed to unmarshal DeleteOptions for VM '%s/%s': %v", ar.Request.Namespace, ar.Request.Name, err)
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	if deleteOpts.GracePeriodSeconds == nil {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	var vm *v1.VirtualMachine
	if err := json.Unmarshal(ar.Request.OldObject.Raw, &vm); err != nil {
		log.Log.Errorf("Failed to unmarshal old VM object for '%s/%s': %v", ar.Request.Namespace, ar.Request.Name, err)
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	currentGrace := *vm.Spec.Template.Spec.TerminationGracePeriodSeconds
	requestedGrace := *deleteOpts.GracePeriodSeconds

	if currentGrace == requestedGrace {
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	gracePatch, err := patch.New(patch.WithReplace("/spec/terminationGracePeriodSeconds", requestedGrace)).GeneratePayload()
	if err != nil {
		log.Log.Errorf("Failed to generate JSON patch for grace period: %v", err)
		return &admissionv1.AdmissionResponse{Allowed: true}
	}

	_, patchErr := admitter.VirtClient.VirtualMachineInstance(ar.Request.Namespace).Patch(
		ctx,
		ar.Request.Name,
		types.JSONPatchType,
		gracePatch,
		metav1.PatchOptions{},
	)

	if patchErr != nil && !errors.IsNotFound(patchErr) {
		return &admissionv1.AdmissionResponse{Allowed: false, Result: &metav1.Status{
			Message: fmt.Sprintf("Failed to update the VMI's terminationGracePeriodSeconds for VM '%s/%s'. Please try again.", ar.Request.Namespace, ar.Request.Name),
			Code:    http.StatusConflict,
		}}
	}

	return &admissionv1.AdmissionResponse{Allowed: true}
}
