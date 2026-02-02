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

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

// schemaWrapper wraps the YAML format of CRDsValidation entries
type schemaWrapper struct {
	OpenAPIV3Schema *extv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"`
}

// LoadSchemasFromCRDsValidation loads all schemas from the CRDsValidation map
func LoadSchemasFromCRDsValidation() (map[string]*extv1.JSONSchemaProps, error) {
	schemas := make(map[string]*extv1.JSONSchemaProps)

	for resourceName, yamlStr := range components.CRDsValidation {
		schema, err := parseSchemaYAML(yamlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schema for %s: %w", resourceName, err)
		}
		schemas[strings.ToLower(resourceName)] = schema
	}

	return schemas, nil
}

// parseSchemaYAML parses a YAML string from CRDsValidation into JSONSchemaProps
func parseSchemaYAML(yamlStr string) (*extv1.JSONSchemaProps, error) {
	var wrapper schemaWrapper
	if err := yaml.Unmarshal([]byte(yamlStr), &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if wrapper.OpenAPIV3Schema == nil {
		return nil, fmt.Errorf("no openAPIV3Schema found in YAML")
	}

	return wrapper.OpenAPIV3Schema, nil
}

// GetResourceNames returns all available resource names from CRDsValidation
func GetResourceNames() []string {
	names := make([]string, 0, len(components.CRDsValidation))
	for name := range components.CRDsValidation {
		names = append(names, strings.ToLower(name))
	}
	return names
}
