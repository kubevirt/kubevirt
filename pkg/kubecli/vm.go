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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

const (
	WebsocketMessageBufferSize = 10240
)

func (k *kubevirt) VM(namespace string) VMInterface {
	return &vms{
		restClient: k.restClient,
		config:     k.config,
		clientSet:  k.Clientset,
		namespace:  namespace,
		resource:   "virtualmachines",
	}
}

type vms struct {
	restClient *rest.RESTClient
	config     *rest.Config
	clientSet  *kubernetes.Clientset
	namespace  string
	resource   string
	master     string
	kubeconfig string
}

type BinaryReadWriter struct {
	Conn *websocket.Conn
}

func (s *BinaryReadWriter) Write(p []byte) (int, error) {
	wsFrameHeaderSize := 2 + 8 + 4 // Fixed header + length + mask (RFC 6455)
	// our websocket package has an issue where it truncates messages
	// when the message+header is greater than the buffer size we allocate.
	// because of this, we have to chunk messages
	chunkSize := WebsocketMessageBufferSize - wsFrameHeaderSize
	bytesWritten := 0

	for i := 0; i < len(p); i += chunkSize {
		w, err := s.Conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			return bytesWritten, s.err(err)
		}
		defer w.Close()

		end := i + chunkSize
		if end > len(p) {
			end = len(p)
		}
		n, err := w.Write(p[i:end])
		if err != nil {
			return bytesWritten, err
		}

		bytesWritten = n + bytesWritten
	}
	return bytesWritten, nil

}

func (s *BinaryReadWriter) Read(p []byte) (int, error) {
	for {
		msgType, r, err := s.Conn.NextReader()
		if err != nil {
			return 0, s.err(err)
		}

		switch msgType {
		case websocket.BinaryMessage:
			n, err := r.Read(p)
			return n, s.err(err)

		case websocket.CloseMessage:
			return 0, io.EOF
		}
	}
}

func (s *BinaryReadWriter) err(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*websocket.CloseError); ok {
		if e.Code == websocket.CloseNormalClosure {
			return io.EOF
		}
	}
	return err
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

type wsCallbackObj struct {
	in  io.Reader
	out io.Writer
}

func (obj *wsCallbackObj) WebsocketCallback(ws *websocket.Conn, resp *http.Response, err error) error {

	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			return fmt.Errorf("Can't connect to websocket (%d): %s\n", resp.StatusCode, buf.String())
		}
		return fmt.Errorf("Can't connect to websocket: %s\n", err.Error())
	}

	wsReadWriter := &BinaryReadWriter{Conn: ws}

	copyErr := make(chan error)

	go func() {
		_, err := io.Copy(wsReadWriter, obj.in)
		copyErr <- err
	}()

	go func() {
		_, err := io.Copy(obj.out, wsReadWriter)
		copyErr <- err
	}()

	err = <-copyErr
	return err
}

func roundTripperFromConfig(config *rest.Config, in io.Reader, out io.Writer) (http.RoundTripper, error) {

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
	}

	obj := &wsCallbackObj{
		in:  in,
		out: out,
	}
	// Create a roundtripper which will pass in the final underlying websocket connection to a callback
	rt := &WebsocketRoundTripper{
		Do:     obj.WebsocketCallback,
		Dialer: dialer,
	}

	// Make sure we inherit all relevant security headers
	return rest.HTTPWrappersForConfig(config, rt)
}

func RequestFromConfig(config *rest.Config, vm string, namespace string, resource string) (*http.Request, error) {

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

	u.Path = fmt.Sprintf("/apis/subresources.kubevirt.io/v1alpha1/namespaces/%s/virtualmachines/%s/%s", namespace, vm, resource)
	req := &http.Request{
		Method: http.MethodGet,
		URL:    u,
	}

	return req, nil
}

func (v *vms) VNC(name string, in io.Reader, out io.Writer) error {
	return v.subresourceHelper(name, "vnc", in, out)
}
func (v *vms) SerialConsole(name string, in io.Reader, out io.Writer) error {
	return v.subresourceHelper(name, "console", in, out)
}

func (v *vms) subresourceHelper(name string, resource string, in io.Reader, out io.Writer) error {

	// Create a round tripper with all necessary kubernetes security details
	wrappedRoundTripper, err := roundTripperFromConfig(v.config, in, out)
	if err != nil {
		return fmt.Errorf("unable to create round tripper for remote execution: %v", err)
	}

	// Create a request out of config and the query parameters
	req, err := RequestFromConfig(v.config, name, v.namespace, resource)
	if err != nil {
		return fmt.Errorf("unable to create request for remote execution: %v", err)
	}

	// Send the request and let the callback do its work
	response, err := wrappedRoundTripper.RoundTrip(req)

	if err != nil {
		return err
	}

	if response != nil {
		switch response.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusNotFound:
			return fmt.Errorf("Virtual Machine not found.")
		case http.StatusInternalServerError:
			return fmt.Errorf("Websocket failed due to internal server error.")
		default:
			return fmt.Errorf("Websocket failed with http status: %s", response.Status)
		}
	} else {
		return fmt.Errorf("no response received")
	}
}

func (v *vms) Get(name string, options k8smetav1.GetOptions) (vm *v1.VirtualMachine, err error) {
	vm = &v1.VirtualMachine{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vm)
	vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vms) List(options k8smetav1.ListOptions) (vmList *v1.VirtualMachineList, err error) {
	vmList = &v1.VirtualMachineList{}
	err = v.restClient.Get().
		Resource(v.resource).
		Namespace(v.namespace).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(vmList)
	for _, vm := range vmList.Items {
		vm.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	}

	return
}

func (v *vms) Create(vm *v1.VirtualMachine) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Post().
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vms) Update(vm *v1.VirtualMachine) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Put().
		Name(vm.ObjectMeta.Name).
		Namespace(v.namespace).
		Resource(v.resource).
		Body(vm).
		Do().
		Into(result)
	result.SetGroupVersionKind(v1.VirtualMachineGroupVersionKind)
	return
}

func (v *vms) Delete(name string, options *k8smetav1.DeleteOptions) error {
	return v.restClient.Delete().
		Namespace(v.namespace).
		Resource(v.resource).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (v *vms) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.VirtualMachine, err error) {
	result = &v1.VirtualMachine{}
	err = v.restClient.Patch(pt).
		Namespace(v.namespace).
		Resource(v.resource).
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
