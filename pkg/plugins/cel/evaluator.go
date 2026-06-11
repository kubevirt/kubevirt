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
	"strings"
	"sync"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/ext"

	v1 "kubevirt.io/api/core/v1"
)

const costLimit = 10_000_000

type variable struct {
	name string
	typ  *cel.Type
}

type config struct {
	variables   []variable
	containers  []string
	nativeTypes []reflect.Type
}

type Option func(*config)

func WithVariable(name string, typ *cel.Type) Option {
	return func(c *config) {
		c.variables = append(c.variables, variable{name: name, typ: typ})
	}
}

func WithContainer(name string) Option {
	return func(c *config) {
		c.containers = append(c.containers, name)
	}
}

func WithNativeTypes(types ...reflect.Type) Option {
	return func(c *config) {
		c.nativeTypes = append(c.nativeTypes, types...)
	}
}

type Evaluator struct {
	env   *cel.Env
	mu    sync.RWMutex
	cache map[string]cel.Program
}

func NewEvaluator(opts ...Option) (*Evaluator, error) {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}

	nativeTypes := []any{reflect.TypeFor[*v1.VirtualMachineInstance]()}
	for _, t := range cfg.nativeTypes {
		nativeTypes = append(nativeTypes, t)
	}

	envOpts := []cel.EnvOption{
		ext.NativeTypes(append(nativeTypes, ext.ParseStructField(func(field reflect.StructField) string {
			if tag, ok := field.Tag.Lookup("json"); ok {
				if name := strings.SplitN(tag, ",", 2)[0]; name != "" {
					return name
				}
			}
			return field.Name
		}))...),
		ext.Strings(),
		ext.Math(),
		ext.Lists(),
		cel.Variable("vmi", cel.ObjectType("v1.VirtualMachineInstance")),
		cel.CrossTypeNumericComparisons(true),
	}

	for _, container := range cfg.containers {
		envOpts = append(envOpts, cel.Container(container))
	}

	for _, v := range cfg.variables {
		envOpts = append(envOpts, cel.Variable(v.name, v.typ))
	}

	env, err := cel.NewEnv(envOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Evaluator{
		env:   env,
		cache: make(map[string]cel.Program),
	}, nil
}

func (e *Evaluator) CompileCondition(expr string) error {
	_, err := e.compile(expr)
	return err
}

func (e *Evaluator) EvaluateCondition(expr string, vars map[string]any) (bool, error) {
	if expr == "" {
		return true, nil
	}

	prg, err := e.getOrCompile(expr)
	if err != nil {
		return false, err
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("CEL evaluation failed: %w", err)
	}

	b, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("CEL expression must return bool, got %T", out.Value())
	}
	return b, nil
}

func (e *Evaluator) getOrCompile(expr string) (cel.Program, error) {
	e.mu.RLock()
	prg, ok := e.cache[expr]
	e.mu.RUnlock()
	if ok {
		return prg, nil
	}

	prg, err := e.compile(expr)
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	e.cache[expr] = prg
	e.mu.Unlock()

	return prg, nil
}

func (e *Evaluator) compile(expr string) (cel.Program, error) {
	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("CEL compilation failed: %w", issues.Err())
	}

	if ast.OutputType() != cel.BoolType {
		return nil, fmt.Errorf("CEL expression must return bool, got %s", ast.OutputType())
	}

	prg, err := e.env.Program(ast, cel.CostLimit(costLimit))
	if err != nil {
		return nil, fmt.Errorf("CEL program creation failed: %w", err)
	}

	return prg, nil
}
