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
	"strings"

	"github.com/go-kit/kit/endpoint"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
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
	for _, d := range vm.Spec.Domain.Devices.Graphics {
		if strings.ToLower(d.Type) == "spice" {
			port := d.Port
			podList, err := coreCli.CoreV1().Pods(api.NamespaceDefault).List(unfinishedVMPodSelector(vm))
			if err != nil {
				return nil, middleware.NewInternalServerError(err)
			}

			// The pod could just have failed now
			if len(podList.Items) == 0 {
				// TODO is that the right return code?
				return nil, middleware.NewResourceNotFoundError("VM is not running")
			}

			pod := podList.Items[0]
			ip := pod.Status.PodIP

			spice := v1.NewSpice(vm.GetObjectMeta().GetName())
			spice.Info = v1.SpiceInfo{
				Type: "spice",
				Host: ip,
				Port: port,
			}
			if spiceProxy != "" {
				spice.Info.Proxy = fmt.Sprintf("http://%s", spiceProxy)
			}
			return spice, nil
		}
	}

	return nil, middleware.NewResourceNotFoundError("No spice device attached to the VM found.")
}

// TODO for now just copied from VMService
func unfinishedVMPodSelector(vm *v1.VM) metav1.ListOptions {
	fieldSelector := fields.ParseSelectorOrDie(
		"status.phase!=" + string(kubev1.PodFailed) +
			",status.phase!=" + string(kubev1.PodSucceeded))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.DomainLabel+" in (%s)", vm.GetObjectMeta().GetName()))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}
