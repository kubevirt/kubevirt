/*
 * This file is part of the kubevirt project
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

package rest

import (
	"flag"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest/endpoints"
)

var spiceProxy string

func init() {
	// TODO should be reloadable, use configmaps and update on every access? Watch a config file and reload?
	flag.StringVar(&spiceProxy, "spice-proxy", "", "Spice proxy to use when spice access is requested")
}

func NewSpiceEndpoint(cli *rest.RESTClient, coreCli *kubernetes.Clientset, gvr schema.GroupVersionResource) endpoint.Endpoint {
	return func(ctx context.Context, payload interface{}) (interface{}, error) {
		metadata := payload.(*endpoints.Metadata)
		obj, err := cli.Get().Namespace(metadata.Namespace).Resource(gvr.Resource).Name(metadata.Name).Do().Get()
		if err != nil {
			return nil, middleware.NewInternalServerError(err)
		}

		vm := obj.(*v1.VM)
		spice, err := spiceFromVM(vm, coreCli)
		if err != nil {
			return nil, err

		}

		return spice, nil
	}
}

func spiceFromVM(vm *v1.VM, coreCli *kubernetes.Clientset) (*v1.Spice, error) {

	if vm.Status.Phase != v1.Running {
		return nil, middleware.NewResourceNotFoundError("VM is not running")
	}

	// TODO allow specifying the spice device. For now select the first one.
	for _, d := range vm.Status.Graphics {
		if d.Type == "spice" {
			spice := v1.NewSpice(vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
			spice.Info = v1.SpiceInfo{
				Type: "spice",
				Host: d.Host,
				Port: d.Port,
			}
			if spiceProxy != "" {
				spice.Info.Proxy = fmt.Sprintf("http://%s", spiceProxy)
			}
			return spice, nil
		}
	}

	return nil, middleware.NewResourceNotFoundError("No spice device attached to the VM found.")
}
