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
	"context"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
)

func (k *kubevirt) KubeVirt(namespace string) KubeVirtInterface {
	return &kv{
		restClient: k.restClient,
		namespace:  namespace,
		resource:   "kubevirts",
	}
}

type kv struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new KubeVirt in the cluster to specified namespace
func (o *kv) Create(vm *v1.KubeVirt) (*v1.KubeVirt, error) {
	newKv := &v1.KubeVirt{}
	err := o.restClient.Post().
		Resource(o.resource).
		Namespace(o.namespace).
		Body(vm).
		Do(context.Background()).
		Into(newKv)

	newKv.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)

	return newKv, err
}

// Get the KubeVirt from the cluster by its name and namespace
func (o *kv) Get(name string, options *k8smetav1.GetOptions) (*v1.KubeVirt, error) {
	newKv := &v1.KubeVirt{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newKv)

	newKv.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)

	return newKv, err
}

// Update the KubeVirt instance in the cluster in given namespace
func (o *kv) Update(vm *v1.KubeVirt) (*v1.KubeVirt, error) {
	updatedVm := &v1.KubeVirt{}
	err := o.restClient.Put().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(vm.Name).
		Body(vm).
		Do(context.Background()).
		Into(updatedVm)

	updatedVm.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)

	return updatedVm, err
}

// Delete the defined KubeVirt in the cluster in defined namespace
func (o *kv) Delete(name string, options *k8smetav1.DeleteOptions) error {
	err := o.restClient.Delete().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()

	return err
}

// List all KubeVirts in given namespace
func (o *kv) List(options *k8smetav1.ListOptions) (*v1.KubeVirtList, error) {
	newKvList := &v1.KubeVirtList{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newKvList)

	for _, vm := range newKvList.Items {
		vm.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	}

	return newKvList, err
}

func (v *kv) Patch(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions, subresources ...string) (result *v1.KubeVirt, err error) {
	result = &v1.KubeVirt{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		VersionedParams(patchOptions, scheme.ParameterCodec).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return result, err
}

func (v *kv) PatchStatus(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions) (result *v1.KubeVirt, err error) {
	result = &v1.KubeVirt{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource("status").
		VersionedParams(patchOptions, scheme.ParameterCodec).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return
}

func (v *kv) UpdateStatus(vmi *v1.KubeVirt) (result *v1.KubeVirt, err error) {
	result = &v1.KubeVirt{}
	err = v.restClient.Put().
		Name(vmi.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource("status").
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	return
}
