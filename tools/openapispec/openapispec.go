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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/emicklei/go-restful/v3"
	"github.com/spf13/pflag"
	"k8s.io/kube-openapi/pkg/validation/spec"

	klog "kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/openapi"
	"kubevirt.io/kubevirt/pkg/virt-api/apiserver"
	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

func dumpOpenApiSpec(dumppath *string, openapispec *spec.Swagger) {
	data, err := json.MarshalIndent(openapispec, " ", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(*dumppath, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

// addSubresourcePaths copies the paths of the aggregated API server into the
// spec built from the go-restful definitions.
func addSubresourcePaths(base, subresources *spec.Swagger) *spec.Swagger {
	if base.Paths == nil {
		base.Paths = &spec.Paths{}
	}
	if base.Paths.Paths == nil {
		base.Paths.Paths = map[string]spec.PathItem{}
	}
	if subresources.Paths != nil {
		for path, item := range subresources.Paths.Paths {
			base.Paths.Paths[path] = item
		}
	}

	if base.Definitions == nil {
		base.Definitions = spec.Definitions{}
	}
	for name, definition := range subresources.Definitions {
		if _, exists := base.Definitions[name]; !exists {
			base.Definitions[name] = definition
		}
	}

	if base.Parameters == nil {
		base.Parameters = map[string]spec.Parameter{}
	}
	for name, parameter := range subresources.Parameters {
		if _, exists := base.Parameters[name]; !exists {
			base.Parameters[name] = parameter
		}
	}
	return base
}

func main() {
	dumpapispecpath := flag.String("dump-api-spec-path", "openapi.json", "Path to OpenApi dump.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	// client-go requires a config or a master to be set in order to configure a client
	pflag.Set("master", "http://127.0.0.1:4321")
	pflag.Parse()

	klog.InitializeLogging("openapispec")

	crdSpec := openapi.LoadOpenAPISpec(append(definitions.ComposeAPIDefinitions(), restful.RegisteredWebServices()...))

	subresourceSpec, err := apiserver.BuildSubresourceOpenAPISpec()
	if err != nil {
		log.Fatal(err)
	}

	dumpOpenApiSpec(dumpapispecpath, addSubresourcePaths(crdSpec, subresourceSpec))
}
