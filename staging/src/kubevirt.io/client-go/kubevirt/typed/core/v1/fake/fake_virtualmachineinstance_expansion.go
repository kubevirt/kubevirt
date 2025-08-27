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
	"time"

	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"
	fake2 "kubevirt.io/client-go/testing"
)

func (c *fakeVirtualMachineInstances) SerialConsole(name string, options *kvcorev1.SerialConsoleOptions) (kvcorev1.StreamInterface, error) {
	return nil, nil
}

func (c *fakeVirtualMachineInstances) USBRedir(vmiName string) (kvcorev1.StreamInterface, error) {
	return nil, nil
}

func (c *fakeVirtualMachineInstances) VNC(name string, preserveSession bool) (kvcorev1.StreamInterface, error) {
	return nil, nil
}

func (c *fakeVirtualMachineInstances) Screenshot(ctx context.Context, name string, options *v1.ScreenshotOptions) ([]byte, error) {
	return nil, nil
}

func (c *fakeVirtualMachineInstances) PortForward(name string, port int, protocol string) (kvcorev1.StreamInterface, error) {
	return nil, nil
}

func (c *fakeVirtualMachineInstances) Pause(ctx context.Context, name string, pauseOptions *v1.PauseOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "pause", name, pauseOptions), nil)

	return err
}

func (c *fakeVirtualMachineInstances) Unpause(ctx context.Context, name string, unpauseOptions *v1.UnpauseOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "unpause", name, unpauseOptions), nil)

	return err
}

func (c *fakeVirtualMachineInstances) Freeze(ctx context.Context, name string, unfreezeTimeout time.Duration) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "freeze", name, struct{}{}), nil)

	return err
}

func (c *fakeVirtualMachineInstances) Unfreeze(ctx context.Context, name string) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "unfreeze", name, struct{}{}), nil)

	return err
}

func (c *fakeVirtualMachineInstances) Reset(ctx context.Context, name string) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "reset", name, struct{}{}), nil)

	return err
}

func (c *fakeVirtualMachineInstances) SoftReboot(ctx context.Context, name string) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "softreboot", name, struct{}{}), nil)

	return err
}

func (c *fakeVirtualMachineInstances) GuestOsInfo(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	_, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "guestosinfo", name), &v1.VirtualMachineInstanceGuestAgentInfo{})

	return v1.VirtualMachineInstanceGuestAgentInfo{}, err
}

func (c *fakeVirtualMachineInstances) UserList(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestOSUserList, error) {
	_, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "userlist", name), &v1.VirtualMachineInstanceGuestOSUserList{})

	return v1.VirtualMachineInstanceGuestOSUserList{}, err

}

func (c *fakeVirtualMachineInstances) FilesystemList(ctx context.Context, name string) (v1.VirtualMachineInstanceFileSystemList, error) {
	_, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "userlist", name), &v1.VirtualMachineInstanceFileSystemList{})

	return v1.VirtualMachineInstanceFileSystemList{}, err
}

func (c *fakeVirtualMachineInstances) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "addvolume", name, addVolumeOptions), nil)

	return err
}

func (c *fakeVirtualMachineInstances) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	_, err := c.Fake.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "removevolume", name, removeVolumeOptions), nil)

	return err
}

func (c *fakeVirtualMachineInstances) VSOCK(name string, options *v1.VSOCKOptions) (kvcorev1.StreamInterface, error) {
	return nil, nil
}

func (c *fakeVirtualMachineInstances) SEVFetchCertChain(ctx context.Context, name string) (v1.SEVPlatformInfo, error) {
	_, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "sev/fetchcertchain", name), &v1.SEVPlatformInfo{})

	return v1.SEVPlatformInfo{}, err
}

func (c *fakeVirtualMachineInstances) SEVQueryLaunchMeasurement(ctx context.Context, name string) (v1.SEVMeasurementInfo, error) {
	_, err := c.Fake.
		Invokes(testing.NewGetSubresourceAction(c.Resource(), c.Namespace(), "sev/querylaunchmeasurement", name), &v1.SEVMeasurementInfo{})

	return v1.SEVMeasurementInfo{}, err
}

func (c *fakeVirtualMachineInstances) SEVSetupSession(ctx context.Context, name string, sevSessionOptions *v1.SEVSessionOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "sev/setupsession", name, sevSessionOptions), nil)

	return err
}

func (c *fakeVirtualMachineInstances) SEVInjectLaunchSecret(ctx context.Context, name string, sevSecretOptions *v1.SEVSecretOptions) error {
	_, err := c.Fake.
		Invokes(fake2.NewPutSubresourceAction(c.Resource(), c.Namespace(), "sev/injectlaunchsecret", name, sevSecretOptions), nil)

	return err
}

func (c *fakeVirtualMachineInstances) ObjectGraph(ctx context.Context, name string, objectGraphOptions *v1.ObjectGraphOptions) (v1.ObjectGraphNode, error) {
	obj, err := c.Fake.
		Invokes(fake2.NewGetSubresourceAction(c.Resource(), c.Namespace(), "objectgraph", name, objectGraphOptions), nil)

	return *obj.(*v1.ObjectGraphNode), err
}
