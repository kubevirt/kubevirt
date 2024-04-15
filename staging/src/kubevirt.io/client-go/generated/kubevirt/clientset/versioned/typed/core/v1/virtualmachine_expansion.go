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
 * Copyright The KubeVirt Authors
 *
 */

package v1

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

const (
	cannotMarshalJSONErrFmt = "cannot Marshal to json: %s"
	vmSubresourceURLFmt     = "/apis/subresources.kubevirt.io/%s"
)

type VirtualMachineExpansion interface {
	GetWithExpandedSpec(ctx context.Context, name string) (*v1.VirtualMachine, error)
	PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions metav1.PatchOptions) (*v1.VirtualMachine, error)
	Restart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error
	ForceRestart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error
	Start(ctx context.Context, name string, startOptions *v1.StartOptions) error
	Stop(ctx context.Context, name string, stopOptions *v1.StopOptions) error
	ForceStop(ctx context.Context, name string, stopOptions *v1.StopOptions) error
	Migrate(ctx context.Context, name string, migrateOptions *v1.MigrateOptions) error
	AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error
	RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error
	PortForward(name string, port int, protocol string) (StreamInterface, error)
	MemoryDump(ctx context.Context, name string, memoryDumpRequest *v1.VirtualMachineMemoryDumpRequest) error
	RemoveMemoryDump(ctx context.Context, name string) error
}

func (c *virtualMachines) GetWithExpandedSpec(ctx context.Context, name string) (*v1.VirtualMachine, error) {
	newVm := &v1.VirtualMachine{}
	err := c.client.Get().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("expand-spec").
		Do(ctx).
		Into(newVm)
	newVm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return newVm, err
}

func (c *virtualMachines) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions metav1.PatchOptions) (*v1.VirtualMachine, error) {
	return c.Patch(ctx, name, pt, data, patchOptions, "status")
}

func (c *virtualMachines) Restart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	body, err := json.Marshal(restartOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("restart").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachines) ForceRestart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	body, err := json.Marshal(restartOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("restart").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachines) Start(ctx context.Context, name string, startOptions *v1.StartOptions) error {
	optsJson, err := json.Marshal(startOptions)
	if err != nil {
		return err
	}
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("start").
		Body(optsJson).
		Do(ctx).
		Error()
}

func (c *virtualMachines) Stop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	optsJson, err := json.Marshal(stopOptions)
	if err != nil {
		return err
	}
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("stop").
		Body(optsJson).
		Do(ctx).
		Error()
}

func (c *virtualMachines) ForceStop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	body, err := json.Marshal(stopOptions)
	if err != nil {
		return fmt.Errorf(cannotMarshalJSONErrFmt, err)
	}
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("stop").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachines) Migrate(ctx context.Context, name string, migrateOptions *v1.MigrateOptions) error {
	optsJson, err := json.Marshal(migrateOptions)
	if err != nil {
		return err
	}
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("migrate").
		Body(optsJson).
		Do(ctx).
		Error()
}

func (c *virtualMachines) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	body, err := json.Marshal(addVolumeOptions)
	if err != nil {
		return err
	}

	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("addvolume").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachines) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	body, err := json.Marshal(removeVolumeOptions)
	if err != nil {
		return err
	}

	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("removevolume").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachines) PortForward(name string, port int, protocol string) (StreamInterface, error) {
	// TODO not implemented yet
	//  requires clientConfig
	return nil, fmt.Errorf("PortForward is not implemented yet in generated client")
}

func (c *virtualMachines) MemoryDump(ctx context.Context, name string, memoryDumpRequest *v1.VirtualMachineMemoryDumpRequest) error {
	body, err := json.Marshal(memoryDumpRequest)
	if err != nil {
		return err
	}

	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("memorydump").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachines) RemoveMemoryDump(ctx context.Context, name string) error {
	return c.client.Put().
		AbsPath(fmt.Sprintf(vmSubresourceURLFmt, v1.ApiStorageVersion)).
		Namespace(c.ns).
		Resource("virtualmachines").
		Name(name).
		SubResource("removememorydump").
		Do(ctx).
		Error()
}
