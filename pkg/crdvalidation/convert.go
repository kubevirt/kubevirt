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

	openapi_spec "github.com/go-openapi/spec"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ConvertToGoOpenAPISchema converts a k8s JSONSchemaProps to a go-openapi spec.Schema
// This enables reuse of the go-openapi/validate library for schema validation.
// The conversion is done via JSON marshal/unmarshal since both types have similar JSON structure.
func ConvertToGoOpenAPISchema(props *extv1.JSONSchemaProps) (*openapi_spec.Schema, error) {
	if props == nil {
		return nil, fmt.Errorf("nil JSONSchemaProps")
	}

	// Convert JSONSchemaProps to JSON
	jsonBytes, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSONSchemaProps: %w", err)
	}

	// Unmarshal into go-openapi Schema
	var schema openapi_spec.Schema
	if err := json.Unmarshal(jsonBytes, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to go-openapi Schema: %w", err)
	}

	return &schema, nil
}

// ConvertToGoOpenAPISchemaWithDefinitions converts a JSONSchemaProps to a go-openapi Schema
// and also handles any $ref definitions by inlining them.
func ConvertToGoOpenAPISchemaWithDefinitions(props *extv1.JSONSchemaProps) (*openapi_spec.Schema, error) {
	schema, err := ConvertToGoOpenAPISchema(props)
	if err != nil {
		return nil, err
	}

	// Expand any $ref references in the schema
	// This is necessary because CRD schemas may contain internal references
	if err := openapi_spec.ExpandSchema(schema, schema, nil); err != nil {
		return nil, fmt.Errorf("failed to expand schema: %w", err)
	}

	return schema, nil
}

// schemaCache caches converted schemas to avoid repeated conversion
type schemaCache struct {
	cache map[string]*openapi_spec.Schema
}

func newSchemaCache() *schemaCache {
	return &schemaCache{
		cache: make(map[string]*openapi_spec.Schema),
	}
}

func (c *schemaCache) get(key string) (*openapi_spec.Schema, bool) {
	s, ok := c.cache[key]
	return s, ok
}

func (c *schemaCache) set(key string, schema *openapi_spec.Schema) {
	c.cache[key] = schema
}
