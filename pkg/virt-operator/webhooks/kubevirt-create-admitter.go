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

package webhooks

import (
	"context"
	"fmt"


	"kubevirt.io/client-go/log"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

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
func (k *kubeVirtCreateAdmitter) Admit(ctx context.Context, review *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	log.Log.Info("Trying to create KV")
	if resp := webhooks.ValidateSchema(v1.KubeVirtGroupVersionKind, review.Request.Object.Raw); resp != nil {
		return resp
	}

	// Get new KubeVirt from admission response
	newKV, _, err := getAdmissionReviewKubeVirt(review)
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}

	var results []metav1.StatusCause

	results = append(results, validateCustomizeComponents(newKV.Spec.CustomizeComponents)...)
	results = append(results, validateCertificates(newKV.Spec.CertificateRotationStrategy.SelfSigned)...)
	results = append(results, validateGuestToRequestHeadroom(newKV.Spec.Configuration.AdditionalGuestMemoryOverheadRatio)...)

	if newKV.Spec.Configuration.TLSConfiguration != nil {
		results = append(results,
			validateTLSConfiguration(newKV.Spec.Configuration.TLSConfiguration)...)
	}

	if newKV.Spec.Infra != nil && newKV.Spec.Infra.NodePlacement != nil {
		results = append(results,
			validateInfraPlacement(ctx, newKV.Namespace, newKV.Spec.Infra.NodePlacement, k.client)...)
	}

	if newKV.Spec.Workloads != nil && newKV.Spec.Workloads.NodePlacement != nil {
		results = append(results,
			validateWorkloadPlacement(ctx, newKV.Namespace, newKV.Spec.Workloads.NodePlacement, k.client)...)
	}

	results = append(results,
		validateSeccompConfiguration(field.NewPath("spec").Child("configuration", "seccompConfiguration"), newKV.Spec.Configuration.SeccompConfiguration)...)

	if newKV.Spec.Infra != nil {
		results = append(results, validateInfraReplicas(newKV.Spec.Infra.Replicas)...)
	}

	// If any validation failed, return the errors
	if len(results) > 0 {
		return webhookutils.NewAdmissionResponse(results)
	}

	// Best effort
	list, err := k.client.KubeVirt(k8sv1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return webhooks.ToAdmissionResponseError(err)
	}
	if len(list.Items) == 0 {
		fmt.Println("Allowed to create KV")
		response := webhookutils.NewPassingAdmissionResponse()
		// warnings
		// feature gates
		featureGates := []string{}
		if newKV.Spec.Configuration.DeveloperConfiguration != nil {
			featureGates = newKV.Spec.Configuration.DeveloperConfiguration.FeatureGates
		}
		response.Warnings = append(response.Warnings, warnDeprecatedFeatureGates(featureGates)...)

		// mdevs
		const mdevWarningfmt = "%s is deprecated, use mediatedDeviceTypes"
		if mdev := newKV.Spec.Configuration.MediatedDevicesConfiguration; mdev != nil {
			f := field.NewPath("spec", "configuration", "mediatedDevicesConfiguration")
			if mdev.MediatedDevicesTypes != nil {
				f := f.Child("mediatedDevicesTypes")
				response.Warnings = append(response.Warnings, fmt.Sprintf(mdevWarningfmt, f.String()))
			}

			f = f.Child("nodeMediatedDeviceTypes")
			for i, mdevType := range mdev.NodeMediatedDeviceTypes {
				f := f.Index(i).Child("mediatedDevicesTypes")
				if mdevType.MediatedDevicesTypes != nil {
					response.Warnings = append(response.Warnings, fmt.Sprintf(mdevWarningfmt, f.String()))
				}
			}
		}

		response.Warnings = append(response.Warnings, warnDeprecatedArchitectures(newKV.Spec.Configuration.ArchitectureConfiguration)...)
		return response
	}
	return webhooks.ToAdmissionResponseError(fmt.Errorf("Kubevirt is already created"))
}
