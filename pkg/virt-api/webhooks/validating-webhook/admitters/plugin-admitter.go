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
	"path/filepath"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/plugin"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	nodecelutil "kubevirt.io/kubevirt/pkg/plugins/cel"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	celutil "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
)

type PluginAdmitter struct {
	Config *virtconfig.ClusterConfig
}

func NewPluginAdmitter(config *virtconfig.ClusterConfig) *PluginAdmitter {
	return &PluginAdmitter{Config: config}
}

func (admitter *PluginAdmitter) Admit(_ context.Context, ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Group != plugin.GroupName {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected group: %s, expected: %s", ar.Request.Resource.Group, plugin.GroupName))
	}
	if ar.Request.Resource.Resource != plugin.ResourcePluginPlural {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("unexpected resource: %s, expected: %s", ar.Request.Resource.Resource, plugin.ResourcePluginPlural))
	}

	if ar.Request.Operation == admissionv1.Create && !admitter.Config.PluginsEnabled() {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf("Plugins feature gate is not enabled"))
	}

	raw := ar.Request.Object.Raw
	p := &pluginv1alpha1.Plugin{}
	if err := json.Unmarshal(raw, p); err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	causes := validatePlugin(p)
	resp := validating_webhooks.NewPassingAdmissionResponse()
	if len(causes) > 0 {
		resp.Allowed = false
		resp.Result = &metav1.Status{
			Message: causes[0].Message,
			Reason:  metav1.StatusReasonInvalid,
			Details: &metav1.StatusDetails{
				Causes: causes,
			},
		}
	}
	return resp
}

func validatePlugin(p *pluginv1alpha1.Plugin) []metav1.StatusCause {
	var causes []metav1.StatusCause
	specPath := field.NewPath("spec")

	if hasCELExpressions(p) {
		eval := celutil.GetEvaluator()
		for i, dh := range p.Spec.DomainHooks {
			causes = append(causes, validateDomainHookCEL(eval, dh, specPath.Child("domainHooks").Index(i))...)
		}

		nodeEval, err := nodecelutil.NewEvaluator()
		if err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("failed to create node hook CEL evaluator: %s", err),
				Field:   specPath.Child("nodeHooks").String(),
			})
		} else {
			for i, nh := range p.Spec.NodeHooks {
				causes = append(causes, validateNodeHookCEL(nodeEval, nh, specPath.Child("nodeHooks").Index(i))...)
			}
		}
	}

	causes = append(causes, validateSidecarSocketPaths(p)...)

	return causes
}

func hasCELExpressions(p *pluginv1alpha1.Plugin) bool {
	for _, dh := range p.Spec.DomainHooks {
		if (dh.CEL != nil && dh.CEL.Expression != "") || dh.Condition != "" {
			return true
		}
	}
	for _, nh := range p.Spec.NodeHooks {
		if nh.Condition != "" {
			return true
		}
	}
	return false
}

func validateDomainHookCEL(eval *celutil.Evaluator, dh pluginv1alpha1.DomainHook, dhPath *field.Path) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if dh.CEL != nil && dh.CEL.Expression != "" {
		if err := eval.CompileMutation(dh.CEL.Expression); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL mutation expression: %s", err),
				Field:   dhPath.Child("cel", "expression").String(),
			})
		}
	}

	if dh.Condition != "" {
		if err := eval.CompileCondition(dh.Condition); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL condition expression: %s", err),
				Field:   dhPath.Child("condition").String(),
			})
		}
	}

	return causes
}

func validateNodeHookCEL(eval *nodecelutil.Evaluator, nh pluginv1alpha1.NodeHook, nhPath *field.Path) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if nh.Condition != "" {
		if err := eval.CompileCondition(nh.Condition); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL condition expression: %s", err),
				Field:   nhPath.Child("condition").String(),
			})
		}
	}

	return causes
}

func validateSidecarSocketPaths(p *pluginv1alpha1.Plugin) []metav1.StatusCause {
	var causes []metav1.StatusCause
	specPath := field.NewPath("spec")

	for i, dh := range p.Spec.DomainHooks {
		if dh.Sidecar == nil {
			continue
		}
		dhPath := specPath.Child("domainHooks").Index(i).Child("sidecar", "socketPath")
		sp := dh.Sidecar.SocketPath

		cleaned := filepath.Clean(sp)
		if cleaned != sp {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: "socketPath must be a clean path (no '..' segments or redundant separators)",
				Field:   dhPath.String(),
			})
		}
	}
	return causes
}
