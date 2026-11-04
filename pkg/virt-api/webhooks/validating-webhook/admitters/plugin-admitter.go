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
	"slices"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/api/plugin"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	plugincel "kubevirt.io/kubevirt/pkg/plugins/cel"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	launchercel "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/plugins/cel"
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
	return validatePluginCEL(p)
}

func validatePluginCEL(p *pluginv1alpha1.Plugin) []metav1.StatusCause {
	if !hasCELExpressions(p) {
		return nil
	}

	specPath := field.NewPath("spec")
	conditionEval, err := plugincel.GetEvaluator()
	if err != nil {
		return []metav1.StatusCause{{
			Type:    metav1.CauseTypeFieldValueInvalid,
			Message: fmt.Sprintf("failed to create VMI condition evaluator: %s", err),
			Field:   specPath.String(),
		}}
	}

	var causes []metav1.StatusCause
	if p.Spec.Condition != "" {
		if err := conditionEval.CompileCondition(p.Spec.Condition); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL condition expression: %s", err),
				Field:   specPath.Child("condition").String(),
			})
		}
	}

	for i, launcherHook := range p.Spec.LauncherHooks {
		causes = append(causes, validateLauncherHookCEL(launchercel.GetEvaluator(), conditionEval, launcherHook, specPath.Child("launcherHooks").Index(i))...)
	}

	for i, nodeHook := range p.Spec.NodeHooks {
		causes = append(causes, validateNodeHookCEL(conditionEval, nodeHook, specPath.Child("nodeHooks").Index(i))...)
	}

	return causes
}

func hasCELExpressions(p *pluginv1alpha1.Plugin) bool {
	if p.Spec.Condition != "" {
		return true
	}
	if slices.ContainsFunc(p.Spec.NodeHooks, func(nodeHook pluginv1alpha1.NodeHook) bool { return nodeHook.Condition != "" }) {
		return true
	}
	return slices.ContainsFunc(p.Spec.LauncherHooks, func(launcherHook pluginv1alpha1.LauncherHook) bool {
		return (launcherHook.CEL != nil && launcherHook.CEL.Expression != "") || launcherHook.Condition != ""
	})
}

func validateLauncherHookCEL(launcherEval *launchercel.Evaluator, conditionEval *plugincel.Evaluator, launcherHook pluginv1alpha1.LauncherHook, hookPath *field.Path) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if launcherHook.CEL != nil && launcherHook.CEL.Expression != "" {
		if err := launcherEval.CompileMutation(launcherHook.CEL.Expression); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL mutation expression: %s", err),
				Field:   hookPath.Child("cel", "expression").String(),
			})
		}
	}

	if launcherHook.Condition != "" {
		if err := conditionEval.CompileCondition(launcherHook.Condition); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL condition expression: %s", err),
				Field:   hookPath.Child("condition").String(),
			})
		}
	}

	return causes
}

func validateNodeHookCEL(eval *plugincel.Evaluator, nodeHook pluginv1alpha1.NodeHook, hookPath *field.Path) []metav1.StatusCause {
	var causes []metav1.StatusCause

	if nodeHook.Condition != "" {
		if err := eval.CompileCondition(nodeHook.Condition); err != nil {
			causes = append(causes, metav1.StatusCause{
				Type:    metav1.CauseTypeFieldValueInvalid,
				Message: fmt.Sprintf("invalid CEL condition expression: %s", err),
				Field:   hookPath.Child("condition").String(),
			})
		}
	}

	return causes
}
