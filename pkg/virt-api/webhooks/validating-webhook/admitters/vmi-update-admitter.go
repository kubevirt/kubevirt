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
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	storageadmitters "kubevirt.io/kubevirt/pkg/storage/admitters"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const nodeNameExtraInfo = "authentication.kubernetes.io/node-name"

type VMIUpdateAdmitter struct {
	clusterConfig           *virtconfig.ClusterConfig
	kubeVirtServiceAccounts map[string]struct{}
}

func NewVMIUpdateAdmitter(config *virtconfig.ClusterConfig, kubeVirtServiceAccounts map[string]struct{}) *VMIUpdateAdmitter {
	return &VMIUpdateAdmitter{
		clusterConfig:           config,
		kubeVirtServiceAccounts: kubeVirtServiceAccounts,
	}
}

func (admitter *VMIUpdateAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := webhookutils.GetVMIFromAdmissionReview(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if admitter.clusterConfig.NodeRestrictionEnabled() && hasRequestOriginatedFromVirtHandler(ar.Request.UserInfo.Username, admitter.kubeVirtServiceAccounts) {
		values, exist := ar.Request.UserInfo.Extra[nodeNameExtraInfo]
		if exist && len(values) > 0 {
			nodeName := values[0]
			sourceNode := oldVMI.Status.NodeName
			targetNode := ""
			if oldVMI.Status.MigrationState != nil {
				targetNode = oldVMI.Status.MigrationState.TargetNode
			}

			// Check that source or target is making this request
			if nodeName != sourceNode && (targetNode == "" || nodeName != targetNode) {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "Node restriction, virt-handler is only allowed to modify VMIs it owns",
					},
				})
			}

			// Check that handler is not setting target
			if targetNode == "" && newVMI.Status.MigrationState != nil && newVMI.Status.MigrationState.TargetNode != targetNode {
				return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "Node restriction, virt-handler is not allowed to set target node",
					},
				})
			}
		} else {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "Node restriction failed, virt-handler service account is missing node name",
				},
			})
		}
	}

	// Reject VMI update if VMI spec changed
	_, isKubeVirtServiceAccount := admitter.kubeVirtServiceAccounts[ar.Request.UserInfo.Username]
	if !equality.Semantic.DeepEqual(newVMI.Spec, oldVMI.Spec) {
		// Only allow the KubeVirt SA to modify the VMI spec, since that means it went through the sub resource.
		if isKubeVirtServiceAccount {
			hotplugResponse := admitHotplug(oldVMI, newVMI, admitter.clusterConfig)
			if hotplugResponse != nil {
				return hotplugResponse
			}
		} else {
			return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
				{
					Type:    metav1.CauseTypeFieldValueNotSupported,
					Message: "update of VMI object is restricted",
				},
			})
		}
	}

	if !isKubeVirtServiceAccount {
		if reviewResponse := admitVMILabelsUpdate(newVMI, oldVMI); reviewResponse != nil {
			return reviewResponse
		}
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnDeprecatedAPIs(&newVMI.Spec, admitter.clusterConfig),
	}
}

func admitVMILabelsUpdate(
	newVMI *v1.VirtualMachineInstance,
	oldVMI *v1.VirtualMachineInstance,
) *admissionv1.AdmissionResponse {
	oldLabels := filterKubevirtLabels(oldVMI.ObjectMeta.Labels)
	newLabels := filterKubevirtLabels(newVMI.ObjectMeta.Labels)

	if !equality.Semantic.DeepEqual(oldLabels, newLabels) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "modification of the following reserved kubevirt.io/ labels on a VMI object is prohibited",
			},
		})
	}

	return nil
}

func filterKubevirtLabels(labels map[string]string) map[string]string {
	m := make(map[string]string)
	if len(labels) == 0 {
		// Return the empty map to avoid edge cases
		return m
	}
	for label, value := range labels {
		if _, ok := restrictedVmiLabels[label]; ok {
			m[label] = value
		}
	}

	return m
}

func admitHotplug(
	oldVMI, newVMI *v1.VirtualMachineInstance,
	clusterConfig *virtconfig.ClusterConfig,
) *admissionv1.AdmissionResponse {

	if response := admitHotplugCPU(oldVMI.Spec.Domain.CPU, newVMI.Spec.Domain.CPU); response != nil {
		return response
	}

	if response := admitHotplugMemory(oldVMI.Spec.Domain.Memory, newVMI.Spec.Domain.Memory); response != nil {
		return response
	}

	return storageadmitters.AdmitHotplugStorage(
		newVMI.Spec.Volumes,
		oldVMI.Spec.Volumes,
		newVMI.Spec.Domain.Devices.Disks,
		oldVMI.Spec.Domain.Devices.Disks,
		oldVMI.Status.VolumeStatus,
		newVMI,
		clusterConfig)

}

func admitHotplugCPU(oldCPUTopology, newCPUTopology *v1.CPU) *admissionv1.AdmissionResponse {

	if oldCPUTopology.MaxSockets != newCPUTopology.MaxSockets {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "CPU topology maxSockets changed",
			},
		})
	}

	return nil
}

func admitHotplugMemory(oldMemory, newMemory *v1.Memory) *admissionv1.AdmissionResponse {
	if oldMemory == nil ||
		oldMemory.MaxGuest == nil ||
		newMemory == nil ||
		newMemory.MaxGuest == nil {
		return nil
	}

	if !oldMemory.MaxGuest.Equal(*newMemory.MaxGuest) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "Memory maxGuest changed",
			},
		})
	}

	return nil
}

func hasRequestOriginatedFromVirtHandler(requestUsername string, kubeVirtServiceAccounts map[string]struct{}) bool {
	if _, isKubeVirtServiceAccount := kubeVirtServiceAccounts[requestUsername]; isKubeVirtServiceAccount {
		return strings.HasSuffix(requestUsername, components.HandlerServiceAccountName)
	}

	return false
}
