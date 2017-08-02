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

package kubecli

import (
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func (k *kubevirt) VM(namespace string) VMInterface {
	return &vms{k.restClient, namespace, "vms"}
}

type vms struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *vms) Get(name string, options k8smetav1.GetOptions) (vm *v1.VM, err error) {
	vm = &v1.VM{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vm)
	vm.SetGroupVersionKind(v1.VMGroupVersionKind)
	return
}

func (v *vms) List(options k8smetav1.ListOptions) (vmList *v1.VMList, err error) {
	vmList = &v1.VMList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vmList)
	for _, vm := range vmList.Items {
		vm.SetGroupVersionKind(v1.VMGroupVersionKind)
	}

	return
}

func (v *vms) Create(vm *v1.VM) (result *v1.VM, err error) {
	result = &v1.VM{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VMGroupVersionKind)
	return
}

func (v *vms) Update(vm *v1.VM) (result *v1.VM, err error) {
	result = &v1.VM{}
	err = v.restClient.Put().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VMGroupVersionKind)
	return
}

func (v *vms) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}
