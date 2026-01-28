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
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"kubevirt.io/virt-template-api/core/v1alpha1"
)

func MergeParameters(tplParams []v1alpha1.Parameter, params map[string]string) ([]v1alpha1.Parameter, error) {
	newTplParams := slices.Clone(tplParams)
	for k, v := range params {
		found := false
		for i := range newTplParams {
			if newTplParams[i].Name == k {
				newTplParams[i].Value = v
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("parameter %s not found in template", k)
		}
	}
	return newTplParams, nil
}

// ValidateParameterReferences validates that all defined parameters are referenced
// in the template and that all referenced parameters are defined.
// Returns warnings for unused parameters and errors for undefined parameter references.
func ValidateParameterReferences(tpl *v1alpha1.VirtualMachineTemplate) ([]string, field.ErrorList) {
	obj, err := getVirtualMachineObject(&tpl.Spec)
	if err != nil {
		return nil, field.ErrorList{err}
	}

	referencedParams, cErr := collectAllReferencedParameters(obj)
	if cErr != nil {
		return nil, field.ErrorList{
			field.Invalid(
				field.NewPath("spec", "virtualMachine"),
				tpl.Spec.VirtualMachine,
				fmt.Sprintf("failed to collect parameter references: %v", cErr),
			),
		}
	}
	for param := range collectReferencedParameters(tpl.Spec.Message) {
		referencedParams[param] = struct{}{}
	}

	definedParams := map[string]int{}
	for i, param := range tpl.Spec.Parameters {
		definedParams[param.Name] = i
	}

	var warnings []string
	for param, i := range definedParams {
		if _, referenced := referencedParams[param]; !referenced {
			path := field.NewPath("spec", "parameters").Index(i).Child("name")
			warnings = append(warnings, fmt.Sprintf("%s: %s is defined but never referenced", path.String(), param))
		}
	}

	var errs field.ErrorList
	for param := range referencedParams {
		if _, defined := definedParams[param]; !defined {
			errs = append(errs, field.Invalid(
				field.NewPath("spec", "virtualMachine"),
				param,
				fmt.Sprintf("references undefined parameter %s", param),
			))
		}
	}

	return warnings, errs
}
