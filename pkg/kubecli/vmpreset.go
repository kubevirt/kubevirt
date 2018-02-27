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

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func (k *kubevirt) VMPreset(namespace string) VMPresetInterface {
	return &vmPresets{k.restClient, namespace, "virtualmachinepresets"}
}

type vmPresets struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *vmPresets) Get(name string, options k8smetav1.GetOptions) (vm *v1.VirtualMachinePreset, err error) {
	vm = &v1.VirtualMachinePreset{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vm)
	vm.SetGroupVersionKind(v1.VirtualMachinePresetGroupVersionKind)
	return
}

func (v *vmPresets) List(options k8smetav1.ListOptions) (vmList *v1.VirtualMachinePresetList, err error) {
	vmList = &v1.VirtualMachinePresetList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vmList)
	for _, vm := range vmList.Items {
		vm.SetGroupVersionKind(v1.VirtualMachinePresetGroupVersionKind)
	}

	return
}

func (v *vmPresets) Create(vm *v1.VirtualMachinePreset) (result *v1.VirtualMachinePreset, err error) {
	result = &v1.VirtualMachinePreset{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachinePresetGroupVersionKind)
	return
}

func (v *vmPresets) Update(vm *v1.VirtualMachinePreset) (result *v1.VirtualMachinePreset, err error) {
	result = &v1.VirtualMachinePreset{}
	err = v.restClient.Put().
		Name(vm.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachinePresetGroupVersionKind)
	return
}

func (v *vmPresets) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (v *vmPresets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachinePreset, err error) {
	result = &v1.VirtualMachinePreset{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
