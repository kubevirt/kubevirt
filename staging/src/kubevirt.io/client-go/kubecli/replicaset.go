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

	autov1 "k8s.io/api/autoscaling/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

func (k *kubevirt) ReplicaSet(namespace string) ReplicaSetInterface {
	return &rc{
		k.GeneratedKubeVirtClient().KubevirtV1().VirtualMachineInstanceReplicaSets(namespace),
		k.restClient,
		namespace,
		"virtualmachineinstancereplicasets",
	}
}

type rc struct {
	kvcorev1.VirtualMachineInstanceReplicaSetInterface
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
		Do(context.Background()).
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
		Do(context.Background()).
		Into(result)
	return
}

func (v *rc) Get(name string, options k8smetav1.GetOptions) (replicaset *v1.VirtualMachineInstanceReplicaSet, err error) {
	replicaset, err = v.VirtualMachineInstanceReplicaSetInterface.Get(context.Background(), name, options)
	replicaset.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) List(options k8smetav1.ListOptions) (replicasetList *v1.VirtualMachineInstanceReplicaSetList, err error) {
	replicasetList, err = v.VirtualMachineInstanceReplicaSetInterface.List(context.Background(), options)
	for i := range replicasetList.Items {
		replicasetList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	}

	return
}

func (v *rc) Create(replicaset *v1.VirtualMachineInstanceReplicaSet) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result, err = v.VirtualMachineInstanceReplicaSetInterface.Create(context.Background(), replicaset, k8smetav1.CreateOptions{})
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) Update(replicaset *v1.VirtualMachineInstanceReplicaSet) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result, err = v.VirtualMachineInstanceReplicaSetInterface.Update(context.Background(), replicaset, k8smetav1.UpdateOptions{})
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) Delete(name string, options *k8smetav1.DeleteOptions) error {
	opts := k8smetav1.DeleteOptions{}
	if options != nil {
		opts = *options
	}
	return v.VirtualMachineInstanceReplicaSetInterface.Delete(context.Background(), name, opts)
}

func (v *rc) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	return v.VirtualMachineInstanceReplicaSetInterface.Patch(context.Background(), name, pt, data, k8smetav1.PatchOptions{}, subresources...)
}

func (v *rc) PatchStatus(name string, pt types.PatchType, data []byte) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	return v.Patch(name, pt, data, "status")
}

func (v *rc) UpdateStatus(vmi *v1.VirtualMachineInstanceReplicaSet) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result, err = v.VirtualMachineInstanceReplicaSetInterface.UpdateStatus(context.Background(), vmi, k8smetav1.UpdateOptions{})
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}
