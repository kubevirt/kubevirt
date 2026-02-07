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
	"encoding/json"
	"fmt"
	"strings"

	openapi_spec "github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	openapi_validate "github.com/go-openapi/validate"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Validator validates objects against CRD OpenAPI v3 schemas
type Validator struct {
	// schemas contains JSONSchemaProps for each resource type, keyed by lowercase resource name
	schemas map[string]*extv1.JSONSchemaProps
	// schemaCache caches converted go-openapi schemas
	schemaCache *schemaCache
	// celValidator handles CEL validation rules
	celValidator *CELValidator
}

// NewValidator creates a new Validator by loading schemas from CRDsValidation
func NewValidator() (*Validator, error) {
	schemas, err := LoadSchemasFromCRDsValidation()
	if err != nil {
		return nil, fmt.Errorf("failed to load schemas: %w", err)
	}

	return &Validator{
		schemas:      schemas,
		schemaCache:  newSchemaCache(),
		celValidator: NewCELValidator(),
	}, nil
}

// NewValidatorWithSchemas creates a Validator with custom schemas (useful for testing)
func NewValidatorWithSchemas(schemas map[string]*extv1.JSONSchemaProps) *Validator {
	return &Validator{
		schemas:      schemas,
		schemaCache:  newSchemaCache(),
		celValidator: NewCELValidator(),
	}
}

// Validate validates a typed object against the CRD schema for the given resource
// The object is first converted to an unstructured map via JSON marshaling
func (v *Validator) Validate(resourceName string, obj any) ValidationErrors {
	// Convert typed object to unstructured map
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to marshal object: %v", err),
			Type:    ErrorTypeUnknown,
		}}
	}

	var unstructured map[string]any
	if err := json.Unmarshal(jsonBytes, &unstructured); err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to unmarshal object: %v", err),
			Type:    ErrorTypeUnknown,
		}}
	}

	return v.ValidateUnstructured(resourceName, unstructured)
}

// ValidateUnstructured validates an unstructured object against the CRD schema
func (v *Validator) ValidateUnstructured(resourceName string, obj map[string]any) ValidationErrors {
	var errs ValidationErrors

	// Standard JSON Schema validation
	errs = append(errs, v.validateSchema(resourceName, obj)...)

	// CEL validation
	errs = append(errs, v.validateCEL(resourceName, obj, nil)...)

	return errs
}

// ValidateUpdate validates an update operation, including transition rules that use oldSelf
func (v *Validator) ValidateUpdate(resourceName string, newObj, oldObj any) ValidationErrors {
	// Convert typed objects to unstructured maps
	newUnstructured, err := toUnstructured(newObj)
	if err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to convert new object: %v", err),
			Type:    ErrorTypeUnknown,
		}}
	}

	oldUnstructured, err := toUnstructured(oldObj)
	if err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to convert old object: %v", err),
			Type:    ErrorTypeUnknown,
		}}
	}

	return v.ValidateUpdateUnstructured(resourceName, newUnstructured, oldUnstructured)
}

// ValidateUpdateUnstructured validates an update with unstructured objects
func (v *Validator) ValidateUpdateUnstructured(resourceName string, newObj, oldObj map[string]any) ValidationErrors {
	var errs ValidationErrors

	// Standard JSON Schema validation (only on new object)
	errs = append(errs, v.validateSchema(resourceName, newObj)...)

	// CEL validation (including transition rules)
	errs = append(errs, v.validateCEL(resourceName, newObj, oldObj)...)

	return errs
}

// validateSchema performs JSON Schema validation using go-openapi/validate
func (v *Validator) validateSchema(resourceName string, obj map[string]any) ValidationErrors {
	schema, err := v.getGoOpenAPISchema(resourceName)
	if err != nil {
		return ValidationErrors{{
			Path:    "",
			Message: fmt.Sprintf("failed to get schema for %s: %v", resourceName, err),
			Type:    ErrorTypeUnknown,
		}}
	}

	validator := openapi_validate.NewSchemaValidator(schema, nil, "", strfmt.Default)
	result := validator.Validate(obj)

	return mapGoOpenAPIErrors(result.Errors)
}

// validateCEL performs CEL validation on the object
func (v *Validator) validateCEL(resourceName string, obj, oldObj map[string]any) ValidationErrors {
	schema, ok := v.schemas[strings.ToLower(resourceName)]
	if !ok {
		return nil
	}

	return v.celValidator.validateCELAtPath(schema, obj, oldObj, "")
}

// getGoOpenAPISchema returns the go-openapi schema for a resource, caching the result
func (v *Validator) getGoOpenAPISchema(resourceName string) (*openapi_spec.Schema, error) {
	key := strings.ToLower(resourceName)

	// Check cache
	if cached, ok := v.schemaCache.get(key); ok {
		return cached, nil
	}

	// Get the k8s schema
	k8sSchema, ok := v.schemas[key]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", resourceName)
	}

	// Convert to go-openapi schema
	schema, err := ConvertToGoOpenAPISchemaWithDefinitions(k8sSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema: %w", err)
	}

	// Cache and return
	v.schemaCache.set(key, schema)
	return schema, nil
}

// GetSchema returns the JSONSchemaProps for a resource
func (v *Validator) GetSchema(resourceName string) (*extv1.JSONSchemaProps, bool) {
	schema, ok := v.schemas[strings.ToLower(resourceName)]
	return schema, ok
}

// GetResourceNames returns all available resource names
func (v *Validator) GetResourceNames() []string {
	names := make([]string, 0, len(v.schemas))
	for name := range v.schemas {
		names = append(names, name)
	}
	return names
}

// toUnstructured converts a typed object to an unstructured map
func toUnstructured(obj any) (map[string]any, error) {
	if obj == nil {
		return nil, nil
	}

	// If already a map, return it
	if m, ok := obj.(map[string]any); ok {
		return m, nil
	}

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// mapGoOpenAPIErrors converts go-openapi validation errors to ValidationErrors
func mapGoOpenAPIErrors(errs []error) ValidationErrors {
	var result ValidationErrors

	for _, err := range errs {
		verr := mapSingleError(err)
		result = append(result, verr)
	}

	return result
}

// mapSingleError maps a single go-openapi error to a ValidationError
func mapSingleError(err error) ValidationError {
	errStr := err.Error()

	// Determine error type based on error message patterns
	errType := determineErrorType(errStr)

	// Extract path from error message
	path := extractPath(errStr)

	return ValidationError{
		Path:    path,
		Message: errStr,
		Type:    errType,
	}
}

// determineErrorType determines the ValidationErrorType from an error message
//
//nolint:gocyclo
func determineErrorType(errStr string) ValidationErrorType {
	switch {
	case strings.Contains(errStr, "pattern"):
		return ErrorTypePattern
	case strings.Contains(errStr, "should be one of") || strings.Contains(errStr, "must be one of"):
		return ErrorTypeEnum
	case strings.Contains(errStr, "is required"):
		return ErrorTypeRequired
	case strings.Contains(errStr, "minItems") || strings.Contains(errStr, "should have at least"):
		return ErrorTypeMinItems
	case strings.Contains(errStr, "maxItems") || strings.Contains(errStr, "should have at most"):
		return ErrorTypeMaxItems
	case strings.Contains(errStr, "minimum") || strings.Contains(errStr, "less than minimum") ||
		strings.Contains(errStr, "should be greater than or equal to"):
		return ErrorTypeMinimum
	case strings.Contains(errStr, "maximum") || strings.Contains(errStr, "greater than maximum") ||
		strings.Contains(errStr, "should be less than or equal to"):
		return ErrorTypeMaximum
	case strings.Contains(errStr, "minLength") || strings.Contains(errStr, "should be at least"):
		return ErrorTypeMinLength
	case strings.Contains(errStr, "maxLength") || strings.Contains(errStr, "should be at most"):
		return ErrorTypeMaxLength
	case strings.Contains(errStr, "Invalid type") || strings.Contains(errStr, "expected type") || strings.Contains(errStr, "must be of type"):
		return ErrorTypeType
	case strings.Contains(errStr, "additional properties"):
		return ErrorTypeAdditionalProperties
	default:
		return ErrorTypeUnknown
	}
}

// extractPath extracts the path from a go-openapi error message
func extractPath(errStr string) string {
	// go-openapi errors typically start with the path in the format:
	// "path.to.field: error message" or "path.to.field in body: error message"
	path, _, _ := strings.Cut(errStr, ":")
	path = strings.TrimSpace(path)
	// Remove " in body" suffix if present
	path = strings.TrimSuffix(path, " in body")
	// Ensure path starts with a dot if it contains nested paths
	if path != "" && !strings.HasPrefix(path, ".") && strings.Contains(path, ".") {
		path = "." + path
	}
	return path
}
