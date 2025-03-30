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
	"fmt"
	"net/http"
	"strings"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/defaults"
	kvpointer "kubevirt.io/kubevirt/pkg/pointer"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMIsMutator struct {
	ClusterConfig           *virtconfig.ClusterConfig
	VMIPresetInformer       cache.SharedIndexInformer
	KubeVirtServiceAccounts map[string]struct{}
}

const presetDeprecationWarning = "kubevirt.io/v1 VirtualMachineInstancePresets is now deprecated and will be removed in v2."

func (mutator *VMIsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineInstanceGroupVersionResource.Group, webhooks.VirtualMachineInstanceGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstanceGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	patchSet := patch.New()

	// Patch the spec, metadata and status with defaults if we deal with a create operation
	if ar.Request.Operation == admissionv1.Create {
		// Apply presets
		err = applyPresets(newVMI, mutator.VMIPresetInformer)
		if err != nil {
			return &admissionv1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
					Code:    http.StatusUnprocessableEntity,
				},
			}
		}

		// Set VirtualMachineInstance defaults
		log.Log.Object(newVMI).V(4).Info("Apply defaults")
		if err = defaults.SetDefaultVirtualMachineInstance(mutator.ClusterConfig, newVMI); err != nil {
			return webhookutils.ToAdmissionResponseError(err)
		}

		if newVMI.Spec.Domain.CPU.IsolateEmulatorThread {
			_, emulatorThreadCompleteToEvenParityAnnotationExists := mutator.ClusterConfig.GetConfigFromKubeVirtCR().Annotations[v1.EmulatorThreadCompleteToEvenParity]
			if emulatorThreadCompleteToEvenParityAnnotationExists &&
				mutator.ClusterConfig.AlignCPUsEnabled() {
				log.Log.V(4).Infof("Copy %s annotation from Kubevirt CR", v1.EmulatorThreadCompleteToEvenParity)
				if newVMI.Annotations == nil {
					newVMI.Annotations = map[string]string{}
				}
				newVMI.Annotations[v1.EmulatorThreadCompleteToEvenParity] = ""
			}
		}

		// Add foreground finalizer
		newVMI.Finalizers = append(newVMI.Finalizers, v1.VirtualMachineInstanceFinalizer)

		// Set the phase to pending to avoid blank status
		newVMI.Status.Phase = v1.Pending

		now := metav1.NewTime(time.Now())
		newVMI.Status.PhaseTransitionTimestamps = append(newVMI.Status.PhaseTransitionTimestamps, v1.VirtualMachineInstancePhaseTransitionTimestamp{
			Phase:                    newVMI.Status.Phase,
			PhaseTransitionTimestamp: now,
		})

		if !mutator.ClusterConfig.RootEnabled() {
			markAsNonroot(newVMI)
		}

		patchSet.AddOption(
			patch.WithReplace("/spec", newVMI.Spec),
			patch.WithReplace("/metadata", newVMI.ObjectMeta),
			patch.WithReplace("/status", newVMI.Status),
		)

	} else if ar.Request.Operation == admissionv1.Update {
		// Ignore status updates if they are not coming from our service accounts
		// TODO: As soon as CRDs support field selectors we can remove this and just enable
		// the status subresource. Until then we need to update Status and Metadata labels in parallel for e.g. Migrations.
		if !equality.Semantic.DeepEqual(newVMI.Status, oldVMI.Status) {
			if _, isKubeVirtServiceAccount := mutator.KubeVirtServiceAccounts[ar.Request.UserInfo.Username]; !isKubeVirtServiceAccount {
				patchSet.AddOption(patch.WithReplace("/status", oldVMI.Status))
			}
		}
	}

	if patchSet.IsEmpty() {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	response := &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: kvpointer.P(admissionv1.PatchTypeJSONPatch),
	}
	// If newVMI has been annotated with presets include a deprecation warning in the response
	for annotation := range newVMI.Annotations {
		if strings.Contains(annotation, "virtualmachinepreset") {
			response.Warnings = []string{presetDeprecationWarning}
			break
		}
	}

	return response
}

func markAsNonroot(vmi *v1.VirtualMachineInstance) {
	vmi.Status.RuntimeUser = 107
}
