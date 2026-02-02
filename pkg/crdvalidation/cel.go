/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package crdvalidation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// CELValidator validates objects against CEL rules defined in x-kubernetes-validations
type CELValidator struct {
	// envCache caches compiled CEL environments per rule to avoid recompilation
	envCache map[string]cel.Program
}

// NewCELValidator creates a new CEL validator
func NewCELValidator() *CELValidator {
	return &CELValidator{
		envCache: make(map[string]cel.Program),
	}
}

// ValidateRules evaluates CEL rules against a value
// self is the current value at the schema location
// oldSelf is the previous value (for transition rules, nil on create)
func (c *CELValidator) ValidateRules(
	rules []extv1.ValidationRule, self, oldSelf any, path string,
) ValidationErrors {
	var errs ValidationErrors

	for _, rule := range rules {
		err := c.evaluateRule(rule, self, oldSelf, path)
		if err != nil {
			errs = append(errs, *err)
		}
	}

	return errs
}

// evaluateRule evaluates a single CEL rule
func (c *CELValidator) evaluateRule(rule extv1.ValidationRule, self, oldSelf any, path string) *ValidationError {
	// Check if rule references oldSelf (transition rule)
	isTransitionRule := strings.Contains(rule.Rule, "oldSelf")

	// Skip transition rules if oldSelf is nil (create operation)
	if isTransitionRule && oldSelf == nil {
		return nil
	}

	program, err := c.compileRule(rule.Rule, isTransitionRule)
	if err != nil {
		return &ValidationError{
			Path:    path,
			Message: fmt.Sprintf("failed to compile CEL rule: %v", err),
			Type:    ErrorTypeCEL,
			Rule:    rule.Rule,
		}
	}

	// Prepare input variables
	vars := map[string]any{
		"self": convertToCELValue(self),
	}
	if isTransitionRule {
		vars["oldSelf"] = convertToCELValue(oldSelf)
	}

	// Evaluate the rule
	result, _, err := program.Eval(vars)
	if err != nil {
		return &ValidationError{
			Path:    path,
			Message: fmt.Sprintf("failed to evaluate CEL rule: %v", err),
			Type:    ErrorTypeCEL,
			Rule:    rule.Rule,
		}
	}

	// Check if the result is true (validation passed)
	if result.Type() == types.BoolType {
		if result.Value().(bool) {
			return nil // Validation passed
		}
	}

	// Validation failed
	message := rule.Message
	if message == "" {
		message = fmt.Sprintf("failed CEL validation rule: %s", rule.Rule)
	}

	return &ValidationError{
		Path:    path,
		Message: message,
		Type:    ErrorTypeCEL,
		Rule:    rule.Rule,
	}
}

// compileRule compiles a CEL expression into a program
func (c *CELValidator) compileRule(rule string, hasOldSelf bool) (cel.Program, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%v", rule, hasOldSelf)
	if prog, ok := c.envCache[cacheKey]; ok {
		return prog, nil
	}

	// Create CEL environment with self and optionally oldSelf
	envOpts := []cel.EnvOption{
		cel.Variable("self", cel.DynType),
	}
	if hasOldSelf {
		envOpts = append(envOpts, cel.Variable("oldSelf", cel.DynType))
	}

	env, err := cel.NewEnv(envOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	// Parse and check the expression
	ast, issues := env.Compile(rule)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression: %w", issues.Err())
	}

	// Create program
	prog, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	// Cache the program
	c.envCache[cacheKey] = prog

	return prog, nil
}

// convertToCELValue converts a Go value to a CEL-compatible value
func convertToCELValue(v any) any {
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	//exhaustive:ignore
	switch rv.Kind() {
	case reflect.Map:
		// Convert map[string]interface{} to map[string]any
		result := make(map[string]any)
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key().Interface().(string)
			result[key] = convertToCELValue(iter.Value().Interface())
		}
		return result
	case reflect.Slice:
		// Convert []interface{} to []any
		result := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = convertToCELValue(rv.Index(i).Interface())
		}
		return result
	default:
		return v
	}
}

// validateCELAtPath walks the schema and validates CEL rules at each level
//
//nolint:gocyclo
func (c *CELValidator) validateCELAtPath(
	schema *extv1.JSONSchemaProps,
	value any,
	oldValue any,
	path string,
) ValidationErrors {
	var errs ValidationErrors

	if schema == nil || value == nil {
		return errs
	}

	// Validate CEL rules at this level
	if len(schema.XValidations) > 0 {
		errs = append(errs, c.ValidateRules(schema.XValidations, value, oldValue, path)...)
	}

	// Recursively validate nested structures
	switch {
	case schema.Type == "object" && schema.Properties != nil:
		valueMap, ok := value.(map[string]any)
		if !ok {
			return errs
		}
		var oldValueMap map[string]any
		if oldValue != nil {
			oldValueMap, _ = oldValue.(map[string]any)
		}

		for propName, propSchema := range schema.Properties {
			propValue, exists := valueMap[propName]
			if !exists {
				continue
			}
			var propOldValue any
			if oldValueMap != nil {
				propOldValue = oldValueMap[propName]
			}
			propPath := path + "." + propName
			propSchemaCopy := propSchema // Create a copy to take address of
			errs = append(errs, c.validateCELAtPath(&propSchemaCopy, propValue, propOldValue, propPath)...)
		}

	case schema.Type == "array" && schema.Items != nil && schema.Items.Schema != nil:
		valueSlice, ok := value.([]any)
		if !ok {
			return errs
		}
		var oldValueSlice []any
		if oldValue != nil {
			oldValueSlice, _ = oldValue.([]any)
		}

		for i, itemValue := range valueSlice {
			var itemOldValue any
			if oldValueSlice != nil && i < len(oldValueSlice) {
				itemOldValue = oldValueSlice[i]
			}
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			errs = append(errs, c.validateCELAtPath(schema.Items.Schema, itemValue, itemOldValue, itemPath)...)
		}
	}

	return errs
}
