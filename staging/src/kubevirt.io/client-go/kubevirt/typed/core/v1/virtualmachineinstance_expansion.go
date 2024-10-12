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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const vmiSubresourceURL = "/apis/subresources.kubevirt.io/%s"

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
	//  requires clientConfig
	return nil, fmt.Errorf("SerialConsole is not implemented yet in generated client")
}

func (c *virtualMachineInstances) USBRedir(vmiName string) (StreamInterface, error) {
	// TODO not implemented yet
	//  requires clientConfig
	return nil, fmt.Errorf("USBRedir is not implemented yet in generated client")
}

func (c *virtualMachineInstances) VNC(name string) (StreamInterface, error) {
	// TODO not implemented yet
	//  requires clientConfig
	return nil, fmt.Errorf("VNC is not implemented yet in generated client")
}

func (c *virtualMachineInstances) Screenshot(ctx context.Context, name string, options *v1.ScreenshotOptions) ([]byte, error) {
	moveCursor := "false"
	if options.MoveCursor == true {
		moveCursor = "true"
	}
	res := c.GetClient().Get().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("vnc", "screenshot").
		Param("moveCursor", moveCursor).
		Do(ctx)

	raw, err := res.Raw()
	if err != nil {
		return nil, res.Error()
	}

	return raw, nil
}

func (c *virtualMachineInstances) PortForward(name string, port int, protocol string) (StreamInterface, error) {
	// TODO not implemented yet
	//  requires clientConfig
	return nil, fmt.Errorf("PortForward is not implemented yet in generated client")
}

func (c *virtualMachineInstances) Pause(ctx context.Context, name string, pauseOptions *v1.PauseOptions) error {
	body, err := json.Marshal(pauseOptions)
	if err != nil {
		return err
	}

	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("pause").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) Unpause(ctx context.Context, name string, unpauseOptions *v1.UnpauseOptions) error {
	body, err := json.Marshal(unpauseOptions)
	if err != nil {
		return err
	}

	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("unpause").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) Freeze(ctx context.Context, name string, unfreezeTimeout time.Duration) error {
	log.Log.Infof("Freeze VMI %s", name)
	freezeUnfreezeTimeout := &v1.FreezeUnfreezeTimeout{
		UnfreezeTimeout: &metav1.Duration{
			Duration: unfreezeTimeout,
		},
	}

	body, err := json.Marshal(freezeUnfreezeTimeout)
	if err != nil {
		return err
	}

	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("freeze").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) Unfreeze(ctx context.Context, name string) error {
	log.Log.Infof("Unfreeze VMI %s", name)
	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("unfreeze").
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) SoftReboot(ctx context.Context, name string) error {
	log.Log.Infof("SoftReboot VMI")
	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("softreboot").
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) GuestOsInfo(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{}
	// WORKAROUND:
	// When doing c.GetClient().Get().RequestURI(uri).Do(ctx).Into(guestInfo)
	// k8s client-go requires the object to have metav1.ObjectMeta inlined and deepcopy generated
	// without deepcopy the Into does not work.
	// With metav1.ObjectMeta added the openapi validation fails on pkg/virt-api/api.go:310
	// When returning object the openapi schema validation fails on invalid type field for
	// metav1.ObjectMeta.CreationTimestamp of type time (the schema validation fails, not the object validation).
	// In our schema we implemented workaround to have multiple types for this field (null, string), which is causing issues
	// with deserialization.
	// The issue popped up for this code since this is the first time anything is returned.
	//
	// The issue is present because KubeVirt have to support multiple k8s version. In newer k8s version (1.17+)
	// this issue should be solved.
	// This workaround can go away once the least supported k8s version is the working one.
	// The issue has been described in: https://github.com/kubevirt/kubevirt/issues/3059
	// Will be replaced by:
	// 	err := c.GetClient().Get().
	//		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
	//		Namespace(c.GetNamespace()).
	//		Resource("virtualmachineinstances").
	//		Name(name).
	//		SubResource("guestosinfo").
	//		Do(ctx).
	//		Into(&guestInfo)
	res := c.GetClient().Get().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("guestosinfo").
		Do(ctx)
	rawInfo, err := res.Raw()
	if err != nil {
		log.Log.Errorf("cannot retrieve GuestOSInfo: %s", err.Error())
		return guestInfo, err
	}

	err = json.Unmarshal(rawInfo, &guestInfo)
	if err != nil {
		log.Log.Errorf("cannot unmarshal GuestOSInfo response: %s", err.Error())
	}

	return guestInfo, err
}

func (c *virtualMachineInstances) UserList(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestOSUserList, error) {
	userList := v1.VirtualMachineInstanceGuestOSUserList{}
	err := c.GetClient().Get().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("userlist").
		Do(ctx).
		Into(&userList)
	return userList, err
}

func (c *virtualMachineInstances) FilesystemList(ctx context.Context, name string) (v1.VirtualMachineInstanceFileSystemList, error) {
	fsList := v1.VirtualMachineInstanceFileSystemList{}
	err := c.GetClient().Get().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("filesystemlist").
		Do(ctx).
		Into(&fsList)

	return fsList, err
}

func (c *virtualMachineInstances) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	body, err := json.Marshal(addVolumeOptions)
	if err != nil {
		return err
	}

	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("addvolume").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	body, err := json.Marshal(removeVolumeOptions)
	if err != nil {
		return err
	}

	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("removevolume").
		Body(body).
		Do(ctx).
		Error()
}

func (c *virtualMachineInstances) VSOCK(name string, options *v1.VSOCKOptions) (StreamInterface, error) {
	// TODO not implemented yet
	//  requires clientConfig
	return nil, fmt.Errorf("VSOCK is not implemented yet in generated client")
}

func (c *virtualMachineInstances) SEVFetchCertChain(ctx context.Context, name string) (v1.SEVPlatformInfo, error) {
	sevPlatformInfo := v1.SEVPlatformInfo{}
	err := c.GetClient().Get().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("sev", "fetchcertchain").
		Do(context.Background()).
		Into(&sevPlatformInfo)

	return sevPlatformInfo, err
}

func (c *virtualMachineInstances) SEVQueryLaunchMeasurement(ctx context.Context, name string) (v1.SEVMeasurementInfo, error) {
	sevMeasurementInfo := v1.SEVMeasurementInfo{}
	err := c.GetClient().Get().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("sev", "querylaunchmeasurement").
		Do(context.Background()).
		Into(&sevMeasurementInfo)

	return sevMeasurementInfo, err
}

func (c *virtualMachineInstances) SEVSetupSession(ctx context.Context, name string, sevSessionOptions *v1.SEVSessionOptions) error {
	body, err := json.Marshal(sevSessionOptions)
	if err != nil {
		return fmt.Errorf("cannot Marshal to json: %s", err)
	}

	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("sev", "setupsession").
		Body(body).
		Do(context.Background()).
		Error()
}

func (c *virtualMachineInstances) SEVInjectLaunchSecret(ctx context.Context, name string, sevSecretOptions *v1.SEVSecretOptions) error {
	body, err := json.Marshal(sevSecretOptions)
	if err != nil {
		return fmt.Errorf("cannot Marshal to json: %s", err)
	}
	return c.GetClient().Put().
		AbsPath(fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion)).
		Namespace(c.GetNamespace()).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource("sev", "injectlaunchsecret").
		Body(body).
		Do(context.Background()).
		Error()
}
