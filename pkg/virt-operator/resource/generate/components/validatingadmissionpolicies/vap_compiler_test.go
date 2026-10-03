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

package validatingadmissionpolicies_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	admissioncel "k8s.io/apiserver/pkg/admission/plugin/cel"
	"k8s.io/apiserver/pkg/admission/plugin/policy/validating"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/matchconditions"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/cel/environment"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
)

type outcome string

const (
	admit outcome = "admit"
	deny  outcome = "deny"
	skip  outcome = "skip"
)

var nodeGVR = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "nodes"}

var _ = Describe("VAP compiler infrastructure", func() {
	It("should compile the handler node-restriction policy", func() {
		const handlerSA = "system:serviceaccount:kubevirt:kubevirt-handler"
		policy := vap.NewHandlerV1ValidatingAdmissionPolicy(handlerSA)
		Expect(policy.Spec.Variables).NotTo(BeEmpty())
		Expect(policy.Spec.Validations).NotTo(BeEmpty())

		validator := compilePolicy(policy)
		Expect(validator.CompileError()).NotTo(HaveOccurred())
	})
})

type stubAuthorizer struct{}

func (stubAuthorizer) Authorize(context.Context, authorizer.Attributes) (authorizer.Decision, string, error) {
	return authorizer.DecisionAllow, "stub", nil
}

func compilePolicy(policy *admissionregistrationv1.ValidatingAdmissionPolicy) validating.Validator {
	optionalVars := admissioncel.OptionalVariableDeclarations{HasParams: false, HasAuthorizer: true}
	expressionOptionalVars := admissioncel.OptionalVariableDeclarations{HasParams: false, HasAuthorizer: false}

	filterCompiler, err := admissioncel.NewCompositedCompiler(environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion()))
	Expect(err).NotTo(HaveOccurred())

	named := make([]admissioncel.NamedExpressionAccessor, len(policy.Spec.Variables))
	for i, variable := range policy.Spec.Variables {
		named[i] = &validating.Variable{Name: variable.Name, Expression: variable.Expression}
	}
	filterCompiler.CompileAndStoreVariables(named, optionalVars, environment.StoredExpressions)

	var matcher matchconditions.Matcher
	if len(policy.Spec.MatchConditions) > 0 {
		accessors := make([]admissioncel.ExpressionAccessor, len(policy.Spec.MatchConditions))
		for i := range policy.Spec.MatchConditions {
			accessors[i] = (*matchconditions.MatchCondition)(&policy.Spec.MatchConditions[i])
		}
		matchFilter := filterCompiler.CompileCondition(accessors, optionalVars, environment.StoredExpressions)
		Expect(matchFilter.CompilationErrors()).To(BeEmpty())
		matcher = matchconditions.NewMatcher(matchFilter, policy.Spec.FailurePolicy, "policy", "validate", policy.Name)
	}

	validations := convertValidations(policy.Spec.Validations)
	validationFilter := filterCompiler.CompileCondition(validations, optionalVars, environment.StoredExpressions)
	Expect(validationFilter.CompilationErrors()).To(BeEmpty())
	auditFilter := filterCompiler.CompileCondition(nil, optionalVars, environment.StoredExpressions)
	messageFilter := filterCompiler.CompileCondition(convertMessageExpressions(policy.Spec.Validations), expressionOptionalVars, environment.StoredExpressions)

	validator := validating.NewValidator(validationFilter, matcher, auditFilter, messageFilter, policy.Spec.FailurePolicy, nil)
	Expect(validator.CompileError()).NotTo(HaveOccurred())
	return validator
}

func convertValidations(in []admissionregistrationv1.Validation) []admissioncel.ExpressionAccessor {
	out := make([]admissioncel.ExpressionAccessor, len(in))
	for i, v := range in {
		cond := validating.ValidationCondition{Expression: v.Expression, Message: v.Message, Reason: v.Reason}
		out[i] = &cond
	}
	return out
}

func convertMessageExpressions(in []admissionregistrationv1.Validation) []admissioncel.ExpressionAccessor {
	out := make([]admissioncel.ExpressionAccessor, len(in))
	for i, v := range in {
		if v.MessageExpression != "" {
			out[i] = &validating.MessageExpressionCondition{MessageExpression: v.MessageExpression}
		}
	}
	return out
}

func evaluate(validator validating.Validator, object, oldObject *corev1.Node, info user.Info) (outcome, []string) {
	attrs := &admission.VersionedAttributes{
		Attributes: admission.NewAttributesRecord(
			object,
			oldObject,
			corev1.SchemeGroupVersion.WithKind("Node"),
			object.Namespace,
			object.Name,
			nodeGVR,
			"",
			admission.Update,
			nil,
			true,
			info,
		),
		VersionedOldObject: oldObject,
		VersionedObject:    object,
		VersionedKind:      corev1.SchemeGroupVersion.WithKind("Node"),
	}
	result := validator.Validate(context.Background(), nodeGVR, attrs, nil, nil, celconfig.RuntimeCELCostBudget, stubAuthorizer{})
	if len(result.Decisions) == 0 {
		return skip, nil
	}
	var messages []string
	denied := false
	for _, d := range result.Decisions {
		if d.Action == validating.ActionDeny || d.Evaluation == validating.EvalError {
			denied = true
			if d.Message != "" {
				messages = append(messages, d.Message)
			}
		}
	}
	if denied {
		return deny, messages
	}
	return admit, messages
}

func containsMessage(messages []string, want string) bool {
	for _, m := range messages {
		if strings.Contains(m, want) {
			return true
		}
	}
	return false
}
