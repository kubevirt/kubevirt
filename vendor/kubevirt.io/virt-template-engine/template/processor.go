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
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/virt-template-api/core/v1alpha1"
	"kubevirt.io/virt-template-engine/template/generator"
)

// processor processes a VirtualMachineTemplate into a VirtualMachine with substituted parameters.
type processor struct {
	generators map[string]generator.Generator
}

var (
	defaultProcessor *processor
	once             sync.Once
)

// GetDefaultProcessor creates a new default processor once and initializes its set of generators.
// Then it returns the default processor.
func GetDefaultProcessor() *processor {
	once.Do(func() {
		defaultProcessor = &processor{
			generators: map[string]generator.Generator{
				"expression": &generator.ExpressionValue{},
			},
		}
	})
	return defaultProcessor
}

// Process processes a VirtualMachineTemplate into a VirtualMachine. It generates
// parameter values using the defined set of generators first, and then it
// substitutes all parameter expression occurrences in the template VirtualMachine
// with their corresponding values. The template VirtualMachine can be
// supplied as raw JSON, as an unstructured.Unstructured or as a VirtualMachine object.
// Hardcoded namespaces in the template VirtualMachine are removed before substituting
// parameter expressions, so it is left up to the user in which namespace to create the
// resulting VirtualMachine. The message of the template is also processed and expressions
// in it are replaced.
func (p *processor) Process(tpl *v1alpha1.VirtualMachineTemplate) (*virtv1.VirtualMachine, string, *field.Error) {
	params, gErr := generateParameterValues(tpl.Spec.Parameters, p.generators)
	if gErr != nil {
		return nil, "", gErr
	}

	obj, gErr := getVirtualMachineObject(&tpl.Spec)
	if gErr != nil {
		return nil, "", gErr
	}

	// If an object definition's metadata includes a hardcoded namespace field, the field will be removed
	// before substituting parameters. Namespace fields that contain a ${PARAMETER_REFERENCE}
	// will be left in place and will be resolved during the parameter substitution.
	if rErr := removeHardcodedNamespace(obj); rErr != nil {
		return nil, "", field.InternalError(field.NewPath("spec", "virtualMachine"),
			fmt.Errorf("error removing hardcoded namespace: %w", rErr))
	}

	if err := substituteAllParameters(obj, params); err != nil {
		return nil, "", field.Invalid(field.NewPath("spec", "parameters"),
			tpl.Spec.Parameters, fmt.Sprintf("error processing template: %v", err))
	}

	vm := &virtv1.VirtualMachine{}
	switch typedObj := obj.(type) {
	case *virtv1.VirtualMachine:
		vm = typedObj
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructuredWithValidation(typedObj.Object, vm, true); err != nil {
			return nil, "", field.Invalid(field.NewPath("spec", "virtualMachine"),
				typedObj, fmt.Sprintf("failed to convert unstructured object to VirtualMachine: %v", err))
		}
	default:
		return nil, "", field.Invalid(field.NewPath("spec", "virtualMachine"),
			typedObj, fmt.Sprintf("unable to convert into VirtualMachine: object is %T", typedObj))
	}

	// Ensure we have a valid GVK
	vm.SetGroupVersionKind(virtv1.VirtualMachineGroupVersionKind)

	// Perform parameter substitution on the template's user message. This can be used to
	// instruct a user on next steps for the returned VirtualMachine.
	msg, _, err := substituteParameters(tpl.Spec.Message, params)
	if err != nil {
		return nil, "", field.Invalid(field.NewPath("spec", "message"),
			tpl.Spec.Message, fmt.Sprintf("error processing message: %v", err))
	}

	return vm, msg, nil
}
