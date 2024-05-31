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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package kubecli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
	"kubevirt.io/client-go/log"
)

const vmiSubresourceURL = "/apis/subresources.kubevirt.io/%s/namespaces/%s/virtualmachineinstances/%s/%s"

func (k *kubevirt) VirtualMachineInstance(namespace string) VirtualMachineInstanceInterface {
	return &vmis{
		VirtualMachineInstanceInterface: k.GeneratedKubeVirtClient().KubevirtV1().VirtualMachineInstances(namespace),
		restClient:                      k.restClient,
		config:                          k.config,
		clientSet:                       k.Clientset,
		namespace:                       namespace,
		resource:                        "virtualmachineinstances",
	}
}

type vmis struct {
	kvcorev1.VirtualMachineInstanceInterface
	restClient *rest.RESTClient
	config     *rest.Config
	clientSet  *kubernetes.Clientset
	namespace  string
	resource   string
	master     string
	kubeconfig string
}

func (v *vmis) USBRedir(name string) (kvcorev1.StreamInterface, error) {
	return kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, "usbredir", url.Values{})
}

func (v *vmis) VNC(name string) (kvcorev1.StreamInterface, error) {
	return kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, "vnc", url.Values{})
}

func (v *vmis) PortForward(name string, port int, protocol string) (kvcorev1.StreamInterface, error) {
	return kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, buildPortForwardResourcePath(port, protocol), url.Values{})
}

func buildPortForwardResourcePath(port int, protocol string) string {
	resource := strings.Builder{}
	resource.WriteString("portforward/")
	resource.WriteString(strconv.Itoa(port))

	if len(protocol) > 0 {
		resource.WriteString("/")
		resource.WriteString(protocol)
	}

	return resource.String()
}

type connectionStruct struct {
	con kvcorev1.StreamInterface
	err error
}

func (v *vmis) SerialConsole(name string, options *kvcorev1.SerialConsoleOptions) (kvcorev1.StreamInterface, error) {

	if options != nil && options.ConnectionTimeout != 0 {
		timeoutChan := time.Tick(options.ConnectionTimeout)
		connectionChan := make(chan connectionStruct)

		go func() {
			for {

				select {
				case <-timeoutChan:
					connectionChan <- connectionStruct{
						con: nil,
						err: fmt.Errorf("Timeout trying to connect to the virtual machine instance"),
					}
					return
				default:
				}

				con, err := kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, "console", url.Values{})
				if err != nil {
					asyncSubresourceError, ok := err.(*kvcorev1.AsyncSubresourceError)
					// return if response status code does not equal to 400
					if !ok || asyncSubresourceError.GetStatusCode() != http.StatusBadRequest {
						connectionChan <- connectionStruct{con: nil, err: err}
						return
					}

					time.Sleep(1 * time.Second)
					continue
				}

				connectionChan <- connectionStruct{con: con, err: nil}
				return
			}
		}()
		conStruct := <-connectionChan
		return conStruct.con, conStruct.err
	} else {
		return kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, "console", url.Values{})
	}
}

func (v *vmis) Freeze(ctx context.Context, name string, unfreezeTimeout time.Duration) error {
	log.Log.Infof("Freeze VMI %s", name)
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "freeze")

	freezeUnfreezeTimeout := &v1.FreezeUnfreezeTimeout{
		UnfreezeTimeout: &metav1.Duration{
			Duration: unfreezeTimeout,
		},
	}

	JSON, err := json.Marshal(freezeUnfreezeTimeout)
	if err != nil {
		return err
	}

	return v.restClient.Put().AbsPath(uri).Body([]byte(JSON)).Do(ctx).Error()
}

func (v *vmis) Unfreeze(ctx context.Context, name string) error {
	log.Log.Infof("Unfreeze VMI %s", name)
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "unfreeze")
	return v.restClient.Put().AbsPath(uri).Do(ctx).Error()
}

func (v *vmis) SoftReboot(ctx context.Context, name string) error {
	log.Log.Infof("SoftReboot VMI")
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "softreboot")
	return v.restClient.Put().AbsPath(uri).Do(ctx).Error()
}

func (v *vmis) Pause(ctx context.Context, name string, pauseOptions *v1.PauseOptions) error {
	body, err := json.Marshal(pauseOptions)
	if err != nil {
		return fmt.Errorf("Cannot Marshal to json: %s", err)
	}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "pause")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}

func (v *vmis) Unpause(ctx context.Context, name string, unpauseOptions *v1.UnpauseOptions) error {
	body, err := json.Marshal(unpauseOptions)
	if err != nil {
		return fmt.Errorf("Cannot Marshal to json: %s", err)
	}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "unpause")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}

func (v *vmis) Get(ctx context.Context, name string, options metav1.GetOptions) (vmi *v1.VirtualMachineInstance, err error) {
	vmi, err = v.VirtualMachineInstanceInterface.Get(ctx, name, options)
	vmi.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	return
}

func (v *vmis) List(ctx context.Context, options metav1.ListOptions) (vmiList *v1.VirtualMachineInstanceList, err error) {
	vmiList, err = v.VirtualMachineInstanceInterface.List(ctx, options)
	for i := range vmiList.Items {
		vmiList.Items[i].SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	}
	return
}

func (v *vmis) Create(ctx context.Context, vmi *v1.VirtualMachineInstance, opts metav1.CreateOptions) (result *v1.VirtualMachineInstance, err error) {
	result, err = v.VirtualMachineInstanceInterface.Create(ctx, vmi, opts)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	return
}

func (v *vmis) Update(ctx context.Context, vmi *v1.VirtualMachineInstance, opts metav1.UpdateOptions) (result *v1.VirtualMachineInstance, err error) {
	result, err = v.VirtualMachineInstanceInterface.Update(ctx, vmi, opts)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	return
}

func (v *vmis) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return v.VirtualMachineInstanceInterface.Delete(ctx, name, options)
}

func (v *vmis) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, patchOptions metav1.PatchOptions, subresources ...string) (result *v1.VirtualMachineInstance, err error) {
	return v.VirtualMachineInstanceInterface.Patch(ctx, name, pt, data, patchOptions, subresources...)
}

func (v *vmis) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return v.VirtualMachineInstanceInterface.Watch(ctx, opts)
}

func (v *vmis) GuestOsInfo(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "guestosinfo")

	// WORKAROUND:
	// When doing v.restClient.Get().RequestURI(uri).Do(ctx).Into(guestInfo)
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
	res := v.restClient.Get().AbsPath(uri).Do(ctx)
	rawInfo, err := res.Raw()
	if err != nil {
		log.Log.Errorf("Cannot retrieve GuestOSInfo: %s", err.Error())
		return guestInfo, err
	}

	err = json.Unmarshal(rawInfo, &guestInfo)
	if err != nil {
		log.Log.Errorf("Cannot unmarshal GuestOSInfo response: %s", err.Error())
	}

	return guestInfo, err
}

func (v *vmis) UserList(ctx context.Context, name string) (v1.VirtualMachineInstanceGuestOSUserList, error) {
	userList := v1.VirtualMachineInstanceGuestOSUserList{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "userlist")
	err := v.restClient.Get().AbsPath(uri).Do(ctx).Into(&userList)
	return userList, err
}

func (v *vmis) FilesystemList(ctx context.Context, name string) (v1.VirtualMachineInstanceFileSystemList, error) {
	fsList := v1.VirtualMachineInstanceFileSystemList{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "filesystemlist")
	err := v.restClient.Get().AbsPath(uri).Do(ctx).Into(&fsList)
	return fsList, err
}

func (v *vmis) Screenshot(ctx context.Context, name string, screenshotOptions *v1.ScreenshotOptions) ([]byte, error) {
	moveCursor := "false"
	if screenshotOptions.MoveCursor == true {
		moveCursor = "true"
	}

	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "vnc/screenshot")
	res := v.restClient.Get().AbsPath(uri).Param("moveCursor", moveCursor).Do(ctx)
	raw, err := res.Raw()
	if err != nil {
		return nil, res.Error()
	}
	return raw, nil
}

func (v *vmis) AddVolume(ctx context.Context, name string, addVolumeOptions *v1.AddVolumeOptions) error {
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "addvolume")

	JSON, err := json.Marshal(addVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().AbsPath(uri).Body([]byte(JSON)).Do(ctx).Error()
}

func (v *vmis) RemoveVolume(ctx context.Context, name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "removevolume")

	JSON, err := json.Marshal(removeVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().AbsPath(uri).Body([]byte(JSON)).Do(ctx).Error()
}

func (v *vmis) VSOCK(name string, options *v1.VSOCKOptions) (kvcorev1.StreamInterface, error) {
	if options == nil || options.TargetPort == 0 {
		return nil, fmt.Errorf("target port is required but not provided")
	}
	queryParams := url.Values{}
	queryParams.Add("port", strconv.FormatUint(uint64(options.TargetPort), 10))
	useTLS := true
	if options.UseTLS != nil {
		useTLS = *options.UseTLS
	}
	queryParams.Add("tls", strconv.FormatBool(useTLS))
	return kvcorev1.AsyncSubresourceHelper(v.config, v.resource, v.namespace, name, "vsock", queryParams)
}

func (v *vmis) SEVFetchCertChain(ctx context.Context, name string) (v1.SEVPlatformInfo, error) {
	sevPlatformInfo := v1.SEVPlatformInfo{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "sev/fetchcertchain")
	err := v.restClient.Get().AbsPath(uri).Do(ctx).Into(&sevPlatformInfo)
	return sevPlatformInfo, err
}

func (v *vmis) SEVQueryLaunchMeasurement(ctx context.Context, name string) (v1.SEVMeasurementInfo, error) {
	sevMeasurementInfo := v1.SEVMeasurementInfo{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "sev/querylaunchmeasurement")
	err := v.restClient.Get().AbsPath(uri).Do(ctx).Into(&sevMeasurementInfo)
	return sevMeasurementInfo, err
}

func (v *vmis) SEVSetupSession(ctx context.Context, name string, sevSessionOptions *v1.SEVSessionOptions) error {
	body, err := json.Marshal(sevSessionOptions)
	if err != nil {
		return fmt.Errorf("Cannot Marshal to json: %s", err)
	}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "sev/setupsession")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}

func (v *vmis) SEVInjectLaunchSecret(ctx context.Context, name string, sevSecretOptions *v1.SEVSecretOptions) error {
	body, err := json.Marshal(sevSecretOptions)
	if err != nil {
		return fmt.Errorf("Cannot Marshal to json: %s", err)
	}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "sev/injectlaunchsecret")
	return v.restClient.Put().AbsPath(uri).Body(body).Do(ctx).Error()
}
