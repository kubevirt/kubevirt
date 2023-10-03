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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package webhooks

import (
	"fmt"

	"kubevirt.io/client-go/log"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	webhooks "kubevirt.io/kubevirt/pkg/util/webhooks"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

func NewKubeVirtCreateAdmitter(client kubecli.KubevirtClient) *kubeVirtCreateAdmitter {
	return &kubeVirtCreateAdmitter{
		client: client,
	}
}

type kubeVirtCreateAdmitter struct {
	client kubecli.KubevirtClient
}

// This validating webhook actually starts running AFTER the KubeVirt CR has been created
// as it gets installed by virt-operator in its sync-loop, this means that this will only
// check for creation of a new KubeVirt CR (rejecting it), no validation can be done here
// as that is done in the 'kubevirt-update-validator.kubevirt.io' webhook

func (k *kubeVirtCreateAdmitter) Admit(review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	log.Log.Info("Trying to create KV")
	if resp := webhooks.ValidateSchema(v1.KubeVirtGroupVersionKind, review.Request.Object.Raw); resp != nil {
		return resp
	}
	//TODO: Do we want semantic validation

	// Best effort
	list, err := k.client.KubeVirt(k8sv1.NamespaceAll).List(&metav1.ListOptions{})
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}
	if len(list.Items) == 0 {
		fmt.Println("Allowed to create KV")
		return webhookutils.NewPassingAdmissionResponse()
	}
	return webhooks.ToAdmissionResponseError(fmt.Errorf("Kubevirt is already created"))
}
