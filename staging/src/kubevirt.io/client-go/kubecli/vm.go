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

	"encoding/json"
	"fmt"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
)

const (
	cannotMarshalJSONErrFmt = "Cannot Marshal to json: %s"
	vmSubresourceURLFmt     = "/apis/subresources.kubevirt.io/%s/namespaces/%s/virtualmachines/%s/%s"
)

func (k *kubevirt) VirtualMachine(namespace string) VirtualMachineInterface {
	return &vm{
		restClient: k.restClient,
		config:     k.config,
		namespace:  namespace,
		resource:   "virtualmachines",
	}
}

type vm struct {
	restClient *rest.RESTClient
	config     *rest.Config
	namespace  string
	resource   string
}

// Create new VirtualMachine in the cluster to specified namespace
func (v *vm) Create(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	newVm := &v1.VirtualMachine{}
	err := v.restClient.Post().
		Resource(v.resource).
		Namespace(v.namespace).
		Body(vm).
		Do(context.Background()).
		Into(newVm)

	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Get the Virtual machine from the cluster by its name and namespace
func (v *vm) Get(name string, options *k8smetav1.GetOptions) (*v1.VirtualMachine, error) {
	newVm := &v1.VirtualMachine{}
	err := v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newVm)

	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Update the VirtualMachine instance in the cluster in given namespace
func (v *vm) Update(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	updatedVm := &v1.VirtualMachine{}
	err := v.restClient.Put().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(vm.Name).
		Body(vm).
		Do(context.Background()).
		Into(updatedVm)

	updatedVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return updatedVm, err
}

// Delete the defined VirtualMachine in the cluster in defined namespace
func (v *vm) Delete(name string, options *k8smetav1.DeleteOptions) error {
	err := v.restClient.Delete().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()

	return err
}

// List all VirtualMachines in given namespace
func (v *vm) List(options *k8smetav1.ListOptions) (*v1.VirtualMachineList, error) {
	newVmList := &v1.VirtualMachineList{}
	err := v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(newVmList)

	for _, vm := range newVmList.Items {
		vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	}

	return newVmList, err
}

func (v *vm) Patch(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions, subresources ...string) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		VersionedParams(patchOptions, scheme.ParameterCodec).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return result, err
}

func (v *vm) PatchStatus(name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource("status").
		VersionedParams(patchOptions, scheme.ParameterCodec).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return
}

func (v *vm) UpdateStatus(vmi *v1.VirtualMachine) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Put().
		Name(vmi.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource("status").
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vm) Restart(name string, restartOptions *v1.RestartOptions) error {
	body, err := json.Marshal(restartOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "restart")
	return v.restClient.Put().RequestURI(uri).Body(body).Do(context.Background()).Error()
}

func (v *vm) ForceRestart(name string, restartOptions *v1.RestartOptions) error {
	body, err := json.Marshal(restartOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "restart")
	return v.restClient.Put().RequestURI(uri).Body(body).Do(context.Background()).Error()
}

func (v *vm) Start(name string, startOptions *v1.StartOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "start")

	optsJson, err := json.Marshal(startOptions)
	if err != nil {
		return err
	}
	return v.restClient.Put().RequestURI(uri).Body(optsJson).Do(context.Background()).Error()
}

func (v *vm) Stop(name string, stopOptions *v1.StopOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "stop")
	optsJson, err := json.Marshal(stopOptions)
	if err != nil {
		return err
	}
	return v.restClient.Put().RequestURI(uri).Body(optsJson).Do(context.Background()).Error()
}

func (v *vm) ForceStop(name string, stopOptions *v1.StopOptions) error {
	body, err := json.Marshal(stopOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "stop")
	return v.restClient.Put().RequestURI(uri).Body(body).Do(context.Background()).Error()
}

func (v *vm) Migrate(name string, migrateOptions *v1.MigrateOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "migrate")
	optsJson, err := json.Marshal(migrateOptions)
	if err != nil {
		return err
	}
	return v.restClient.Put().RequestURI(uri).Body(optsJson).Do(context.Background()).Error()
}

func (v *vm) AddVolume(name string, addVolumeOptions *v1.AddVolumeOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "addvolume")

	JSON, err := json.Marshal(addVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().RequestURI(uri).Body([]byte(JSON)).Do(context.Background()).Error()
}

func (v *vm) RemoveVolume(name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "removevolume")

	JSON, err := json.Marshal(removeVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().RequestURI(uri).Body([]byte(JSON)).Do(context.Background()).Error()
}

func (v *vm) PortForward(name string, port int, protocol string) (StreamInterface, error) {
	return asyncSubresourceHelper(v.config, v.resource, v.namespace, name, buildPortForwardResourcePath(port, protocol))
}
