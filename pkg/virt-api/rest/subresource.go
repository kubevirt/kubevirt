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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

import (
	goerror "errors"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"

	"net"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type SubresourceAPIApp struct {
	VirtCli kubecli.KubevirtClient
}

func (app *SubresourceAPIApp) requestHandler(request *restful.Request, response *restful.Response, cmd []string) {

	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	podName, httpStatusCode, err := app.remoteExecInfo(vmName, namespace)
	if err != nil {
		log.Log.Reason(err).Error("Failed to gather remote exec info for subresource request.")
		response.WriteError(httpStatusCode, err)
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  kubecli.WebsocketMessageBufferSize,
		WriteBufferSize: kubecli.WebsocketMessageBufferSize,
	}

	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Log.Reason(err).Error("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer clientSocket.Close()

	log.Log.Infof("Websocket connection upgraded")
	wsReadWriter := &kubecli.BinaryReadWriter{Conn: clientSocket}

	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()

	httpResponseChan := make(chan int)
	copyErr := make(chan error)
	go func() {
		httpCode, err := remoteExecHelper(podName, namespace, cmd, inReader, outWriter)
		log.Log.Errorf("%v", err)
		httpResponseChan <- httpCode
	}()

	go func() {
		_, err := io.Copy(wsReadWriter, outReader)
		if err != nil {
			log.Log.Reason(err).Error("error ecountered reading from remote podExec stream")
		}
		copyErr <- err
	}()

	go func() {
		_, err := io.Copy(inWriter, wsReadWriter)
		if err != nil {
			log.Log.Reason(err).Error("error ecountered reading from websocket stream")
		}
		copyErr <- err
	}()

	httpResponseCode := http.StatusOK
	select {
	case httpResponseCode = <-httpResponseChan:
	case err := <-copyErr:
		if err != nil {
			log.Log.Reason(err).Error("Error in websocket proxy")
			httpResponseCode = http.StatusInternalServerError
		}
	}
	response.WriteHeader(httpResponseCode)
}

func (app *SubresourceAPIApp) VNCRequestHandler(request *restful.Request, response *restful.Response) {

	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	cmd := []string{"/sock-connector", fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-%s", namespace, vmName, "vnc")}
	app.requestHandler(request, response, cmd)
}

func (app *SubresourceAPIApp) ConsoleRequestHandler(request *restful.Request, response *restful.Response) {
	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	cmd := []string{"/sock-connector", fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-%s", namespace, vmName, "serial0")}

	app.requestHandler(request, response, cmd)
}

func (app *SubresourceAPIApp) SSHRequestHandler(request *restful.Request, response *restful.Response) {
	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	vm, httpResponseCode, err := app.isVirtualMachineReady(vmName, namespace)
	if err != nil {
		response.WriteError(httpResponseCode, err)
		return
	}

	ip := vm.Status.Interfaces[0].IP
	tcpConn, err := net.Dial("tcp", ip+":22")
	if err != nil {
		response.WriteError(http.StatusServiceUnavailable, fmt.Errorf("could not open tcp connection: %v", err))
		return
	}

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  kubecli.WebsocketMessageBufferSize,
		WriteBufferSize: kubecli.WebsocketMessageBufferSize,
	}

	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Log.Reason(err).Error("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer clientSocket.Close()

	log.Log.Infof("Websocket connection upgraded")
	copyErr := make(chan error)

	go func() {
		_, err := io.Copy(clientSocket.UnderlyingConn(), tcpConn)
		if err != nil {
			log.Log.Reason(err).Error("error ecountered reading from remote podExec stream")
		}
		copyErr <- err
	}()

	go func() {
		_, err := io.Copy(tcpConn, clientSocket.UnderlyingConn())
		if err != nil {
			log.Log.Reason(err).Error("error ecountered reading from websocket stream")
		}
		copyErr <- err
	}()

	select {
	case err := <-copyErr:
		if err != nil {
			log.Log.Reason(err).Error("Error in websocket proxy")
			httpResponseCode = http.StatusInternalServerError
		}
	}
	response.WriteHeader(httpResponseCode)
}

func (app *SubresourceAPIApp) findPod(namespace string, name string) (string, error) {
	fieldSelector := fields.ParseSelectorOrDie("status.phase==" + string(k8sv1.PodRunning))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel+"=virt-launcher,"+v1.DomainLabel+" in (%s)", name))
	if err != nil {
		return "", err
	}
	selector := k8smetav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}

	podList, err := app.VirtCli.CoreV1().Pods(namespace).List(selector)
	if err != nil {
		return "", err
	}

	if len(podList.Items) == 0 {
		return "", goerror.New("connection failed. No VM pod is running")
	}
	return podList.Items[0].ObjectMeta.Name, nil
}

func (app *SubresourceAPIApp) remoteExecInfo(name string, namespace string) (string, int, error) {
	podName := ""

	_, errorCode, err := app.isVirtualMachineReady(name, namespace)
	if err != nil {
		return "", errorCode, err
	}

	podName, err = app.findPod(namespace, name)
	if err != nil {
		return podName, http.StatusBadRequest, fmt.Errorf("unable to find matching pod for remote execution: %v", err)
	}

	return podName, http.StatusOK, nil
}

func (app *SubresourceAPIApp) isVirtualMachineReady(name string, namespace string) (vm *v1.VirtualMachine, httpError int, err error) {
	vm, err = app.VirtCli.VM(namespace).Get(name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return vm, http.StatusNotFound, goerror.New(fmt.Sprintf("VM %s in namespace %s not found.", name, namespace))
		}
		return vm, http.StatusInternalServerError, err
	}

	if vm.IsRunning() == false {
		return vm, http.StatusBadRequest, goerror.New(fmt.Sprintf("Unable to connect to VM because phase is %s instead of %s", vm.Status.Phase, v1.Running))
	}

	return vm, http.StatusOK, nil
}

func remoteExecHelper(podName string, namespace string, cmd []string, in io.Reader, out io.Writer) (int, error) {

	config, err := kubecli.GetConfig()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to build api config for remote execution: %v", err)
	}

	gv := k8sv1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/api"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	restClient, err := restclient.RESTClientFor(config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to create restClient for remote execution: %v", err)
	}
	containerName := "compute"
	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName)

	req = req.VersionedParams(&k8sv1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	// execute request
	method := "POST"
	url := req.URL()
	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("remote execution failed: %v", err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             in,
		Stdout:            out,
		Stderr:            out,
		Tty:               false,
		TerminalSizeQueue: nil,
	})

	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("connection failed: %v", err)
	}
	return http.StatusOK, nil
}
