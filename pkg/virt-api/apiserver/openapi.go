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

package apiserver

import (
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	apiopenapi "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/spec3"
	"k8s.io/kube-openapi/pkg/validation/spec"

	kubevirtopenapi "kubevirt.io/client-go/api"
)

var info = &spec.Info{
	InfoProps: spec.InfoProps{
		Title:       "KubeVirt virt-api (aggregated)",
		Description: "KubeVirt's subresources.kubevirt.io API group, served via k8s.io/apiserver.",
		Contact: &spec.ContactInfo{
			Name:  "kubevirt-dev",
			Email: "kubevirt-dev@googlegroups.com",
			URL:   "https://github.com/kubevirt/kubevirt",
		},
		License: &spec.License{
			Name: "Apache 2.0",
			URL:  "https://www.apache.org/licenses/LICENSE-2.0",
		},
	},
}

func NewOpenAPIConfig(scheme *runtime.Scheme) *common.Config {
	_ = scheme
	return &common.Config{
		ProtocolList: []string{"https"},
		Info:         info,
		DefaultResponse: &spec.Response{
			ResponseProps: spec.ResponseProps{
				Description: "Default Response.",
			},
		},
		// GetOperationIDAndTags derives the OpenAPI operation ID from the
		// serving group/version in the request path, so the same subresource
		// served under both subresources.kubevirt.io/v1 and /v1alpha3 gets
		// distinct operation IDs. Without it would see duplicate IDs and crash
		GetOperationIDAndTags: apiopenapi.GetOperationIDAndTags,
		GetDefinitions:        getDefinitions,
		GetDefinitionName:     getDefinitionName,
		CommonResponses: map[int]spec.Response{
			http.StatusUnauthorized: unauthorizedResponse(),
		},
	}
}

// NewOpenAPIV3Config mirrors NewOpenAPIConfig for OpenAPI v3.
func NewOpenAPIV3Config(scheme *runtime.Scheme) *common.OpenAPIV3Config {
	_ = scheme
	return &common.OpenAPIV3Config{
		Info: info,
		DefaultResponse: &spec3.Response{
			ResponseProps: spec3.ResponseProps{
				Description: "Default Response.",
			},
		},
		GetOperationIDAndTags: apiopenapi.GetOperationIDAndTags,
		GetDefinitions:        getDefinitions,
		GetDefinitionName:     getDefinitionName,
		CommonResponses: map[int]*spec3.Response{
			http.StatusUnauthorized: {
				ResponseProps: spec3.ResponseProps{Description: http.StatusText(http.StatusUnauthorized)},
			},
		},
	}
}

// unauthorizedResponse is the shared 401 response documented on every operation.
func unauthorizedResponse() spec.Response {
	return spec.Response{
		ResponseProps: spec.ResponseProps{Description: http.StatusText(http.StatusUnauthorized)},
	}
}

// stringResponse builds a response whose body is a plain string, matching the
// shape the pre-migration go-restful routes documented for their status codes.
func stringResponse(code int) spec.Response {
	return spec.Response{
		ResponseProps: spec.ResponseProps{
			Description: http.StatusText(code),
			Schema:      &spec.Schema{SchemaProps: spec.SchemaProps{Type: spec.StringOrArray{"string"}}},
		},
	}
}

func getDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	defs := kubevirtopenapi.GetOpenAPIDefinitions(ref)
	defs["k8s.io/apimachinery/pkg/version.Info"] = schemaVersionInfo()
	return defs
}

func getDefinitionName(name string) (string, spec.Extensions) {
	if strings.Contains(name, "kubevirt.io") {
		return name[strings.LastIndex(name, "/")+1:], nil
	}
	return strings.ReplaceAll(name, "/", "."), nil
}

func schemaVersionInfo() common.OpenAPIDefinition {
	stringProp := func(desc string) spec.Schema {
		return spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: desc,
				Type:        []string{"string"},
				Format:      "",
			},
		}
	}

	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Info contains versioning information for an API server, as exposed on /version.",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"major":                 stringProp("Major is the major version of the binary version."),
					"minor":                 stringProp("Minor is the minor version of the binary version."),
					"emulationMajor":        stringProp("EmulationMajor is the major version of the emulation version."),
					"emulationMinor":        stringProp("EmulationMinor is the minor version of the emulation version."),
					"minCompatibilityMajor": stringProp("MinCompatibilityMajor is the major version of the minimum compatibility version."),
					"minCompatibilityMinor": stringProp("MinCompatibilityMinor is the minor version of the minimum compatibility version."),
					"gitVersion":            stringProp("GitVersion is the git version of the binary."),
					"gitCommit":             stringProp("GitCommit is the git commit of the binary."),
					"gitTreeState":          stringProp("GitTreeState is the state of the git tree the binary was built from."),
					"buildDate":             stringProp("BuildDate is the date the binary was built."),
					"goVersion":             stringProp("GoVersion is the Go version of the binary."),
					"compiler":              stringProp("Compiler is the Go compiler used to build the binary."),
					"platform":              stringProp("Platform is the platform the binary was built for."),
				},
				Required: []string{
					"major",
					"minor",
					"gitVersion",
					"gitCommit",
					"gitTreeState",
					"buildDate",
					"goVersion",
					"compiler",
					"platform",
				},
			},
		},
	}
}
