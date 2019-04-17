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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func (k *kubevirt) VirtualMachineSnapshot(namespace string) VirtualMachineSnapshotInterface {
	return &snapshot{
		k.restClient,
		namespace,
		"virtualmachinesnapshots",
	}
}

type snapshot struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *snapshot) Get(name string, options k8smetav1.GetOptions) (ss *v1.VirtualMachineSnapshot, err error) {
	ss = &v1.VirtualMachineSnapshot{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(ss)
	ss.SetGroupVersionKind(v1.VirtualMachineSnapshotGroupVersionKind)
	return
}

func (v *snapshot) List(options k8smetav1.ListOptions) (ssList *v1.VirtualMachineSnapshotList, err error) {
	ssList = &v1.VirtualMachineSnapshotList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(ssList)
	for _, ss := range ssList.Items {
		ss.SetGroupVersionKind(v1.VirtualMachineSnapshotGroupVersionKind)
	}

	return
}

func (v *snapshot) Create(ss *v1.VirtualMachineSnapshot) (result *v1.VirtualMachineSnapshot, err error) {
	result = &v1.VirtualMachineSnapshot{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(ss).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineSnapshotGroupVersionKind)
	return
}

func (v *snapshot) Update(ss *v1.VirtualMachineSnapshot) (result *v1.VirtualMachineSnapshot, err error) {
	result = &v1.VirtualMachineSnapshot{}
	err = v.restClient.Put().
		Name(ss.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(ss).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineSnapshotGroupVersionKind)
	return
}

func (v *snapshot) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (v *snapshot) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineSnapshot, err error) {
	result = &v1.VirtualMachineSnapshot{}
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
