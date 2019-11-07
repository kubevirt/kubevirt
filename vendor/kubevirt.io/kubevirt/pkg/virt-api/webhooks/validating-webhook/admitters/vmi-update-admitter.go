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
	"fmt"
	"reflect"
	"strings"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-operator/creation/rbac"
)

type VMIUpdateAdmitter struct {
}

func (admitter *VMIUpdateAdmitter) Admit(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {

	if resp := webhooks.ValidateSchema(v1.VirtualMachineInstanceGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}
	// Get new VMI from admission response
	newVMI, oldVMI, err := getAdmissionReviewVMI(ar)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	// Reject VMI update if VMI spec changed
	if !reflect.DeepEqual(newVMI.Spec, oldVMI.Spec) {
		return webhooks.ToAdmissionResponse([]metav1.StatusCause{
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
	allowed := getAllowedServiceAccounts()
	if _, ok := allowed[ar.Request.UserInfo.Username]; ok {
		return nil
	}

	oldLabels := filterKubevirtLabels(oldVMI)
	newLabels := filterKubevirtLabels(newVMI)

	if !reflect.DeepEqual(oldLabels, newLabels) {
		return webhooks.ToAdmissionResponse([]metav1.StatusCause{
			{
				Type:    metav1.CauseTypeFieldValueNotSupported,
				Message: "modification of kubevirt.io/ labels on a VMI object is restricted",
			},
		})
	}

	return nil
}

func filterKubevirtLabels(vmi *v1.VirtualMachineInstance) map[string]string {
	m := make(map[string]string)
	if vmi.Labels == nil {
		// Return the empty map to avoid edge cases
		return m
	}
	for label, value := range vmi.Labels {
		if strings.HasPrefix(label, "kubevirt.io") {
			m[label] = value
		}
	}
	return m
}

func getAllowedServiceAccounts() map[string]struct{} {
	ns, err := clientutil.GetNamespace()
	logger := log.DefaultLogger()

	if err != nil {
		logger.Info("Failed to get namespace. Fallback to default: 'kubevirt'")
		ns = "kubevirt"
	}

	// system:serviceaccount:{namespace}:{kubevirt-component}
	prefix := fmt.Sprintf("%s:%s:%s", "system", "serviceaccount", ns)
	return map[string]struct{}{
		fmt.Sprintf("%s:%s", prefix, rbac.ApiServiceAccountName):        {},
		fmt.Sprintf("%s:%s", prefix, rbac.HandlerServiceAccountName):    {},
		fmt.Sprintf("%s:%s", prefix, rbac.ControllerServiceAccountName): {},
	}
}
