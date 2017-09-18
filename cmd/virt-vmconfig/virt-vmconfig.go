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
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/rest/filter"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
	vmconfigv1 "kubevirt.io/kubevirt/pkg/virt-vmconfig/api/v1"
	vmconfigrest "kubevirt.io/kubevirt/pkg/virt-vmconfig/rest"
)

func test(request *restful.Request, response *restful.Response) {
	response.Write([]byte("testing"))
}

func main() {
	swaggerui := flag.String("swagger-ui", "third_party/swagger-ui", "swagger-ui location")
	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8187, "Port to listen on")

	ctx := context.Background()

	vmcGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "vmconfigs"}

	// Create basic REST paths.
	ws, err := rest.GroupVersionProxyBase(ctx, v1.GroupVersion)
	if err != nil {
		log.Fatal(err)
	}

	ws, err = rest.GenericResourceProxy(ws, ctx, vmcGVR, &vmconfigv1.VMConfig{}, vmconfigv1.VMConfigGroupVersionKind.Kind, &vmconfigv1.VMConfigList{})
	if err != nil {
		log.Fatal(err)
	}

	// TODO: although the service responds to rest API calls directly, we still need to use autodiscovery to communicate via reasonable format (aka JSON).
	// That can be left for later when apiserver aggregation is in.

	ws.Route(ws.GET(rest.ResourcePath(vmcGVR) + rest.SubResourcePath("start")).
		To(vmconfigrest.StartFunc).
		Param(rest.NamespaceParam(ws)).
		Param(rest.NameParam(ws)).
		Doc("Creates a VM from the given VMConfig."))

	ws.Route(ws.GET(rest.ResourcePath(vmcGVR) + rest.SubResourcePath("stop")).
		To(vmconfigrest.StopFunc).
		Param(rest.NamespaceParam(ws)).
		Param(rest.NameParam(ws)).
		Doc("Stops a VM from given VMConfig."))

	log.Printf("Registered routes: %v", ws.Routes())

	restful.Add(ws)

	// We want to see the requests (at least for development) & make sure the service plays nicely with virt-api.
	restful.Filter(filter.RequestLoggingFilter())
	restful.Filter(restful.OPTIONSFilter())

	config := swagger.Config{
		WebServices:     restful.RegisteredWebServices(), // you control what services are visible
		WebServicesUrl:  "http://localhost:8187",
		ApiPath:         "/swaggerapi",
		SwaggerPath:     "/swagger-ui/",
		SwaggerFilePath: *swaggerui,
	}
	swagger.InstallSwaggerService(config)

	log.Fatal(http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil))
}
