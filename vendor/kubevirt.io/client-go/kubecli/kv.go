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
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
)

func (k *kubevirtClient) KubeVirt(namespace string) KubeVirtInterface {
	return &kv{
		KubeVirtInterface: k.GeneratedKubeVirtClient().KubevirtV1().KubeVirts(namespace),
		restClient:        k.restClient,
		namespace:         namespace,
		resource:          "kubevirts",
	}
}

type kv struct {
	kvcorev1.KubeVirtInterface
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new KubeVirt in the cluster to specified namespace
func (o *kv) Create(ctx context.Context, kv *v1.KubeVirt, opts k8smetav1.CreateOptions) (*v1.KubeVirt, error) {
	newKv, err := o.KubeVirtInterface.Create(ctx, kv, opts)
	newKv.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	return newKv, err
}

// Get the KubeVirt from the cluster by its name and namespace
func (o *kv) Get(ctx context.Context, name string, options k8smetav1.GetOptions) (*v1.KubeVirt, error) {
	newKv, err := o.KubeVirtInterface.Get(ctx, name, options)
	newKv.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	return newKv, err
}

// Update the KubeVirt instance in the cluster in given namespace
func (o *kv) Update(ctx context.Context, kv *v1.KubeVirt, opts k8smetav1.UpdateOptions) (*v1.KubeVirt, error) {
	updatedKv, err := o.KubeVirtInterface.Update(ctx, kv, opts)
	updatedKv.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	return updatedKv, err
}

// Delete the defined KubeVirt in the cluster in defined namespace
func (o *kv) Delete(ctx context.Context, name string, options k8smetav1.DeleteOptions) error {
	return o.KubeVirtInterface.Delete(ctx, name, options)
}

// List all KubeVirts in given namespace
func (o *kv) List(ctx context.Context, options k8smetav1.ListOptions) (*v1.KubeVirtList, error) {
	newKvList, err := o.KubeVirtInterface.List(ctx, options)
	for i := range newKvList.Items {
		newKvList.Items[i].SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	}
	return newKvList, err
}

func (o *kv) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions k8smetav1.PatchOptions, subresources ...string) (result *v1.KubeVirt, err error) {
	return o.KubeVirtInterface.Patch(ctx, name, pt, data, patchOptions, subresources...)
}

func (o *kv) UpdateStatus(ctx context.Context, kv *v1.KubeVirt, opts k8smetav1.UpdateOptions) (result *v1.KubeVirt, err error) {
	result, err = o.KubeVirtInterface.UpdateStatus(ctx, kv, opts)
	result.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	return
}
