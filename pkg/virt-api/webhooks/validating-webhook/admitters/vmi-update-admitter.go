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
	"reflect"

	"kubevirt.io/kubevirt/pkg/util"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
)

type VMIUpdateAdmitter struct {
}

func (admitter *VMIUpdateAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := getAdmissionReviewVMI(ar)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	// Reject VMI update if VMI spec changed
	if !reflect.DeepEqual(newVMI.Spec, oldVMI.Spec) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "update of VMI object is restricted",
			},
		})
	}

	if reviewResponse := admitVMILabelsUpdate(newVMI, oldVMI, ar); reviewResponse != nil {
		return reviewResponse
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true
	return &reviewResponse
}

func admitVMILabelsUpdate(
	newVMI *v1.VirtualMachineInstance,
	oldVMI *v1.VirtualMachineInstance,
	ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	// Skip admission for internal components
	allowed := util.GetAllowedServiceAccounts()
	if _, ok := allowed[ar.Request.UserInfo.Username]; ok {
		return nil
	}

	oldLabels := util.FilterKubevirtLabels(oldVMI.ObjectMeta.Labels)
	newLabels := util.FilterKubevirtLabels(newVMI.ObjectMeta.Labels)

	if !reflect.DeepEqual(oldLabels, newLabels) {
		return webhookutils.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "modification of kubevirt.io/ labels on a VMI object is restricted",
			},
		})
	}

	return nil
}
