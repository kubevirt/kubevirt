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

package kubecli

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

type KubevirtClient interface {
	VM(namespace string) VMInterface
}

type kubevirt struct {
	restClient *rest.RESTClient
}

func (k *kubevirt) VM(namespace string) VMInterface {
	return &vms{k.restClient, namespace}
}

type VMInterface interface {
	Get(name string, options k8smetav1.GetOptions) (*v1.VM, error)
	List(opts k8smetav1.ListOptions) (*v1.VMList, error)
	Create(*v1.VM) (*v1.VM, error)
	Update(*v1.VM) (*v1.VM, error)
	Delete(name string, options *k8smetav1.DeleteOptions) error
}

type vms struct {
	restClient *rest.RESTClient
	namespace  string
}

func (v *vms) Get(name string, options k8smetav1.GetOptions) (vm *v1.VM, err error) {
	vm = &v1.VM{}
	err = v.restClient.Get().
		Resource("vms").
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, api.ParameterCodec).
		Do().
		Into(vm)
	vm.SetGroupVersionKind(v1.GroupVersionKind)
	return
}

func (v *vms) List(options k8smetav1.ListOptions) (vmList *v1.VMList, err error) {
	vmList = &v1.VMList{}
	err = v.restClient.Get().
		Resource("vms").
		Namespace(v.namespace).
		VersionedParams(&options, api.ParameterCodec).
		Do().
		Into(vmList)
	for _, vm := range vmList.Items {
		vm.SetGroupVersionKind(v1.GroupVersionKind)
	}

	return
}

func (v *vms) Create(vm *v1.VM) (result *v1.VM, err error) {
	result = &v1.VM{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource("vms").
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.GroupVersionKind)
	return
}

func (v *vms) Update(vm *v1.VM) (result *v1.VM, err error) {
	result = &v1.VM{}
	err = v.restClient.Put().
		Namespace(v.namespace).
		Resource("vms").
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.GroupVersionKind)
	return
}

func (v *vms) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource("vms").
		Name(name).
		Body(options).
		Do().
		Error()
}
