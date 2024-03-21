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
	"net/url"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

const (
	cannotMarshalJSONErrFmt = "Cannot Marshal to json: %s"
	vmSubresourceURLFmt     = "/apis/subresources.kubevirt.io/%s/namespaces/%s/virtualmachines/%s/%s"
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
func (v *vm) Create(ctx context.Context, vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	newVm, err := v.VirtualMachineInterface.Create(ctx, vm, k8smetav1.CreateOptions{})
	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Get the Virtual machine from the cluster by its name and namespace
func (v *vm) Get(ctx context.Context, name string, options *k8smetav1.GetOptions) (*v1.VirtualMachine, error) {
	opts := k8smetav1.GetOptions{}
	if options != nil {
		opts = *options
	}
	newVm, err := v.VirtualMachineInterface.Get(ctx, name, opts)
	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

func (v *vm) GetWithExpandedSpec(ctx context.Context, name string) (*v1.VirtualMachine, error) {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "expand-spec")
	newVm := &v1.VirtualMachine{}
	err := v.restClient.Get().
		AbsPath(uri).
		Do(ctx).
		Into(newVm)

	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return newVm, err
}

// Update the VirtualMachine instance in the cluster in given namespace
func (v *vm) Update(ctx context.Context, vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	updatedVm, err := v.VirtualMachineInterface.Update(ctx, vm, k8smetav1.UpdateOptions{})
	updatedVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)

	return updatedVm, err
}

// Delete the defined VirtualMachine in the cluster in defined namespace
func (v *vm) Delete(ctx context.Context, name string, options *k8smetav1.DeleteOptions) error {
	opts := k8smetav1.DeleteOptions{}
	if options != nil {
		opts = *options
	}
	return v.VirtualMachineInterface.Delete(ctx, name, opts)
}

// List all VirtualMachines in given namespace
func (v *vm) List(ctx context.Context, options *k8smetav1.ListOptions) (*v1.VirtualMachineList, error) {
	opts := k8smetav1.ListOptions{}
	if options != nil {
		opts = *options
	}
	newVmList, err := v.VirtualMachineInterface.List(ctx, opts)
	for i := range newVmList.Items {
		newVmList.Items[i].SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	}

	return newVmList, err
}

func (v *vm) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions, subresources ...string) (result *v1.VirtualMachine, err error) {
	opts := k8smetav1.PatchOptions{}
	if patchOptions != nil {
		opts = *patchOptions
	}
	return v.VirtualMachineInterface.Patch(ctx, name, pt, data, opts, subresources...)
}

func (v *vm) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions *k8smetav1.PatchOptions) (result *v1.VirtualMachine, err error) {
	return v.Patch(ctx, name, pt, data, patchOptions, "status")
}

func (v *vm) UpdateStatus(ctx context.Context, vmi *v1.VirtualMachine) (result *v1.VirtualMachine, err error) {
	result, err = v.VirtualMachineInterface.UpdateStatus(ctx, vmi, k8smetav1.UpdateOptions{})
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vm) Restart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	body, err := json.Marshal(restartOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "restart")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}

func (v *vm) ForceRestart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	body, err := json.Marshal(restartOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "restart")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}

func (v *vm) Start(ctx context.Context, name string, startOptions *v1.StartOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "start")

	optsJson, err := json.Marshal(startOptions)
	if err != nil {
		return err
	}
	return v.restClient.Put().AbsPath(uri).Body(optsJson).Do(ctx).Error()
}

func (v *vm) Stop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "stop")
	optsJson, err := json.Marshal(stopOptions)
	if err != nil {
		return err
	}
	return v.restClient.Put().AbsPath(uri).Body(optsJson).Do(ctx).Error()
}

func (v *vm) ForceStop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	body, err := json.Marshal(stopOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "stop")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}

func (v *vm) Migrate(ctx context.Context, name string, migrateOptions *v1.MigrateOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "migrate")
	optsJson, err := json.Marshal(migrateOptions)
	if err != nil {
		return err
	}
	return v.restClient.Put().AbsPath(uri).Body(optsJson).Do(ctx).Error()
}

func (v *vm) MemoryDump(ctx context.Context, name string, memoryDumpRequest *v1.VirtualMachineMemoryDumpRequest) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "memorydump")

	JSON, err := json.Marshal(memoryDumpRequest)
	if err != nil {
		return err
	}

	return v.restClient.Put().AbsPath(uri).Body([]byte(JSON)).Do(ctx).Error()
}

func (v *vm) RemoveMemoryDump(ctx context.Context, name string) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "removememorydump")

	return v.restClient.Put().AbsPath(uri).Do(ctx).Error()
}

func (v *vm) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "addvolume")

	JSON, err := json.Marshal(addVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().AbsPath(uri).Body([]byte(JSON)).Do(ctx).Error()
}

func (v *vm) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	uri := fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion, v.namespace, name, "removevolume")

	JSON, err := json.Marshal(removeVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().AbsPath(uri).Body([]byte(JSON)).Do(ctx).Error()
}

func (v *vm) PortForward(name string, port int, protocol string) (StreamInterface, error) {
	return asyncSubresourceHelper(v.config, v.resource, v.namespace, name, buildPortForwardResourcePath(port, protocol), url.Values{})
}
