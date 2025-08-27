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

package fake

import (
	"context"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	kubevirtv1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	fake2 "kubevirt.io/client-go/testing"
)

func (c *fakeVirtualMachines) GetWithExpandedSpec(ctx context.Context, name string) (*v1.VirtualMachine, error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "expand-spec", name), &v1.VirtualMachine{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1.VirtualMachine), err
}

func (c *fakeVirtualMachines) PatchStatus(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions k8smetav1.PatchOptions) (*v1.VirtualMachine, error) {
	return c.Patch(ctx, name, pt, data, patchOptions, "status")
}

func (c *fakeVirtualMachines) Restart(ctx context.Context, name string, restartOptions *v1.RestartOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "restart", name, restartOptions), nil)

	return err
}

func (c *fakeVirtualMachines) Start(ctx context.Context, name string, startOptions *v1.StartOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "start", name, startOptions), nil)

	return err
}

func (c *fakeVirtualMachines) Stop(ctx context.Context, name string, stopOptions *v1.StopOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "stop", name, stopOptions), nil)

	return err
}

func (c *fakeVirtualMachines) Migrate(ctx context.Context, name string, migrateOptions *v1.MigrateOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "migrate", name, migrateOptions), nil)

	return err
}

func (c *fakeVirtualMachines) MemoryDump(ctx context.Context, name string, memoryDumpRequest *v1.VirtualMachineMemoryDumpRequest) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "memorydump", name, memoryDumpRequest), nil)

	return err
}

func (c *fakeVirtualMachines) RemoveMemoryDump(ctx context.Context, name string) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "removememorydump", name, struct{}{}), nil)

	return err
}

func (c *fakeVirtualMachines) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "addvolume", name, addVolumeOptions), nil)

	return err
}

func (c *fakeVirtualMachines) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "removevolume", name, removeVolumeOptions), nil)

	return err
}

func (c *fakeVirtualMachines) PortForward(name string, port int, protocol string) (kubevirtv1.StreamInterface, error) {
	return nil, nil
}

func (c *fakeVirtualMachines) ObjectGraph(ctx context.Context, name string, objectGraphOptions *v1.ObjectGraphOptions) (v1.ObjectGraphNode, error) {
	obj, err := c.Fake.
		Invokes(fake2.NewGetSubresourceAction(c.Resource(), c.Namespace(), "objectgraph", name, objectGraphOptions), nil)

	return *obj.(*v1.ObjectGraphNode), err
}
