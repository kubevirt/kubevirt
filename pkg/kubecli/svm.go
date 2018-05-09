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

func (k *kubevirt) StatefulVirtualMachine(namespace string) StatefulVirtualMachineInterface {
	return &svm{
		restClient: k.restClient,
		namespace:  namespace,
		resource:   "statefulvirtualmachines",
	}
}

type svm struct {
	restClient *rest.RESTClient
	namespace  string
	resource   string
}

// Create new StatefulVirtualMachine in the cluster to specified namespace
func (o *svm) Create(statefulvm *v1.StatefulVirtualMachine) (*v1.StatefulVirtualMachine, error) {
	newSvm := &v1.StatefulVirtualMachine{}
	err := o.restClient.Post().
		Resource(o.resource).
		Namespace(o.namespace).
		Body(statefulvm).
		Do().
		Into(newSvm)

	newSvm.SetGroupVersionKind(v1.StatefulVirtualMachineGroupVersionKind)

	return newSvm, err
}

// Get the StatefulVirtual machine from the cluster by its name and namespace
func (o *svm) Get(name string, options *k8smetav1.GetOptions) (*v1.StatefulVirtualMachine, error) {
	newSvm := &v1.StatefulVirtualMachine{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(newSvm)

	newSvm.SetGroupVersionKind(v1.StatefulVirtualMachineGroupVersionKind)

	return newSvm, err
}

// Update the StatefulVirtualMachine instance in the cluster in given namespace
func (o *svm) Update(statefulvm *v1.StatefulVirtualMachine) (*v1.StatefulVirtualMachine, error) {
	updatedSvm := &v1.StatefulVirtualMachine{}
	err := o.restClient.Put().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(statefulvm.Name).
		Body(statefulvm).
		Do().
		Into(updatedSvm)

	updatedSvm.SetGroupVersionKind(v1.StatefulVirtualMachineGroupVersionKind)

	return updatedSvm, err
}

// Delete the defined StatefulVirtualMachine in the cluster in defined namespace
func (o *svm) Delete(name string, options *k8smetav1.DeleteOptions) error {
	err := o.restClient.Delete().
		Resource(o.resource).
		Namespace(o.namespace).
		Name(name).
		Body(options).
		Do().
		Error()

	return err
}

// List all StatefulVirtualMachines in given namespace
func (o *svm) List(options *k8smetav1.ListOptions) (*v1.StatefulVirtualMachineList, error) {
	newSvmList := &v1.StatefulVirtualMachineList{}
	err := o.restClient.Get().
		Resource(o.resource).
		Namespace(o.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do().
		Into(newSvmList)

	for _, svm := range newSvmList.Items {
		svm.SetGroupVersionKind(v1.StatefulVirtualMachineGroupVersionKind)
	}

	return newSvmList, err
}
