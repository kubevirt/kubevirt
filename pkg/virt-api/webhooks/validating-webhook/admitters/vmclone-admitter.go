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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package admitters

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clonebase "kubevirt.io/api/clone"
	clone "kubevirt.io/api/clone/v1beta1"
	"kubevirt.io/client-go/kubecli"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	clonecontroller "kubevirt.io/kubevirt/pkg/virt-controller/watch/clone"
)

const (
	virtualMachineKind         = "VirtualMachine"
	virtualMachineSnapshotKind = "VirtualMachineSnapshot"
)

// VirtualMachineCloneAdmitter validates VirtualMachineClones
type VirtualMachineCloneAdmitter struct {
	Config *virtconfig.ClusterConfig
	Client kubecli.KubevirtClient
}

// NewVMCloneAdmitter creates a VM Clone Admitter
func NewVMCloneAdmitter(config *virtconfig.ClusterConfig, client kubecli.KubevirtClient) *VirtualMachineCloneAdmitter {
	return &VirtualMachineCloneAdmitter{
		Config: config,
		Client: client,
	}
}

// Admit validates an AdmissionReview
func (admitter *VirtualMachineCloneAdmitter) Admit(ctx context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != clone.VirtualMachineCloneKind.Group {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected group: %+v. Expected group: %+v", ar.Request.Resource.Group, clone.VirtualMachineCloneKind.Group))
	}
	if ar.Request.Resource.Resource != clonebase.ResourceVMClonePlural {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource: %+v. Expected resource: %+v", ar.Request.Resource.Resource, clonebase.ResourceVMClonePlural))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.SnapshotEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("snapshot feature gate is not enabled"))
	}

	vmClone := &clone.VirtualMachineClone{}
	err := json.Unmarshal(ar.Request.Object.Raw, vmClone)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	var causes []metav1.StatusCause

	if newCauses := clonecontroller.ValidateFilters(vmClone.Spec.AnnotationFilters, "spec.annotations"); newCauses != nil {
		causes = append(causes, newCauses...)
	}
	if newCauses := clonecontroller.ValidateFilters(vmClone.Spec.LabelFilters, "spec.labels"); newCauses != nil {
		causes = append(causes, newCauses...)
	}
	if newCauses := clonecontroller.ValidateFilters(vmClone.Spec.Template.AnnotationFilters, "spec.template.annotations"); newCauses != nil {
		causes = append(causes, newCauses...)
	}
	if newCauses := clonecontroller.ValidateFilters(vmClone.Spec.Template.LabelFilters, "spec.template.labels"); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := clonecontroller.ValidateSourceAndTargetKind(vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := clonecontroller.ValidateSource(ctx, admitter.Client, vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := clonecontroller.ValidateTarget(vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if newCauses := clonecontroller.ValidateNewMacAddresses(vmClone); newCauses != nil {
		causes = append(causes, newCauses...)
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	reviewResponse := admissionv1.AdmissionResponse{
		Allowed: true,
	}
	return &reviewResponse
}
