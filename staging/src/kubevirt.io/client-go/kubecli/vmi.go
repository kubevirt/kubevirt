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
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/subresources"
)

const vmiSubresourceURL = "/apis/subresources.kubevirt.io/%s/namespaces/%s/virtualmachineinstances/%s/%s"

func (k *kubevirt) VirtualMachineInstance(namespace string) VirtualMachineInstanceInterface {
	return &vmis{
		restClient: k.restClient,
		config:     k.config,
		clientSet:  k.Clientset,
		namespace:  namespace,
		resource:   "virtualmachineinstances",
	}
}

type vmis struct {
	restClient *rest.RESTClient
	config     *rest.Config
	clientSet  *kubernetes.Clientset
	namespace  string
	resource   string
	master     string
	kubeconfig string
}

type RoundTripCallback func(conn *websocket.Conn, resp *http.Response, err error) error

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Do     RoundTripCallback
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	if err == nil {
		defer conn.Close()
	}
	return resp, d.Do(conn, resp, err)
}

type asyncWSRoundTripper struct {
	Done       chan struct{}
	Connection chan *websocket.Conn
}

func (aws *asyncWSRoundTripper) WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error) error {

	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			return enrichError(err, resp)
		}
		return fmt.Errorf("Can't connect to websocket: %s\n", err.Error())
	}
	aws.Connection <- ws

	// Keep the roundtripper open until we are done with the stream
	<-aws.Done
	return nil
}

func roundTripperFromConfig(config *rest.Config, callback RoundTripCallback) (http.RoundTripper, error) {

	// Configure TLS
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return nil, err
	}

	// Configure the websocket dialer
	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: tlsConfig,
		WriteBufferSize: WebsocketMessageBufferSize,
		ReadBufferSize:  WebsocketMessageBufferSize,
		Subprotocols:    []string{subresources.PlainStreamProtocolName},
	}

	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	rt := &WebsocketRoundTripper{
		Do:     callback,
		Dialer: dialer,
	}

	// Make sure we inherit all relevant security headers
	return rest.HTTPWrappersForConfig(config, rt)
}

func RequestFromConfig(config *rest.Config, vmi string, namespace string, resource string) (*http.Request, error) {

	u, err := url.Parse(config.Host)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return nil, fmt.Errorf("Unsupported Protocol %s", u.Scheme)
	}

	u.Path = fmt.Sprintf("/apis/subresources.kubevirt.io/%s/namespaces/%s/virtualmachineinstances/%s/%s", v1.ApiStorageVersion, namespace, vmi, resource)
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}

	return req, nil
}

type wsStreamer struct {
	conn *websocket.Conn
	done chan struct{}
}

func (ws *wsStreamer) streamDone() {
	close(ws.done)
}

func (ws *wsStreamer) Stream(options StreamOptions) error {
	copyErr := make(chan error, 1)

	go func() {
		_, err := CopyTo(ws.conn, options.In)
		copyErr <- err
	}()

	go func() {
		_, err := CopyFrom(options.Out, ws.conn)
		copyErr <- err
	}()

	defer ws.streamDone()
	return <-copyErr
}

func (v *vmis) VNC(name string) (StreamInterface, error) {
	return v.asyncSubresourceHelper(name, "vnc")
}

type connectionStruct struct {
	con StreamInterface
	err error
}

type SerialConsoleOptions struct {
	ConnectionTimeout time.Duration
}

func (v *vmis) SerialConsole(name string, options *SerialConsoleOptions) (StreamInterface, error) {

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

				con, err := v.asyncSubresourceHelper(name, "console")
				if err != nil {
					asyncSubresourceError, ok := err.(*AsyncSubresourceError)
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
		return v.asyncSubresourceHelper(name, "console")
	}
}

type AsyncSubresourceError struct {
	err        string
	StatusCode int
}

func (a *AsyncSubresourceError) Error() string {
	return a.err
}

func (a *AsyncSubresourceError) GetStatusCode() int {
	return a.StatusCode
}

func (v *vmis) asyncSubresourceHelper(name string, resource string) (StreamInterface, error) {

	done := make(chan struct{})

	aws := &asyncWSRoundTripper{
		Connection: make(chan *websocket.Conn),
		Done:       done,
	}
	// Create a round tripper with all necessary kubernetes security details
	wrappedRoundTripper, err := roundTripperFromConfig(v.config, aws.WebsocketCallback)
	if err != nil {
		return nil, fmt.Errorf("unable to create round tripper for remote execution: %v", err)
	}

	// Create a request out of config and the query parameters
	req, err := RequestFromConfig(v.config, name, v.namespace, resource)
	if err != nil {
		return nil, fmt.Errorf("unable to create request for remote execution: %v", err)
	}

	errChan := make(chan error, 1)

	go func() {
		// Send the request and let the callback do its work
		response, err := wrappedRoundTripper.RoundTrip(req)

		if err != nil {
			statusCode := 0
			if response != nil {
				statusCode = response.StatusCode
			}
			errChan <- &AsyncSubresourceError{err: err.Error(), StatusCode: statusCode}
			return
		}

		if response != nil {
			switch response.StatusCode {
			case http.StatusOK:
			case http.StatusNotFound:
				err = &AsyncSubresourceError{err: "Virtual Machine not found.", StatusCode: response.StatusCode}
			case http.StatusInternalServerError:
				err = &AsyncSubresourceError{err: "Websocket failed due to internal server error.", StatusCode: response.StatusCode}
			default:
				err = &AsyncSubresourceError{err: fmt.Sprintf("Websocket failed with http status: %s", response.Status), StatusCode: response.StatusCode}
			}
		} else {
			err = &AsyncSubresourceError{err: "no response received"}
		}
		errChan <- err
	}()

	select {
	case err = <-errChan:
		return nil, err
	case ws := <-aws.Connection:
		return &wsStreamer{
			conn: ws,
			done: done,
		}, nil
	}
}

func (v *vmis) Pause(name string) error {
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "pause")
	return v.restClient.Put().RequestURI(uri).Do(context.Background()).Error()
}

func (v *vmis) Unpause(name string) error {
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "unpause")
	return v.restClient.Put().RequestURI(uri).Do(context.Background()).Error()
}

func (v *vmis) Get(name string, options *k8smetav1.GetOptions) (vmi *v1.VirtualMachineInstance, err error) {
	vmi = &v1.VirtualMachineInstance{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(vmi)
	vmi.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	return
}

func (v *vmis) List(options *k8smetav1.ListOptions) (vmiList *v1.VirtualMachineInstanceList, err error) {
	vmiList = &v1.VirtualMachineInstanceList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(options, scheme.ParameterCodec).
		Do(context.Background()).
		Into(vmiList)
	for _, vmi := range vmiList.Items {
		vmi.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	}

	return
}

func (v *vmis) Create(vmi *v1.VirtualMachineInstance) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	return
}

func (v *vmis) Update(vmi *v1.VirtualMachineInstance) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = v.restClient.Put().
		Name(vmi.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vmi).
		Do(context.Background()).
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineInstanceGroupVersionKind)
	return
}

func (v *vmis) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do(context.Background()).
		Error()
}

func (v *vmis) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachineInstance, err error) {
	result = &v1.VirtualMachineInstance{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do(context.Background()).
		Into(result)
	return
}

// enrichError checks the response body for a k8s Status object and extracts the error from it.
// TODO the k8s http REST client has very sophisticated handling, investigate on how we can reuse it
func enrichError(httpErr error, resp *http.Response) error {
	if resp == nil {
		return httpErr
	}
	httpErr = fmt.Errorf("Can't connect to websocket (%d): %s\n", resp.StatusCode, httpErr)
	status := &k8smetav1.Status{}

	if resp.Header.Get("Content-Type") != "application/json" {
		return httpErr
	}
	// decode, but if the result is Status return that as an error instead.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return httpErr
	}
	err = json.Unmarshal(body, status)
	if err != nil {
		return err
	}
	if status.Kind == "Status" && status.APIVersion == "v1" {
		if status.Status != k8smetav1.StatusSuccess {
			return errors.FromObject(status)
		}
	}
	return httpErr
}

func (v *vmis) GuestOsInfo(name string) (v1.VirtualMachineInstanceGuestAgentInfo, error) {
	guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "guestosinfo")

	// WORKAROUND:
	// When doing v.restClient.Get().RequestURI(uri).Do(context.Background()).Into(guestInfo)
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
	res := v.restClient.Get().RequestURI(uri).Do(context.Background())
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

func (v *vmis) UserList(name string) (v1.VirtualMachineInstanceGuestOSUserList, error) {
	userList := v1.VirtualMachineInstanceGuestOSUserList{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "userlist")
	err := v.restClient.Get().RequestURI(uri).Do(context.Background()).Into(&userList)
	return userList, err
}

func (v *vmis) FilesystemList(name string) (v1.VirtualMachineInstanceFileSystemList, error) {
	fsList := v1.VirtualMachineInstanceFileSystemList{}
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "filesystemlist")
	err := v.restClient.Get().RequestURI(uri).Do(context.Background()).Into(&fsList)
	return fsList, err
}

func (v *vmis) AddVolume(name string, addVolumeOptions *v1.AddVolumeOptions) error {
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "addvolume")

	JSON, err := json.Marshal(addVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().RequestURI(uri).Body([]byte(JSON)).Do(context.Background()).Error()
}

func (v *vmis) RemoveVolume(name string, removeVolumeOptions *v1.RemoveVolumeOptions) error {
	uri := fmt.Sprintf(vmiSubresourceURL, v1.ApiStorageVersion, v.namespace, name, "removevolume")

	JSON, err := json.Marshal(removeVolumeOptions)

	if err != nil {
		return err
	}

	return v.restClient.Put().RequestURI(uri).Body([]byte(JSON)).Do(context.Background()).Error()
}
