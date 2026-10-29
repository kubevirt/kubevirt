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
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

// admissionInput is the set of CEL variables a ValidatingAdmissionPolicy can see.
// Nil oldObject and params are exposed as CEL null, matching CREATE requests and
// bindings that did not resolve a parameter object.
type admissionInput struct {
	object    map[string]interface{}
	oldObject interface{}
	params    interface{}
	request   interface{}
}

func (in admissionInput) activation(vars map[string]interface{}) map[string]interface{} {
	oldObject := in.oldObject
	if oldObject == nil {
		oldObject = types.NullValue
	}
	params := in.params
	if params == nil {
		params = types.NullValue
	}
	request := in.request
	if request == nil {
		request = map[string]interface{}{}
	}
	return map[string]interface{}{
		"object":    in.object,
		"oldObject": oldObject,
		"params":    params,
		"request":   request,
		"variables": vars,
	}
}

func evalCELValue(expression string, in admissionInput, vars map[string]interface{}) interface{} {
	env, err := cel.NewEnv(
		cel.Variable("object", cel.DynType),
		cel.Variable("oldObject", cel.DynType),
		cel.Variable("params", cel.DynType),
		cel.Variable("request", cel.DynType),
		cel.Variable("variables", cel.DynType),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	ast, issues := env.Compile(expression)
	ExpectWithOffset(1, issues.Err()).ToNot(HaveOccurred())

	prg, err := env.Program(ast)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	out, _, err := prg.Eval(in.activation(vars))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return out.Value()
}

func evaluatePolicy(policy *admissionregistrationv1.ValidatingAdmissionPolicy, obj map[string]interface{}) error {
	return evaluatePolicyInput(policy, admissionInput{object: obj})
}

func evaluatePolicyInput(policy *admissionregistrationv1.ValidatingAdmissionPolicy, in admissionInput) error {
	vars := map[string]interface{}{}
	for _, v := range policy.Spec.Variables {
		vars[v.Name] = evalCELValue(v.Expression, in, vars)
	}

	var msgs []string
	for _, v := range policy.Spec.Validations {
		if !evalCELValue(v.Expression, in, vars).(bool) {
			msgs = append(msgs, v.Message)
		}
	}
	if len(msgs) > 0 {
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	return nil
}

// policyApplies reports whether every match condition is true. An empty list
// matches every object, which is how the API server treats a policy with no
// match conditions.
func policyApplies(policy *admissionregistrationv1.ValidatingAdmissionPolicy, in admissionInput) bool {
	for _, condition := range policy.Spec.MatchConditions {
		if !evalCELValue(condition.Expression, in, map[string]interface{}{}).(bool) {
			return false
		}
	}
	return true
}
