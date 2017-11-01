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
	"net/http"

	"github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	kithttp "github.com/go-kit/kit/transport/http"
	openapispec "github.com/go-openapi/spec"
	"github.com/spf13/pflag"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	klog "kubevirt.io/kubevirt/pkg/log"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

type virtAPIApp struct {
	Service   *service.Service
	SwaggerUI string
}

func newVirtAPIApp(host *string, port *int, swaggerUI *string) *virtAPIApp {
	return &virtAPIApp{
		Service:   service.NewService("virt-api", host, port),
		SwaggerUI: *swaggerUI,
	}
}

func (app *virtAPIApp) Compose() {
	ctx := context.Background()
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachines"}
	migrationGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "migrations"}
	vmrsGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "virtualmachinereplicasets"}

	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmGVR, &v1.VirtualMachine{}, v1.VirtualMachineGroupVersionKind.Kind, &v1.VirtualMachineList{})
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, migrationGVR, &v1.Migration{}, v1.MigrationGroupVersionKind.Kind, &v1.MigrationList{})
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmrsGVR, &v1.VirtualMachineReplicaSet{}, v1.VMReplicaSetGroupVersionKind.Kind, &v1.VirtualMachineReplicaSetList{})
	if err != nil {
		log.Fatal(err)
	}

	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatal(err)
	}

	//  TODO, allow Encoder and Decoders per type and combine the endpoint logic
	spice := endpoints.MakeGoRestfulWrapper(endpoints.NewHandlerBuilder().Get().
		Endpoint(rest.NewSpiceEndpoint(virtCli.RestClient(), vmGVR)).Encoder(
		endpoints.NewMimeTypeAwareEncoder(endpoints.NewEncodeINIResponse(http.StatusOK),
			map[string]kithttp.EncodeResponseFunc{
				mime.MIME_INI:  endpoints.NewEncodeINIResponse(http.StatusOK),
				mime.MIME_JSON: endpoints.NewEncodeJsonResponse(http.StatusOK),
				mime.MIME_YAML: endpoints.NewEncodeYamlResponse(http.StatusOK),
			})).Build(ctx))

	ws.Route(ws.GET(rest.ResourcePath(vmGVR)+rest.SubResourcePath("spice")).
		To(spice).Produces(mime.MIME_INI, mime.MIME_JSON, mime.MIME_YAML).
		Param(rest.NamespaceParam(ws)).Param(rest.NameParam(ws)).
		Operation("spice").
		Doc("Returns a remote-viewer configuration file. Run `man 1 remote-viewer` to learn more about the configuration format."))

	ws.Route(ws.GET(rest.ResourcePath(vmGVR) + rest.SubResourcePath("console")).
		To(rest.NewConsoleResource(virtCli, virtCli.CoreV1()).Console).
		Param(restful.QueryParameter("console", "Name of the serial console to connect to")).
		Param(rest.NamespaceParam(ws)).Param(rest.NameParam(ws)).
		Operation("console").
		Doc("Open a websocket connection to a serial console on the specified VM."))

	restful.Add(ws)

	ws.Route(ws.GET("/healthz").To(healthz.KubeConnectionHealthzFunc).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON).Doc("Health endpoint"))
	ws, err = rest.ResourceProxyAutodiscovery(ctx, vmGVR)
	if err != nil {
		log.Fatal(err)
	}

	restful.Add(ws)

	restful.Filter(filter.RequestLoggingFilter())
	restful.Filter(restful.OPTIONSFilter())
}

func (app *virtAPIApp) ConfigureOpenAPIService() {
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(createOpenAPIConfig()))
	http.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir(app.SwaggerUI))))
}

func createOpenAPIConfig() restfulspec.Config {
	return restfulspec.Config{
		WebServices:    restful.RegisteredWebServices(),
		WebServicesURL: "http://localhost:8183",
		APIPath:        "/swaggerapi",
		PostBuildSwaggerObjectHandler: addInfoToSwaggerObject,
	}
}

func addInfoToSwaggerObject(swo *openapispec.Swagger) {
	swo.Info = &openapispec.Info{
		InfoProps: openapispec.InfoProps{
			Title:       "KubeVirt API, ",
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
	}
}

func (app *virtAPIApp) Run() {
	log.Fatal(http.ListenAndServe(app.Service.Address(), nil))
}

func dumpOpenApiSpec(dumppath *string) {
	openapispec := restfulspec.BuildSwagger(createOpenAPIConfig())
	data, err := json.MarshalIndent(openapispec, " ", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(*dumppath, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	klog.InitializeLogging("virt-api")
	swaggerui := flag.String("swagger-ui", "third_party/swagger-ui", "swagger-ui location")
	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8183, "Port to listen on")
	dumpapispec := flag.Bool("dump-api-spec", false, "Dump OpenApi spec, and exit immediately.")
	dumpapispecpath := flag.String("dump-api-spec-path", "openapi.json", "Path to OpenApi dump.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	app := newVirtAPIApp(host, port, swaggerui)
	app.Compose()
	if *dumpapispec == true {
		dumpOpenApiSpec(dumpapispecpath)
	} else {
		app.ConfigureOpenAPIService()
		app.Run()
	}
}
