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
	"context"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
)

func (k *kubevirt) VirtualMachineInstancePreset(namespace string) VirtualMachineInstancePresetInterface {
	return &vmiPresets{k.restClient, namespace, "virtualmachineinstancepresets"}
}

type vmiPresets struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *vmiPresets) Get(name string, options k8smetav1.GetOptions) (vmi *v1.VirtualMachineInstancePreset, err error) {
	vmi = &v1.VirtualMachineInstancePreset{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(vmi)
	vmi.SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	return
}

func (v *vmiPresets) List(options k8smetav1.ListOptions) (vmiPresetList *v1.VirtualMachineInstancePresetList, err error) {
	vmiPresetList = &v1.VirtualMachineInstancePresetList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(vmiPresetList)
	for i := range vmiPresetList.Items {
		vmiPresetList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	}

	return
}

func (v *vmiPresets) Create(vmi *v1.VirtualMachineInstancePreset) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	return
}

func (v *vmiPresets) Update(vmi *v1.VirtualMachineInstancePreset) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = v.restClient.Put().
		Name(vmi.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstancePresetGroupVersionKind)
	return
}

func (v *vmiPresets) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()
}

func (v *vmiPresets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstancePreset, err error) {
	result = &v1.VirtualMachineInstancePreset{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return
}
