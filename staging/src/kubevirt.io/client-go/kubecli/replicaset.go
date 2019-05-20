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
	autov1 "k8s.io/api/autoscaling/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/client-go/api/v1"
)

func (k *kubevirt) ReplicaSet(namespace string) ReplicaSetInterface {
	return &rc{k.restClient, namespace, "virtualmachineinstancereplicasets"}
}

type rc struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

func (v *rc) GetScale(replicaSetName string, options k8smetav1.GetOptions) (result *autov1.Scale, err error) {
	result = &autov1.Scale{}
	err = v.restClient.Get().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(replicaSetName).
		SubResource("scale").
		Do().
		Into(result)
	return
}

func (v *rc) UpdateScale(replicaSetName string, scale *autov1.Scale) (result *autov1.Scale, err error) {
	result = &autov1.Scale{}
	err = v.restClient.Put().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(replicaSetName).
		SubResource("scale").
		Body(scale).
		Do().
		Into(result)
	return
}

func (v *rc) Get(name string, options k8smetav1.GetOptions) (replicaset *v1.VirtualMachineInstanceReplicaSet, err error) {
	replicaset = &v1.VirtualMachineInstanceReplicaSet{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(replicaset)
	replicaset.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) List(options k8smetav1.ListOptions) (replicasetList *v1.VirtualMachineInstanceReplicaSetList, err error) {
	replicasetList = &v1.VirtualMachineInstanceReplicaSetList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(replicasetList)
	for _, replicaset := range replicasetList.Items {
		replicaset.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	}

	return
}

func (v *rc) Create(replicaset *v1.VirtualMachineInstanceReplicaSet) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result = &v1.VirtualMachineInstanceReplicaSet{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(replicaset).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) Update(replicaset *v1.VirtualMachineInstanceReplicaSet) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result = &v1.VirtualMachineInstanceReplicaSet{}
	err = v.restClient.Put().
		Name(replicaset.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(replicaset).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (v *rc) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result = &v1.VirtualMachineInstanceReplicaSet{}
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
