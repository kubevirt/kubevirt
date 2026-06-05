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
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

func (k *kubevirtClient) ReplicaSet(namespace string) ReplicaSetInterface {
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

func (v *rc) Get(ctx context.Context, name string, options k8smetav1.GetOptions) (replicaset *v1.VirtualMachineInstanceReplicaSet, err error) {
	replicaset, err = v.VirtualMachineInstanceReplicaSetInterface.Get(ctx, name, options)
	replicaset.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) List(ctx context.Context, options k8smetav1.ListOptions) (replicasetList *v1.VirtualMachineInstanceReplicaSetList, err error) {
	replicasetList, err = v.VirtualMachineInstanceReplicaSetInterface.List(ctx, options)
	for i := range replicasetList.Items {
		replicasetList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	}
	return
}

func (v *rc) Create(ctx context.Context, replicaset *v1.VirtualMachineInstanceReplicaSet, opts k8smetav1.CreateOptions) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result, err = v.VirtualMachineInstanceReplicaSetInterface.Create(ctx, replicaset, opts)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) Update(ctx context.Context, replicaset *v1.VirtualMachineInstanceReplicaSet, opts k8smetav1.UpdateOptions) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result, err = v.VirtualMachineInstanceReplicaSetInterface.Update(ctx, replicaset, opts)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}

func (v *rc) Delete(ctx context.Context, name string, options k8smetav1.DeleteOptions) error {
	return v.VirtualMachineInstanceReplicaSetInterface.Delete(ctx, name, options)
}

func (v *rc) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts k8smetav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	return v.VirtualMachineInstanceReplicaSetInterface.Patch(ctx, name, pt, data, opts, subresources...)
}

func (v *rc) UpdateStatus(ctx context.Context, replicaset *v1.VirtualMachineInstanceReplicaSet, opts k8smetav1.UpdateOptions) (result *v1.VirtualMachineInstanceReplicaSet, err error) {
	result, err = v.VirtualMachineInstanceReplicaSetInterface.UpdateStatus(ctx, replicaset, opts)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceReplicaSetGroupVersionKind)
	return
}
