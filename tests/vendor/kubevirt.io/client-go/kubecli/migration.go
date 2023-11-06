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

func (k *kubevirt) VirtualMachineInstanceMigration(namespace string) VirtualMachineInstanceMigrationInterface {
	return &migration{
		restClient: k.restClient,
		namespace:  namespace,
		resource:   "virtualmachineinstancemigrations",
	}
}

type migration struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new VirtualMachineInstanceMigration in the cluster to specified namespace
func (o *migration) Create(newMigration *v1.VirtualMachineInstanceMigration, options *k8smetav1.CreateOptions) (*v1.VirtualMachineInstanceMigration, error) {
	newMigrationResult := &v1.VirtualMachineInstanceMigration{}
	err := o.restClient.Post().
		Resource(o.resource).
		Namespace(o.namespace).
		Body(newMigration).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newMigrationResult)

	newMigrationResult.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)

	return newMigrationResult, err
}

// Get the VirtualMachineInstanceMigration from the cluster by its name and namespace
func (o *migration) Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachineInstanceMigration, error) {
	newVm := &v1.VirtualMachineInstanceMigration{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newVm)

	newVm.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)

	return newVm, err
}

// Update the VirtualMachineInstanceMigration instance in the cluster in given namespace
func (o *migration) Update(migration *v1.VirtualMachineInstanceMigration) (*v1.VirtualMachineInstanceMigration, error) {
	updatedVm := &v1.VirtualMachineInstanceMigration{}
	err := o.restClient.Put().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(migration.Name).
		Body(migration).
		Do(context.Background()).
		Into(updatedVm)

	updatedVm.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)

	return updatedVm, err
}

// Delete the defined VirtualMachineInstanceMigration in the cluster in defined namespace
func (o *migration) Delete(name string, options *k8smetav1.DeleteOptions) error {
	err := o.restClient.Delete().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()

	return err
}

// List all VirtualMachineInstanceMigrations in given namespace
func (o *migration) List(options *k8smetav1.ListOptions) (*v1.VirtualMachineInstanceMigrationList, error) {
	newVmiMigrationList := &v1.VirtualMachineInstanceMigrationList{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newVmiMigrationList)

	for i := range newVmiMigrationList.Items {
		newVmiMigrationList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)
	}

	return newVmiMigrationList, err
}

func (v *migration) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstanceMigration, err error) {
	result = &v1.VirtualMachineInstanceMigration{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return result, err
}

func (v *migration) PatchStatus(name string, pt types.PatchType, data []byte) (result *v1.VirtualMachineInstanceMigration, err error) {
	result = &v1.VirtualMachineInstanceMigration{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource("status").
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return
}

func (v *migration) UpdateStatus(vmi *v1.VirtualMachineInstanceMigration) (result *v1.VirtualMachineInstanceMigration, err error) {
	result = &v1.VirtualMachineInstanceMigration{}
	err = v.restClient.Put().
		Name(vmi.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource("status").
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceMigrationGroupVersionKind)
	return
}
