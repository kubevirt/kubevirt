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

package template

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/virt-template-api/core/v1alpha1"
	"kubevirt.io/virt-template-engine/template/generator"
)

var (
	// match expressions in the form of ${KEY}
	stringParamExpr = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)
	// match expressions in the form of ${{KEY}}
	nonStringParamExpr = regexp.MustCompile(`^\$\{\{([a-zA-Z0-9_]+)\}\}$`)
)

// generateParameterValues generates values for each parameter that has
// the Generate field specified and where its Value is empty.
// Returned errors relate to the template that is being processed,
// therefore field paths start with 'spec'.
func generateParameterValues(
	parameters []v1alpha1.Parameter,
	generators map[string]generator.Generator,
) (map[string]v1alpha1.Parameter, *field.Error) {
	visited := make(map[string]struct{})
	params := make(map[string]v1alpha1.Parameter)
	for i, param := range parameters {
		path := field.NewPath("spec", "parameters").Index(i)

		if param.Name == "" {
			return nil, field.Invalid(path.Child("name"), param.Name, "parameter name is empty")
		}
		if _, found := visited[param.Name]; found {
			return nil, field.Duplicate(path.Child("name"), param.Name)
		}
		visited[param.Name] = struct{}{}

		newParam := param.DeepCopy()
		if newParam.Value == "" && newParam.Generate != "" {
			g, ok := generators[newParam.Generate]
			if !ok {
				return nil, field.Invalid(path.Child("generate"), newParam.Generate,
					fmt.Sprintf("unknown generator name '%v' for parameter '%s'", newParam.Generate, newParam.Name),
				)
			}
			if newParam.From == "" {
				return nil, field.Invalid(path.Child("from"), newParam.From,
					fmt.Sprintf("from cannot be empty for parameter '%s' using generator '%s'", newParam.Name, newParam.Generate),
				)
			}

			var err error
			newParam.Value, err = g.GenerateValue(newParam.From)
			if err != nil {
				return nil, field.Invalid(path.Child("from"), newParam.From, err.Error())
			}
		}

		if newParam.Value == "" && newParam.Required {
			return nil, field.Required(path.Child("value"),
				fmt.Sprintf("parameter '%s' is required and a value must be specified", param.Name),
			)
		}

		params[newParam.Name] = *newParam
	}

	return params, nil
}

// getVirtualMachineObject extracts the VirtualMachine runtime.Object from the spec of a VirtualMachineTemplate.
// It handles both Raw JSON bytes and embedded Object representations.
func getVirtualMachineObject(tplSpec *v1alpha1.VirtualMachineTemplateSpec) (runtime.Object, *field.Error) {
	if tplSpec.VirtualMachine == nil || (len(tplSpec.VirtualMachine.Raw) == 0 && tplSpec.VirtualMachine.Object == nil) {
		return nil, field.Invalid(field.NewPath("spec", "virtualMachine"),
			tplSpec.VirtualMachine, "virtualMachine is required and cannot be empty")
	}

	if len(tplSpec.VirtualMachine.Raw) == 0 {
		return tplSpec.VirtualMachine.Object.DeepCopyObject(), nil
	}

	obj, err := decode(tplSpec.VirtualMachine.Raw)
	if err != nil {
		return nil, field.Invalid(field.NewPath("spec", "virtualMachine", "raw"),
			tplSpec.VirtualMachine.Raw, fmt.Sprintf("error decoding virtualMachine: %v", err))
	}

	return obj, nil
}

func decode(raw []byte) (runtime.Object, error) {
	// Do not use runtime.Decode and unstructured.UnstructuredJSONScheme
	// so we can ignore missing apiVersion and kind. Those will be forced later.
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

// removeHardcodedNamespace removes the namespace from an object if it is
// not empty and not parametrized.
func removeHardcodedNamespace(obj runtime.Object) error {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	if objMeta.GetNamespace() != "" && !stringParamExpr.MatchString(objMeta.GetNamespace()) {
		objMeta.SetNamespace("")
	}

	return nil
}

// substituteAllParameters recursively visits all string values of an object and substitutes parameters.
func substituteAllParameters(obj runtime.Object, params map[string]v1alpha1.Parameter) error {
	return visitValue(reflect.ValueOf(obj), func(in string) (string, bool, error) {
		return substituteParameters(in, params)
	})
}

// substituteParameters replaces parameters in a string with values from the provided map.
// It returns the substituted value (if any substitution applied) and a boolean
// indicating if the resulting value should be treated as a string(true) or a non-string
// value(false).
func substituteParameters(in string, params map[string]v1alpha1.Parameter) (out string, asString bool, err error) {
	// First check if the value matches the "${{KEY}}" substitution syntax, which
	// means replace and drop the quotes because the parameter value is to be used
	// as a non-string value. If we hit a match here, we're done because the
	// "${{KEY}}" syntax is exact match only, it cannot be used in a value like
	// "FOO_${{KEY}}_BAR", no substitution will be performed if it is used in that way.
	if match := nonStringParamExpr.FindStringSubmatch(in); len(match) > 1 {
		if param, found := params[match[1]]; found {
			return strings.Replace(in, match[0], param.Value, 1), false, nil
		} else {
			return "", false, fmt.Errorf("found parameter '%s' but it was not defined", match[1])
		}
	}

	// If we didn't do a non-string substitution above, do normal string substitution
	// on the value here if it contains a "${KEY}" reference. This substitution does
	// allow multiple matches and prefix/postfix, e.g. "FOO_${KEY1}_${KEY2}_BAR".
	out = in
	for _, match := range stringParamExpr.FindAllStringSubmatch(out, -1) {
		if len(match) > 1 {
			if param, found := params[match[1]]; found {
				out = strings.Replace(out, match[0], param.Value, 1)
			} else {
				return "", false, fmt.Errorf("found parameter '%s' but it was not defined", match[1])
			}
		}
	}

	return out, true, nil
}

// collectAllReferencedParameters recursively visits all string values in an object
// and collects all referenced parameters.
func collectAllReferencedParameters(obj runtime.Object) (map[string]struct{}, error) {
	params := map[string]struct{}{}
	err := visitValue(reflect.ValueOf(obj), func(in string) (string, bool, error) {
		for param := range collectReferencedParameters(in) {
			params[param] = struct{}{}
		}
		return in, true, nil
	})

	return params, err
}

// collectReferencedParameters extracts all parameter names referenced in a string.
// It checks for both ${KEY} and ${{KEY}} patterns.
func collectReferencedParameters(in string) map[string]struct{} {
	params := map[string]struct{}{}

	if match := nonStringParamExpr.FindStringSubmatch(in); len(match) > 1 {
		params[match[1]] = struct{}{}
	}

	for _, match := range stringParamExpr.FindAllStringSubmatch(in, -1) {
		if len(match) > 1 {
			params[match[1]] = struct{}{}
		}
	}

	return params
}
