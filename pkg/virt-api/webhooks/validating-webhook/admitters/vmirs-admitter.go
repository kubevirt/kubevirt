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

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

type VMIRSAdmitter struct {
	ClusterConfig *virtconfig.ClusterConfig
}

func (admitter *VMIRSAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if !webhookutils.ValidateRequestResource(ar.Request.Resource, webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource.Group, webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource.Resource) {
		err := fmt.Errorf("expect resource to be '%s'", webhooks.VirtualMachineInstanceReplicaSetGroupVersionResource.Resource)
		return webhookutils.ToAdmissionResponseError(err)
	}

	if resp := webhookutils.ValidateSchema(v1.VirtualMachineInstanceReplicaSetGroupVersionKind, ar.Request.Object.Raw); resp != nil {
		return resp
	}

	raw := ar.Request.Object.Raw
	vmirs := v1.VirtualMachineInstanceReplicaSet{}

	err := json.Unmarshal(raw, &vmirs)
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := ValidateVMIRSSpec(k8sfield.NewPath("spec"), &vmirs.Spec, admitter.ClusterConfig)

	if ar.Request.Operation == admissionv1.Create {
		clusterCfg := admitter.ClusterConfig.GetConfig()
		featureGatesConfigs := clusterCfg.FeatureGates
		var legacyFeatureGates []string
		if devCfg := clusterCfg.DeveloperConfiguration; devCfg != nil {
			legacyFeatureGates = devCfg.FeatureGates
		}
		causes = append(causes, featuregate.ValidateFeatureGates(featureGatesConfigs, legacyFeatureGates, &vmirs.Spec.Template.Spec)...)
	}

	if len(causes) > 0 {
		return webhookutils.ToAdmissionResponse(causes)
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnDeprecatedAPIs(&vmirs.Spec.Template.Spec, admitter.ClusterConfig),
	}
}

func ValidateVMIRSSpec(field *k8sfield.Path, spec *v1.VirtualMachineInstanceReplicaSetSpec, config *virtconfig.ClusterConfig) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if spec.Template == nil {
		return append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueRequired,
			Message: fmt.Sprintf("missing virtual machine template."),
			Field:   field.Child("template").String(),
		})
	}
	causes = append(causes, ValidateVirtualMachineInstanceSpec(field.Child("template", "spec"), &spec.Template.Spec, config)...)

	selector, err := metav1.LabelSelectorAsSelector(spec.Selector)
	if err != nil {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: err.Error(),
			Field:   field.Child("selector").String(),
		})
	} else if !selector.Matches(labels.Set(spec.Template.ObjectMeta.Labels)) {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("selector does not match labels."),
			Field:   field.Child("selector").String(),
		})
	}

	return causes
}
