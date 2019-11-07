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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package kubecli

import (
	"fmt"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/client-go/api/v1"
)

const vmSubresourceURL = "/apis/subresources.kubevirt.io/%s/namespaces/%s/virtualmachines/%s/%s"

func (k *kubevirt) VirtualMachine(namespace string) VirtualMachineInterface {
	return &vm{
		restClient: k.restClient,
		namespace:  namespace,
		resource:   "virtualmachines",
	}
}

type vm struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new VirtualMachine in the cluster to specified namespace
func (o *vm) Create(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	newVm := &v1.VirtualMachine{}
	err := o.restClient.Post().
		Resource(o.resource).
		Namespace(o.namespace).
		Body(vm).
		Do().
		Into(newVm)

	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Get the Virtual machine from the cluster by its name and namespace
func (o *vm) Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachine, error) {
	newVm := &v1.VirtualMachine{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(newVm)

	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Update the VirtualMachine instance in the cluster in given namespace
func (o *vm) Update(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	updatedVm := &v1.VirtualMachine{}
	err := o.restClient.Put().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(vm.Name).
		Body(vm).
		Do().
		Into(updatedVm)

	updatedVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return updatedVm, err
}

// Delete the defined VirtualMachine in the cluster in defined namespace
func (o *vm) Delete(name string, options *k8smetav1.DeleteOptions) error {
	err := o.restClient.Delete().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		Body(options).
		Do().
		Error()

	return err
}

// List all VirtualMachines in given namespace
func (o *vm) List(options *k8smetav1.ListOptions) (*v1.VirtualMachineList, error) {
	newVmList := &v1.VirtualMachineList{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(newVmList)

	for _, vm := range newVmList.Items {
		vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	}

	return newVmList, err
}

func (v *vm) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return result, err
}

func (v *vm) Restart(name string) error {
	uri := fmt.Sprintf(vmSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "restart")
	return v.restClient.Put().RequestURI(uri).Do().Error()
}

func (v *vm) Start(name string) error {
	uri := fmt.Sprintf(vmSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "start")
	return v.restClient.Put().RequestURI(uri).Do().Error()
}

func (v *vm) Stop(name string) error {
	uri := fmt.Sprintf(vmSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "stop")
	return v.restClient.Put().RequestURI(uri).Do().Error()
}

func (v *vm) Migrate(name string) error {
	uri := fmt.Sprintf(vmSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "migrate")
	return v.restClient.Put().RequestURI(uri).Do().Error()
}
