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
	"strconv"
	"strings"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	poolv1 "kubevirt.io/api/pool/v1alpha1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

type VMPoolAdmitter struct {
	ClusterConfig           *virtconfig.ClusterConfig
	KubeVirtServiceAccounts map[string]struct{}
}

func (admitter *VMPoolAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {

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

	_, isKubeVirtServiceAccount := admitter.KubeVirtServiceAccounts[ar.Request.UserInfo.Username]
	causes := ValidateVMPoolSpec(ar, k8sfield.NewPath("spec"), &pool, admitter.ClusterConfig, isKubeVirtServiceAccount)

	if ar.Request.Operation == admissionv1.Create {
		clusterCfg := admitter.ClusterConfig.GetConfig()
		featureGatesConfigs := clusterCfg.FeatureGates
		var legacyFeatureGates []string
		if devCfg := clusterCfg.DeveloperConfiguration; devCfg != nil {
			legacyFeatureGates = devCfg.FeatureGates
		}
		causes = append(causes, featuregate.ValidateFeatureGates(featureGatesConfigs, legacyFeatureGates, &pool.Spec.VirtualMachineTemplate.Spec.Template.Spec)...)
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnDeprecatedAPIs(&pool.Spec.VirtualMachineTemplate.Spec.Template.Spec, admitter.ClusterConfig),
	}
}

func ValidateVMPoolSpec(ar *admissionv1.AdmissionReview, field *k8sfield.Path, pool *poolv1.VirtualMachinePool, config *virtconfig.ClusterConfig, isKubeVirtServiceAccount bool) []metav1.StatusCause {
	var causes []metav1.StatusCause

	spec := &pool.Spec

	if spec.VirtualMachineTemplate == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: "missing virtual machine template.",
			Field:   field.Child("template").String(),
		})
	}

	causes = append(causes, ValidateVirtualMachineSpec(field.Child("virtualMachineTemplate", "spec"), &spec.VirtualMachineTemplate.Spec, config, isKubeVirtServiceAccount)...)

	selector, err := metav1.LabelSelectorAsSelector(spec.Selector)
	if err != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   field.Child("selector").String(),
		})
	} else if !selector.Matches(labels.Set(spec.VirtualMachineTemplate.ObjectMeta.Labels)) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: "selector does not match labels.",
			Field:   field.Child("selector").String(),
		})
	}

	if spec.MaxUnavailable != nil {
		if spec.MaxUnavailable.Type == intstr.String {
			if !strings.HasSuffix(spec.MaxUnavailable.StrVal, "%") {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "maxUnavailable percentage must end with %",
					Field:   field.Child("maxUnavailable").String(),
				})
			} else {
				percentage := strings.TrimSuffix(spec.MaxUnavailable.StrVal, "%")
				if val, err := strconv.Atoi(percentage); err != nil {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("maxUnavailable percentage value %q is invalid: %v", percentage, err),
						Field:   field.Child("maxUnavailable").String(),
					})
				} else if val <= 0 || val > 100 {
					causes = append(causes, metav1.StatusCause{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: fmt.Sprintf("maxUnavailable percentage value %d must be between 1 and 100", val),
						Field:   field.Child("maxUnavailable").String(),
					})
				}
			}
		} else if spec.MaxUnavailable.Type == intstr.Int {
			if spec.MaxUnavailable.IntVal <= 0 {
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: "maxUnavailable must be greater than 0",
					Field:   field.Child("maxUnavailable").String(),
				})
			}
		}
	}

	if ar.Request.Operation == admissionv1.Update {
		oldPool := &poolv1.VirtualMachinePool{}
		if err := json.Unmarshal(ar.Request.OldObject.Raw, oldPool); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeUnexpectedServerResponse,
				Message: "Could not fetch old vmpool",
			})
		}

		if !equality.Semantic.DeepEqual(pool.Spec.Selector, oldPool.Spec.Selector) {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "selector is immutable after creation.",
				Field:   field.Child("selector").String(),
			})
		}
	}
	return causes
}
