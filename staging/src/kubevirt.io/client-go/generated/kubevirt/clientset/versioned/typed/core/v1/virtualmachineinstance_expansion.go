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
	"time"

	v1 "kubevirt.io/api/core/v1"
)

type SerialConsoleOptions struct {
	ConnectionTimeout time.Duration
}

type VirtualMachineInstanceExpansion interface {
	SerialConsole(name string, options *SerialConsoleOptions) (StreamInterface, error)
	USBRedir(vmiName string) (StreamInterface, error)
	VNC(name string) (StreamInterface, error)
	Screenshot(ctx context.Context, name string, options *v1.ScreenshotOptions) ([]byte, error)
	PortForward(name string, port int, protocol string) (StreamInterface, error)
	Pause(ctx context.Context, name string, pauseOptions *v1.PauseOptions) error
	Unpause(ctx context.Context, name string, unpauseOptions *v1.UnpauseOptions) error
	Freeze(ctx context.Context, name string, unfreezeTimeout time.Duration) error
	Unfreeze(ctx context.Context, name string) error
	SoftReboot(ctx context.Context, name string) error
	GuestOsInfo(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestAgentInfo, error)
	UserList(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestOSUserList, error)
	FilesystemList(ctx context.Context, name string) (v1.VirtualMachineInstanceFileSystemList, error)
	AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error
	RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error
	VSOCK(name string, options *v1.VSOCKOptions) (StreamInterface, error)
	SEVFetchCertChain(ctx context.Context, name string) (v1.SEVPlatformInfo, error)
	SEVQueryLaunchMeasurement(ctx context.Context, name string) (v1.SEVMeasurementInfo, error)
	SEVSetupSession(ctx context.Context, name string, sevSessionOptions *v1.SEVSessionOptions) error
	SEVInjectLaunchSecret(ctx context.Context, name string, sevSecretOptions *v1.SEVSecretOptions) error
}

func (c *virtualMachineInstances) SerialConsole(name string, options *SerialConsoleOptions) (StreamInterface, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachineInstances) USBRedir(vmiName string) (StreamInterface, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachineInstances) VNC(name string) (StreamInterface, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachineInstances) Screenshot(ctx context.Context, name string, options *v1.ScreenshotOptions) ([]byte, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachineInstances) PortForward(name string, port int, protocol string) (StreamInterface, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachineInstances) Pause(ctx context.Context, name string, pauseOptions *v1.PauseOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) Unpause(ctx context.Context, name string, unpauseOptions *v1.UnpauseOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) Freeze(ctx context.Context, name string, unfreezeTimeout time.Duration) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) Unfreeze(ctx context.Context, name string) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) SoftReboot(ctx context.Context, name string) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) GuestOsInfo(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	// TODO not implemented yet
	return v1.VirtualMachineInstanceGuestAgentInfo{}, nil
}

func (c *virtualMachineInstances) UserList(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestOSUserList, error) {
	// TODO not implemented yet
	return v1.VirtualMachineInstanceGuestOSUserList{}, nil
}

func (c *virtualMachineInstances) FilesystemList(ctx context.Context, name string) (v1.VirtualMachineInstanceFileSystemList, error) {
	// TODO not implemented yet
	return v1.VirtualMachineInstanceFileSystemList{}, nil
}

func (c *virtualMachineInstances) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) VSOCK(name string, options *v1.VSOCKOptions) (StreamInterface, error) {
	// TODO not implemented yet
	return nil, nil
}

func (c *virtualMachineInstances) SEVFetchCertChain(ctx context.Context, name string) (v1.SEVPlatformInfo, error) {
	// TODO not implemented yet
	return v1.SEVPlatformInfo{}, nil
}

func (c *virtualMachineInstances) SEVQueryLaunchMeasurement(ctx context.Context, name string) (v1.SEVMeasurementInfo, error) {
	// TODO not implemented yet
	return v1.SEVMeasurementInfo{}, nil
}

func (c *virtualMachineInstances) SEVSetupSession(ctx context.Context, name string, sevSessionOptions *v1.SEVSessionOptions) error {
	// TODO not implemented yet
	return nil
}

func (c *virtualMachineInstances) SEVInjectLaunchSecret(ctx context.Context, name string, sevSecretOptions *v1.SEVSecretOptions) error {
	// TODO not implemented yet
	return nil
}
