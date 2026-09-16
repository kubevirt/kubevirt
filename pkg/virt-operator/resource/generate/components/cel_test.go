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

package components

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	. "github.com/onsi/gomega"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func crdSchema(crdFunc func() (*extv1.CustomResourceDefinition, error), versionName string) *extv1.JSONSchemaProps {
	crd, err := crdFunc()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	var schema *extv1.JSONSchemaProps
	for _, version := range crd.Spec.Versions {
		if version.Name != versionName {
			continue
		}
		if version.Schema != nil {
			schema = version.Schema.OpenAPIV3Schema
		}
		break
	}
	ExpectWithOffset(1, schema).ToNot(BeNil(), "CRD has no schema for version %q", versionName)
	return schema
}

// schemaAt walks a dotted path of property names. A "[]" suffix descends into the array item
// schema, which is where controller-gen places a rule declared on a named element type.
func schemaAt(root *extv1.JSONSchemaProps, path string) *extv1.JSONSchemaProps {
	ExpectWithOffset(1, root).ToNot(BeNil())
	node := root
	if path == "" {
		return node
	}
	for _, segment := range strings.Split(path, ".") {
		name := strings.TrimSuffix(segment, "[]")
		next, found := node.Properties[name]
		ExpectWithOffset(1, found).To(BeTrue(), "no property %q while resolving %q", name, path)
		node = &next
		if strings.HasSuffix(segment, "[]") {
			ExpectWithOffset(1, node.Items).ToNot(BeNil(), "property %q is not an array in %q", name, path)
			node = node.Items.Schema
			ExpectWithOffset(1, node).ToNot(BeNil(), "array %q has no item schema in %q", name, path)
		}
	}
	return node
}

func evalCELRule(rule string, self any) bool {
	env, err := cel.NewEnv(cel.Variable("self", cel.DynType))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	ast, issues := env.Compile(rule)
	ExpectWithOffset(1, issues.Err()).ToNot(HaveOccurred(), "rule does not compile: %s", rule)

	prg, err := env.Program(ast)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	out, _, err := prg.Eval(map[string]interface{}{"self": self})
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "rule does not evaluate: %s", rule)

	result, ok := out.Value().(bool)
	ExpectWithOffset(1, ok).To(BeTrue(), "rule is not boolean: %s", rule)
	return result
}

func validateCELRules(rules []extv1.ValidationRule, self any) error {
	ExpectWithOffset(1, rules).ToNot(BeEmpty())

	var messages []string
	for _, rule := range rules {
		if !evalCELRule(rule.Rule, self) {
			messages = append(messages, rule.Message)
		}
	}
	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, "; "))
	}
	return nil
}

func validateCELAt(root *extv1.JSONSchemaProps, path string, self any) error {
	return validateCELRules(schemaAt(root, path).XValidations, self)
}

func unstructuredOf(obj any) any {
	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return out
}
