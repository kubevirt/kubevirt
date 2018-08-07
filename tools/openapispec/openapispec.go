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
	"io/ioutil"
	"log"

	"github.com/emicklei/go-restful"
	openapispec "github.com/go-openapi/spec"
	"github.com/spf13/pflag"

	openapibuilder "k8s.io/kube-openapi/pkg/builder"
	openapicommon "k8s.io/kube-openapi/pkg/common"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	klog "kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-api"
)

func dumpOpenApiSpec(dumppath *string) {
	spec, err := openapibuilder.BuildOpenAPISpec(restful.RegisteredWebServices(), createOpenAPIConfig())
	if err != nil {
		log.Fatal(err)
	}
	data, err := json.MarshalIndent(spec, " ", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(*dumppath, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func createOpenAPIConfig() *openapicommon.Config {
	security := make([]map[string][]string, 1)
	security[0] = map[string][]string{"BearerToken": {}}
	return &openapicommon.Config{
		GetDefinitions: kubev1.GetOpenAPIDefinitions,
		ProtocolList:   []string{"https"},
		IgnorePrefixes: []string{"/swaggerapi"},
		Info: &openapispec.Info{
			InfoProps: openapispec.InfoProps{
				Title:       "KubeVirt API",
				Description: "This is KubeVirt API an add-on for Kubernetes.",
				Contact: &openapispec.ContactInfo{
					Name:  "kubevirt-dev",
					Email: "kubevirt-dev@googlegroups.com",
					URL:   "https://github.com/kubevirt/kubevirt",
				},
				License: &openapispec.License{
					Name: "Apache 2.0",
					URL:  "https://www.apache.org/licenses/LICENSE-2.0",
				},
			},
		},
		SecurityDefinitions: &openapispec.SecurityDefinitions{
			"BearerToken": &openapispec.SecurityScheme{
				SecuritySchemeProps: openapispec.SecuritySchemeProps{
					Type:        "apiKey",
					Name:        "authorization",
					In:          "header",
					Description: "Bearer Token authentication",
				},
			},
		},
		DefaultSecurity: security,
	}
}

func main() {
	dumpapispecpath := flag.String("dump-api-spec-path", "openapi.json", "Path to OpenApi dump.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	// client-go requires a config or a master to be set in order to configure a client
	pflag.Set("master", "http://127.0.0.1:4321")
	pflag.Parse()

	klog.InitializeLogging("openapispec")

	// arguments for NewVirtAPIApp have no influence on the generated spec
	app := virt_api.NewVirtApi()
	app.Compose()
	dumpOpenApiSpec(dumpapispecpath)
}
