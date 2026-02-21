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
	"strings"
)

// ValidationErrorType categorizes the type of validation error
type ValidationErrorType string

const (
	// ErrorTypePattern indicates a regex pattern validation failure
	ErrorTypePattern ValidationErrorType = "pattern"
	// ErrorTypeEnum indicates an enum validation failure
	ErrorTypeEnum ValidationErrorType = "enum"
	// ErrorTypeRequired indicates a required field is missing
	ErrorTypeRequired ValidationErrorType = "required"
	// ErrorTypeMinItems indicates array has fewer items than allowed
	ErrorTypeMinItems ValidationErrorType = "minItems"
	// ErrorTypeMaxItems indicates array has more items than allowed
	ErrorTypeMaxItems ValidationErrorType = "maxItems"
	// ErrorTypeMinimum indicates value is less than minimum
	ErrorTypeMinimum ValidationErrorType = "minimum"
	// ErrorTypeMaximum indicates value is greater than maximum
	ErrorTypeMaximum ValidationErrorType = "maximum"
	// ErrorTypeMinLength indicates string is shorter than minimum
	ErrorTypeMinLength ValidationErrorType = "minLength"
	// ErrorTypeMaxLength indicates string is longer than maximum
	ErrorTypeMaxLength ValidationErrorType = "maxLength"
	// ErrorTypeType indicates a type mismatch
	ErrorTypeType ValidationErrorType = "type"
	// ErrorTypeCEL indicates a CEL validation rule failure
	ErrorTypeCEL ValidationErrorType = "cel"
	// ErrorTypeAdditionalProperties indicates unknown fields were found
	ErrorTypeAdditionalProperties ValidationErrorType = "additionalProperties"
	// ErrorTypeUnknown indicates an unknown validation error type
	ErrorTypeUnknown ValidationErrorType = "unknown"
)

// ValidationError represents a single validation error
type ValidationError struct {
	// Path is the JSON path to the field that failed validation (e.g., ".spec.template.spec.domain")
	Path string
	// Message is the human-readable error message
	Message string
	// Type indicates what kind of validation failed
	Type ValidationErrorType
	// Value is the actual value that failed validation (optional)
	Value any
	// Rule is the validation rule that was violated (e.g., pattern regex, CEL expression)
	Rule string
}

// Error implements the error interface
func (e ValidationError) Error() string {
	if e.Rule != "" {
		return fmt.Sprintf("%s: %s (rule: %s)", e.Path, e.Message, e.Rule)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface for ValidationErrors
func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors returns true if there are any validation errors
func (errs ValidationErrors) HasErrors() bool {
	return len(errs) > 0
}

// ByType returns all errors of a specific type
func (errs ValidationErrors) ByType(t ValidationErrorType) ValidationErrors {
	var result ValidationErrors
	for _, e := range errs {
		if e.Type == t {
			result = append(result, e)
		}
	}
	return result
}

// ByPath returns all errors matching the given path (exact match)
func (errs ValidationErrors) ByPath(path string) ValidationErrors {
	var result ValidationErrors
	for _, e := range errs {
		if e.Path == path {
			result = append(result, e)
		}
	}
	return result
}

// ByPathPrefix returns all errors with paths starting with the given prefix
func (errs ValidationErrors) ByPathPrefix(prefix string) ValidationErrors {
	var result ValidationErrors
	for _, e := range errs {
		if strings.HasPrefix(e.Path, prefix) {
			result = append(result, e)
		}
	}
	return result
}
