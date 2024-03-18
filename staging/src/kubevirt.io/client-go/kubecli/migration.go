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

func (k *kubevirt) VirtualMachineInstanceMigration(namespace string) VirtualMachineInstanceMigrationInterface {
	return &migration{
		VirtualMachineInstanceMigrationInterface: k.GeneratedKubeVirtClient().KubevirtV1().VirtualMachineInstanceMigrations(namespace),
		restClient:                               k.restClient,
		namespace:                                namespace,
		resource:                                 "virtualmachineinstancemigrations",
	}
}

type migration struct {
	kvcorev1.VirtualMachineInstanceMigrationInterface
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new VirtualMachineInstanceMigration in the cluster to specified namespace
func (o *migration) Create(ctx context.Context, newMigration *v1.VirtualMachineInstanceMigration, options k8smetav1.CreateOptions) (*v1.VirtualMachineInstanceMigration, error) {
	newMigrationResult, err := o.VirtualMachineInstanceMigrationInterface.Create(ctx, newMigration, options)
	newMigrationResult.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)

	return newMigrationResult, err
}

// Get the VirtualMachineInstanceMigration from the cluster by its name and namespace
func (o *migration) Get(ctx context.Context, name string, options k8smetav1.GetOptions) (*v1.VirtualMachineInstanceMigration, error) {
	newVm, err := o.VirtualMachineInstanceMigrationInterface.Get(ctx, name, options)
	newVm.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)

	return newVm, err
}

// Update the VirtualMachineInstanceMigration instance in the cluster in given namespace
func (o *migration) Update(ctx context.Context, migration *v1.VirtualMachineInstanceMigration, opts k8smetav1.UpdateOptions) (*v1.VirtualMachineInstanceMigration, error) {
	updatedVm, err := o.VirtualMachineInstanceMigrationInterface.Update(ctx, migration, opts)
	updatedVm.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)

	return updatedVm, err
}

// Delete the defined VirtualMachineInstanceMigration in the cluster in defined namespace
func (o *migration) Delete(ctx context.Context, name string, options k8smetav1.DeleteOptions) error {
	return o.VirtualMachineInstanceMigrationInterface.Delete(ctx, name, options)
}

// List all VirtualMachineInstanceMigrations in given namespace
func (o *migration) List(ctx context.Context, options k8smetav1.ListOptions) (*v1.VirtualMachineInstanceMigrationList, error) {
	newVmiMigrationList, err := o.VirtualMachineInstanceMigrationInterface.List(ctx, options)
	for i := range newVmiMigrationList.Items {
		newVmiMigrationList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)
	}

	return newVmiMigrationList, err
}

func (o *migration) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts k8smetav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstanceMigration, err error) {
	return o.VirtualMachineInstanceMigrationInterface.Patch(ctx, name, pt, data, opts, subresources...)
}

func (o *migration) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, opts k8smetav1.PatchOptions) (result *v1.VirtualMachineInstanceMigration, err error) {
	return o.Patch(ctx, name, pt, data, opts, "status")
}

func (o *migration) UpdateStatus(ctx context.Context, vmim *v1.VirtualMachineInstanceMigration, opts k8smetav1.UpdateOptions) (result *v1.VirtualMachineInstanceMigration, err error) {
	result, err = o.VirtualMachineInstanceMigrationInterface.UpdateStatus(ctx, vmim, opts)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)
	return
}
