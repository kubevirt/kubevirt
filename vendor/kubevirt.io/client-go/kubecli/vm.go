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
	"net/url"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

func (k *kubevirt) VirtualMachine(namespace string) VirtualMachineInterface {
	return &vm{
		VirtualMachineInterface: k.GeneratedKubeVirtClient().KubevirtV1().VirtualMachines(namespace),
		restClient:              k.restClient,
		config:                  k.config,
		namespace:               namespace,
		resource:                "virtualmachines",
	}
}

type vm struct {
	kvcorev1.VirtualMachineInterface
	restClient *rest.RESTClient
	config     *rest.Config
	namespace  string
	resource   string
}

// Create new VirtualMachine in the cluster to specified namespace
func (v *vm) Create(ctx context.Context, vm *v1.VirtualMachine, opts k8smetav1.CreateOptions) (*v1.VirtualMachine, error) {
	newVm, err := v.VirtualMachineInterface.Create(ctx, vm, opts)
	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Get the Virtual machine from the cluster by its name and namespace
func (v *vm) Get(ctx context.Context, name string, options k8smetav1.GetOptions) (*v1.VirtualMachine, error) {
	newVm, err := v.VirtualMachineInterface.Get(ctx, name, options)
	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Update the VirtualMachine instance in the cluster in given namespace
func (v *vm) Update(ctx context.Context, vm *v1.VirtualMachine, opts k8smetav1.UpdateOptions) (*v1.VirtualMachine, error) {
	updatedVm, err := v.VirtualMachineInterface.Update(ctx, vm, opts)
	updatedVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return updatedVm, err
}

// Delete the defined VirtualMachine in the cluster in defined namespace
func (v *vm) Delete(ctx context.Context, name string, options k8smetav1.DeleteOptions) error {
	return v.VirtualMachineInterface.Delete(ctx, name, options)
}

// List all VirtualMachines in given namespace
func (v *vm) List(ctx context.Context, options k8smetav1.ListOptions) (*v1.VirtualMachineList, error) {
	newVmList, err := v.VirtualMachineInterface.List(ctx, options)
	for i := range newVmList.Items {
		newVmList.Items[i].SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	}

	return newVmList, err
}

func (v *vm) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions k8smetav1.PatchOptions, subresources ...string) (result *v1.VirtualMachine, err error) {
	return v.VirtualMachineInterface.Patch(ctx, name, pt, data, patchOptions, subresources...)
}

func (v *vm) UpdateStatus(ctx context.Context, vmi *v1.VirtualMachine, opts k8smetav1.UpdateOptions) (result *v1.VirtualMachine, err error) {
	result, err = v.VirtualMachineInterface.UpdateStatus(ctx, vmi, opts)
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vm) PortForward(name string, port int, protocol string) (kvcorev1.StreamInterface, error) {
	return kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, buildPortForwardResourcePath(port, protocol), url.Values{})
}
