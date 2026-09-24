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
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
)

func evalCELValue(expression string, object map[string]interface{}, vars map[string]interface{}) interface{} {
	env, err := cel.NewEnv(
		cel.Variable("object", cel.DynType),
		cel.Variable("variables", cel.DynType),
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	ast, issues := env.Compile(expression)
	ExpectWithOffset(1, issues.Err()).ToNot(HaveOccurred())

	prg, err := env.Program(ast)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	out, _, err := prg.Eval(map[string]interface{}{
		"object":    object,
		"variables": vars,
	})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return out.Value()
}

func evalCELWithVars(expression string, object map[string]interface{}, vars map[string]interface{}) bool {
	return evalCELValue(expression, object, vars).(bool)
}

func evaluatePolicy(policy *admissionregistrationv1.ValidatingAdmissionPolicy, obj map[string]interface{}) error {
	vars := map[string]interface{}{}
	for _, v := range policy.Spec.Variables {
		val := evalCELValue(v.Expression, obj, vars)
		vars[v.Name] = val
	}

	var msgs []string
	for _, v := range policy.Spec.Validations {
		if !evalCELWithVars(v.Expression, obj, vars) {
			msgs = append(msgs, v.Message)
		}
	}
	if len(msgs) > 0 {
		return fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	return nil
}
