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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
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
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachines) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions metav1.PatchOptions) (*v1.VirtualMachine, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachines) Restart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) ForceRestart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) Start(ctx context.Context, name string, startOptions *v1.StartOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) Stop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) ForceStop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) Migrate(ctx context.Context, name string, migrateOptions *v1.MigrateOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) PortForward(name string, port int, protocol string) (StreamInterface, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachines) MemoryDump(ctx context.Context, name string, memoryDumpRequest *v1.VirtualMachineMemoryDumpRequest) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachines) RemoveMemoryDump(ctx context.Context, name string) error {
	// TODO not implemented yet
	return nil
}
