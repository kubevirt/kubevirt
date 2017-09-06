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
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/spf13/pflag"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/healthz"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	mime "kubevirt.io/kubevirt/pkg/rest"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

func main() {

	logging.InitializeLogging("virt-api")
	swaggerui := flag.String("swagger-ui", "third_party/swagger-ui", "swagger-ui location")
	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8183, "Port to listen on")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	ctx := context.Background()
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "vms"}
	migrationGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "migrations"}

	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmGVR, &v1.VM{}, v1.VMGroupVersionKind.Kind, &v1.VMList{})
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, migrationGVR, &v1.Migration{}, v1.MigrationGroupVersionKind.Kind, &v1.MigrationList{})
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
		Doc("Returns a remote-viewer configuration file. Run `man 1 remote-viewer` to learn more about the configuration format."))

	ws.Route(ws.GET(rest.ResourcePath(vmGVR) + rest.SubResourcePath("console")).
		To(rest.NewConsoleResource(virtCli, virtCli.CoreV1()).Console).
		Param(restful.QueryParameter("console", "Name of the serial console to connect to")).
		Param(rest.NamespaceParam(ws)).Param(rest.NameParam(ws)).
		Doc("Open a websocket connection to a serial console on the specified VM."))

	restful.Add(ws)

	ws.Route(ws.GET("/healthz").To(healthz.KubeConnectionHealthzFunc).Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON).Doc("Health endpoint"))
	ws, err = rest.ResourceProxyAutodiscovery(ctx, vmGVR)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Registered routes: %v", ws.Routes())

	restful.Add(ws)

	restful.Filter(filter.RequestLoggingFilter())
	restful.Filter(restful.OPTIONSFilter())

	config := swagger.Config{
		WebServices:     restful.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl:  "http://localhost:8183",
		ApiPath:         "/swaggerapi",
		SwaggerPath:     "/swagger-ui/",
		SwaggerFilePath: *swaggerui,
	}
	swagger.InstallSwaggerService(config)

	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
}
