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
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func (k *kubevirt) OfflineVirtualMachine(namespace string) OfflineVirtualMachineInterface {
	return &ovm{
		restClient: k.restClient,
		namespace:  namespace,
		resource:   "offlinevirtualmachines",
	}
}

type ovm struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new OfflineVirtualMachine in the cluster to specified namespace
func (o *ovm) Create(offlinevm *v1.OfflineVirtualMachine) (*v1.OfflineVirtualMachine, error) {
	newOvm := &v1.OfflineVirtualMachine{}
	err := o.restClient.Post().
		Resource(o.resource).
		Namespace(o.namespace).
		Body(offlinevm).
		Do().
		Into(newOvm)

	newOvm.SetGroupVersionKind(v1.OfflineVirtualMachineGroupVersionKind)

	return newOvm, err
}

// Get the OfflineVirtual machine from the cluster by its name and namespace
func (o *ovm) Get(name string, options *k8smetav1.GetOptions) (*v1.OfflineVirtualMachine, error) {
	newOvm := &v1.OfflineVirtualMachine{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(newOvm)

	newOvm.SetGroupVersionKind(v1.OfflineVirtualMachineGroupVersionKind)

	return newOvm, err
}

// Update the OfflineVirtualMachine instance in the cluster in given namespace
func (o *ovm) Update(offlinevm *v1.OfflineVirtualMachine) (*v1.OfflineVirtualMachine, error) {
	updatedOvm := &v1.OfflineVirtualMachine{}
	err := o.restClient.Put().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(offlinevm.Name).
		Body(offlinevm).
		Do().
		Into(updatedOvm)

	updatedOvm.SetGroupVersionKind(v1.OfflineVirtualMachineGroupVersionKind)

	return updatedOvm, err
}

// Delete the defined OfflineVirtualMachine in the cluster in defined namespace
func (o *ovm) Delete(name string, options *k8smetav1.DeleteOptions) error {
	err := o.restClient.Delete().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		Body(options).
		Do().
		Error()

	return err
}

// List all OfflineVirtualMachines in given namespace
func (o *ovm) List(options *k8smetav1.ListOptions) (*v1.OfflineVirtualMachineList, error) {
	newOvmList := &v1.OfflineVirtualMachineList{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(newOvmList)

	for _, ovm := range newOvmList.Items {
		ovm.SetGroupVersionKind(v1.OfflineVirtualMachineGroupVersionKind)
	}

	return newOvmList, err
}
