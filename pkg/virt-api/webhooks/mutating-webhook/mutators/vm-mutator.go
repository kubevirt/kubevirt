/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package mutators

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/defaults"
	instancetypeVMWebhooks "kubevirt.io/kubevirt/pkg/instancetype/webhooks/vm"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type instancetypeVMsMutator interface {
	Mutate(vm, oldVM *v1.VirtualMachine, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse
}

type VMsMutator struct {
	ClusterConfig       *virtconfig.ClusterConfig
	instancetypeMutator instancetypeVMsMutator
	virtClient          kubecli.KubevirtClient
}

func NewVMsMutator(clusterConfig *virtconfig.ClusterConfig, virtCli kubecli.KubevirtClient) *VMsMutator {
	return &VMsMutator{
		ClusterConfig:       clusterConfig,
		instancetypeMutator: instancetypeVMWebhooks.NewMutator(virtCli),
		virtClient:          virtCli,
	}
}

func (mutator *VMsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineGroupVersionResource.Group, webhooks.VirtualMachineGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	vm, oldVM, err := webhookutils.GetVMFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// If the VirtualMachine is being deleted return early and avoid racing any other in-flight resource deletions that might be happening
	if vm.DeletionTimestamp != nil {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	if response := mutator.instancetypeMutator.Mutate(vm, oldVM, ar); response != nil {
		return response
	}

	// Only assign a new-style firmware UUID when the VM is created.
	// On update, the mutator does not modify the UUID field to avoid
	// race conditions with the VM controller.
	if ar.Request.Operation == admissionv1.Create {
		setFirmwareDefaultsIfEmpty(vm)
	}

	// Set VM defaults
	log.Log.Object(vm).V(4).Info("Apply defaults")

	defaults.SetVirtualMachineDefaults(vm, mutator.ClusterConfig, mutator.virtClient)

	patchBytes, err := patch.New(
		patch.WithReplace("/spec", vm.Spec),
		patch.WithReplace("/metadata", vm.ObjectMeta),
	).GeneratePayload()

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

func setFirmwareDefaultsIfEmpty(vm *v1.VirtualMachine) {
	if vm.Spec.Template.Spec.Domain.Firmware == nil {
		vm.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{}
	}
	fw := vm.Spec.Template.Spec.Domain.Firmware

	if fw.UUID == "" {
		fw.UUID = types.UID(uuid.New().String())
	}

	if fw.Serial == "" {
		fw.Serial = uuid.New().String()
	}
}
