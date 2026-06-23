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

package cel

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"

	v1 "kubevirt.io/api/core/v1"

	"libvirt.org/go/libvirtxml"
)

const costLimit = 10_000_000

type Evaluator struct {
	env *cel.Env
}

var (
	evaluatorOnce     sync.Once
	evaluatorInstance *Evaluator
)

func GetEvaluator() *Evaluator {
	evaluatorOnce.Do(func() {
		var err error
		evaluatorInstance, err = NewEvaluator()
		if err != nil {
			panic(fmt.Sprintf("failed to create CEL evaluator: %v", err))
		}
	})
	return evaluatorInstance
}

func NewEvaluator() (*Evaluator, error) {
	base, err := cel.NewEnv(
		ext.NativeTypes(
			reflect.TypeOf(&libvirtxml.Domain{}),
			reflect.TypeOf(&v1.VirtualMachineInstance{}),
		),
		cel.Container("libvirtxml"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating base CEL environment: %w", err)
	}

	wrapper := &sparseProvider{delegate: base.CELTypeProvider()}

	env, err := base.Extend(
		cel.CustomTypeProvider(wrapper),
		cel.Variable("vmi", cel.ObjectType("v1.VirtualMachineInstance")),
		cel.Variable("domainSpec", cel.ObjectType("libvirtxml.Domain")),
	)
	if err != nil {
		return nil, fmt.Errorf("extending CEL environment: %w", err)
	}

	return &Evaluator{env: env}, nil
}

func (e *Evaluator) EvaluateCondition(expr string, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) (bool, error) {
	ast, err := e.compile(expr)
	if err != nil {
		return false, err
	}
	if ast.OutputType() != cel.BoolType {
		return false, fmt.Errorf("condition must return bool, got %s", ast.OutputType())
	}
	out, err := e.eval(ast, vmi, domain)
	if err != nil {
		return false, err
	}

	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("condition returned %T, expected bool", out.Value())
	}
	return result, nil
}

func (e *Evaluator) EvaluateMutation(expr string, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) (*libvirtxml.Domain, error) {
	ast, err := e.compile(expr)
	if err != nil {
		return nil, err
	}
	out, err := e.eval(ast, vmi, domain)
	if err != nil {
		return nil, err
	}

	partial, ok := out.(*objectVal)
	if !ok {
		return nil, fmt.Errorf("mutation must return a Domain object, got %T", out)
	}

	result, err := copyDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("copying domain: %w", err)
	}
	if err := deepMerge(reflect.ValueOf(result).Elem(), partial); err != nil {
		return nil, fmt.Errorf("merging mutation result: %w", err)
	}
	return result, nil
}

func (e *Evaluator) CompileCondition(expr string) error {
	ast, err := e.compile(expr)
	if err != nil {
		return err
	}
	if ast.OutputType() != cel.BoolType {
		return fmt.Errorf("condition must return bool, got %s", ast.OutputType())
	}
	return nil
}

func (e *Evaluator) CompileMutation(expr string) error {
	ast, err := e.compile(expr)
	if err != nil {
		return err
	}
	if !ast.OutputType().IsEquivalentType(cel.ObjectType("libvirtxml.Domain")) {
		return fmt.Errorf("mutation must return Domain, got %s", ast.OutputType())
	}
	return nil
}

func (e *Evaluator) compile(expr string) (*cel.Ast, error) {
	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compiling expression: %w", issues.Err())
	}
	return ast, nil
}

func (e *Evaluator) eval(ast *cel.Ast, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) (ref.Val, error) {
	prg, err := e.env.Program(ast, cel.CostLimit(costLimit))
	if err != nil {
		return nil, fmt.Errorf("creating program: %w", err)
	}

	out, _, err := prg.Eval(map[string]any{
		"vmi":        vmi,
		"domainSpec": domain,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluating expression: %w", err)
	}
	return out, nil
}

func copyDomain(src *libvirtxml.Domain) (*libvirtxml.Domain, error) {
	xmlStr, err := src.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshaling domain for copy: %w", err)
	}
	dst := &libvirtxml.Domain{}
	if err := dst.Unmarshal(xmlStr); err != nil {
		return nil, fmt.Errorf("unmarshaling domain for copy: %w", err)
	}
	return dst, nil
}
