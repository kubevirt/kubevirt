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
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

func (k *kubevirt) KubeVirt(namespace string) KubeVirtInterface {
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
func (o *kv) Create(kv *v1.KubeVirt) (*v1.KubeVirt, error) {
	newKv, err := o.KubeVirtInterface.Create(context.Background(), kv, k8smetav1.CreateOptions{})
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
func (o *kv) Update(kv *v1.KubeVirt) (*v1.KubeVirt, error) {
	updatedKv, err := o.KubeVirtInterface.Update(context.Background(), kv, k8smetav1.UpdateOptions{})
	updatedKv.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)

	return updatedKv, err
}

// Delete the defined KubeVirt in the cluster in defined namespace
func (o *kv) Delete(name string, options *k8smetav1.DeleteOptions) error {
	opts := k8smetav1.DeleteOptions{}
	if options != nil {
		opts = *options
	}
	return o.KubeVirtInterface.Delete(context.Background(), name, opts)
}

// List all KubeVirts in given namespace
func (o *kv) List(options *k8smetav1.ListOptions) (*v1.KubeVirtList, error) {
	opts := k8smetav1.ListOptions{}
	if options != nil {
		opts = *options
	}
	newKvList, err := o.KubeVirtInterface.List(context.Background(), opts)
	for i := range newKvList.Items {
		newKvList.Items[i].SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	}

	return newKvList, err
}

func (o *kv) Patch(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions, subresources ...string) (result *v1.KubeVirt, err error) {
	opts := k8smetav1.PatchOptions{}
	if patchOptions != nil {
		opts = *patchOptions
	}
	return o.KubeVirtInterface.Patch(context.Background(), name, pt, data, opts, subresources...)
}

func (o *kv) PatchStatus(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions) (result *v1.KubeVirt, err error) {
	return o.Patch(name, pt, data, patchOptions, "status")
}

func (o *kv) UpdateStatus(kv *v1.KubeVirt) (result *v1.KubeVirt, err error) {
	result, err = o.KubeVirtInterface.UpdateStatus(context.Background(), kv, k8smetav1.UpdateOptions{})
	result.SetGroupVersionKind(v1.KubeVirtGroupVersionKind)
	return
}
