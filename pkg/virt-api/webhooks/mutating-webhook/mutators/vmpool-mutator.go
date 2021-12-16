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
 * Copyright 2019 Red Hat, Inc.
 */

package mutators

import (
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/log"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type VMPoolsMutator struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (mutator *VMPoolsMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {

	if ar.Request == nil {
		err := fmt.Errorf("Empty request for virtual machine pool validation")
		return webhookutils.ToAdmissionResponseError(err)
	} else if ar.Request.Resource.Resource != webhooks.VirtualMachinePoolGroupVersionResource.Resource {
		err := fmt.Errorf("expect resource [%s], but got [%s]", ar.Request.Resource.Resource, webhooks.VirtualMachinePoolGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	} else if ar.Request.Resource.Group != webhooks.VirtualMachinePoolGroupVersionResource.Group {
		err := fmt.Errorf("expect resource group [%s], but got [%s]", ar.Request.Resource.Group, webhooks.VirtualMachinePoolGroupVersionResource.Group)
		return webhookutils.ToAdmissionResponseError(err)
	} else if ar.Request.Resource.Version != webhooks.VirtualMachinePoolGroupVersionResource.Version {
		err := fmt.Errorf("expect resource version [%s], but got [%s]", ar.Request.Resource.Version, webhooks.VirtualMachinePoolGroupVersionResource.Version)
		return webhookutils.ToAdmissionResponseError(err)
	}

	gvk := schema.GroupVersionKind{
		Group:   webhooks.VirtualMachinePoolGroupVersionResource.Group,
		Version: webhooks.VirtualMachinePoolGroupVersionResource.Version,
		Kind:    poolv1.VirtualMachinePoolKind,
	}

	if resp := webhookutils.ValidateSchema(gvk, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	pool := poolv1.VirtualMachinePool{}

	err := json.Unmarshal(raw, &pool)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// TODO Set VMPool VM and VMI Hash Annotations
	log.Log.Object(&pool).V(4).Info("Apply vm/vmi hashes")

	var patch []utiltypes.PatchOperation
	var value interface{}
	value = pool.ObjectMeta
	patch = append(patch, utiltypes.PatchOperation{
		Op:    "replace",
		Path:  "/metadata",
		Value: value,
	})

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		log.Log.Reason(err).Errorf("vmpool-mutator: unable to marshal object in request")
		return webhookutils.ToAdmissionResponseError(err)
	}

	jsonPatchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}
