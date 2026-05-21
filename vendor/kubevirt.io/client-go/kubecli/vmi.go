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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
)

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
