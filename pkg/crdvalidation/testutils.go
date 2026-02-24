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

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// ValidateObject is a convenience function for validating an object against a CRD schema
// It creates a new validator, validates the object, and returns any errors
func ValidateObject(resourceName string, obj any) ValidationErrors {
	validator, err := NewValidator()
	if err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to create validator: %v", err),
			Type:    ErrorTypeUnknown,
		}}
	}
	return validator.Validate(resourceName, obj)
}

// ValidateUnstructuredObject is a convenience function for validating an unstructured object
func ValidateUnstructuredObject(resourceName string, obj map[string]any) ValidationErrors {
	validator, err := NewValidator()
	if err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to create validator: %v", err),
			Type:    ErrorTypeUnknown,
		}}
	}
	return validator.ValidateUnstructured(resourceName, obj)
}

// ExpectValid asserts that an object passes all CRD validations
func ExpectValid(resourceName string, obj any) {
	errs := ValidateObject(resourceName, obj)
	gomega.Expect(errs).To(gomega.BeEmpty(), "expected object to be valid but got errors: %v", errs)
}

// ExpectInvalid asserts that an object fails CRD validation
func ExpectInvalid(resourceName string, obj any) {
	errs := ValidateObject(resourceName, obj)
	gomega.Expect(errs).ToNot(gomega.BeEmpty(), "expected object to be invalid but it passed validation")
}

// ExpectInvalidWithError asserts that an object fails validation with a specific error type at a given path
func ExpectInvalidWithError(resourceName string, obj any, path string, errType ValidationErrorType) {
	errs := ValidateObject(resourceName, obj)
	gomega.Expect(errs).To(HaveValidationError(path, errType))
}

// HaveValidationError returns a matcher that checks for a validation error at a specific path with a specific type
func HaveValidationError(path string, errType ValidationErrorType) types.GomegaMatcher {
	return &validationErrorMatcher{
		path:    path,
		errType: errType,
	}
}

// HavePatternError returns a matcher that checks for a pattern validation error at a specific path
func HavePatternError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypePattern)
}

// HaveEnumError returns a matcher that checks for an enum validation error at a specific path
func HaveEnumError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypeEnum)
}

// HaveRequiredError returns a matcher that checks for a required field validation error at a specific path
func HaveRequiredError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypeRequired)
}

// HaveMinItemsError returns a matcher that checks for a minItems validation error at a specific path
func HaveMinItemsError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypeMinItems)
}

// HaveMaxItemsError returns a matcher that checks for a maxItems validation error at a specific path
func HaveMaxItemsError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypeMaxItems)
}

// HaveCELError returns a matcher that checks for a CEL validation error at a specific path
func HaveCELError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypeCEL)
}

// HaveTypeError returns a matcher that checks for a type validation error at a specific path
func HaveTypeError(path string) types.GomegaMatcher {
	return HaveValidationError(path, ErrorTypeType)
}

// validationErrorMatcher is a Gomega matcher for ValidationErrors
type validationErrorMatcher struct {
	path    string
	errType ValidationErrorType
}

func (m *validationErrorMatcher) Match(actual any) (success bool, err error) {
	errs, ok := actual.(ValidationErrors)
	if !ok {
		return false, fmt.Errorf("expected ValidationErrors, got %T", actual)
	}

	for _, e := range errs {
		if e.Path == m.path && e.Type == m.errType {
			return true, nil
		}
	}

	return false, nil
}

func (m *validationErrorMatcher) FailureMessage(actual any) string {
	errs, ok := actual.(ValidationErrors)
	if !ok {
		return fmt.Sprintf("expected ValidationErrors, got %T", actual)
	}

	return fmt.Sprintf("expected to find validation error at path %q with type %q\n\tActual errors: %v",
		m.path, m.errType, errs)
}

func (m *validationErrorMatcher) NegatedFailureMessage(actual any) string {
	return fmt.Sprintf("expected not to find validation error at path %q with type %q",
		m.path, m.errType)
}

// HaveValidationErrorCount returns a matcher that checks for a specific number of validation errors
func HaveValidationErrorCount(count int) types.GomegaMatcher {
	return &validationErrorCountMatcher{count: count}
}

type validationErrorCountMatcher struct {
	count int
}

func (m *validationErrorCountMatcher) Match(actual any) (success bool, err error) {
	errs, ok := actual.(ValidationErrors)
	if !ok {
		return false, fmt.Errorf("expected ValidationErrors, got %T", actual)
	}
	return len(errs) == m.count, nil
}

func (m *validationErrorCountMatcher) FailureMessage(actual any) string {
	errs, _ := actual.(ValidationErrors)
	return fmt.Sprintf("expected %d validation errors, got %d: %v", m.count, len(errs), errs)
}

func (m *validationErrorCountMatcher) NegatedFailureMessage(actual any) string {
	return fmt.Sprintf("expected not to have %d validation errors", m.count)
}

// ContainValidationErrorWithMessage returns a matcher that checks for an error containing a specific message
func ContainValidationErrorWithMessage(message string) types.GomegaMatcher {
	return &validationErrorMessageMatcher{message: message}
}

type validationErrorMessageMatcher struct {
	message string
}

func (m *validationErrorMessageMatcher) Match(actual any) (success bool, err error) {
	errs, ok := actual.(ValidationErrors)
	if !ok {
		return false, fmt.Errorf("expected ValidationErrors, got %T", actual)
	}

	for _, e := range errs {
		matched, _ := gomega.ContainSubstring(m.message).Match(e.Message)
		if matched {
			return true, nil
		}
	}

	return false, nil
}

func (m *validationErrorMessageMatcher) FailureMessage(actual any) string {
	errs, _ := actual.(ValidationErrors)
	return fmt.Sprintf("expected to find validation error containing message %q\n\tActual errors: %v",
		m.message, errs)
}

func (m *validationErrorMessageMatcher) NegatedFailureMessage(actual any) string {
	return fmt.Sprintf("expected not to find validation error containing message %q", m.message)
}
