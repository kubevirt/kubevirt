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

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type SubresourceAPIApp struct {
	VirtCli kubecli.KubevirtClient
}

func (app *SubresourceAPIApp) requestHandler(request *restful.Request, response *restful.Response, vmi *v1.VirtualMachineInstance, cmd []string) {

	podName, httpStatusCode, err := app.remoteExecInfo(vmi)
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
		httpCode, err := remoteExecHelper(podName, vmi.Namespace, cmd, inReader, outWriter)
		log.Log.Errorf("%v", err)
		httpResponseChan <- httpCode
	}()

	go func() {
		_, err := io.Copy(wsReadWriter, outReader)
		log.Log.Reason(err).Error("error ecountered reading from remote podExec stream")
		copyErr <- err
	}()

	go func() {
		_, err := io.Copy(inWriter, wsReadWriter)
		log.Log.Reason(err).Error("error ecountered reading from websocket stream")
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

	vmiName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	cmd := []string{"/usr/share/kubevirt/virt-launcher/sock-connector", fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-%s", namespace, vmiName, "vnc")}

	vmi, code, err := app.fetchVirtualMachineInstance(vmiName, namespace)
	if err != nil {
		log.Log.Reason(err).Error("Failed to gather remote exec info for subresource request.")
		response.WriteError(code, err)
		return
	}

	// If there are no graphics devices present, we can't proceed
	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == false {
		err := fmt.Errorf("No graphics devices are present.")
		log.Log.Reason(err).Error("Can't establish VNC connection.")
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	app.requestHandler(request, response, vmi, cmd)
}

func (app *SubresourceAPIApp) ConsoleRequestHandler(request *restful.Request, response *restful.Response) {
	vmiName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	cmd := []string{"/usr/share/kubevirt/virt-launcher/sock-connector", fmt.Sprintf("/var/run/kubevirt-private/%s/%s/virt-%s", namespace, vmiName, "serial0")}

	vmi, code, err := app.fetchVirtualMachineInstance(vmiName, namespace)
	if err != nil {
		log.Log.Reason(err).Error("Failed to gather remote exec info for subresource request.")
		response.WriteError(code, err)
		return
	}

	app.requestHandler(request, response, vmi, cmd)
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
		return "", goerror.New("connection failed. No VirtualMachineInstance pod is running")
	}
	return podList.Items[0].ObjectMeta.Name, nil
}

func (app *SubresourceAPIApp) fetchVirtualMachineInstance(name string, namespace string) (*v1.VirtualMachineInstance, int, error) {

	vmi, err := app.VirtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, http.StatusNotFound, goerror.New(fmt.Sprintf("VirtualMachineInstance %s in namespace %s not found.", name, namespace))
		}
		return nil, http.StatusInternalServerError, err
	}
	return vmi, 0, nil
}

func (app *SubresourceAPIApp) remoteExecInfo(vmi *v1.VirtualMachineInstance) (string, int, error) {
	podName := ""

	if vmi.IsRunning() == false {
		return podName, http.StatusBadRequest, goerror.New(fmt.Sprintf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s", vmi.Status.Phase, v1.Running))
	}

	podName, err := app.findPod(vmi.Namespace, vmi.Name)
	if err != nil {
		return podName, http.StatusBadRequest, fmt.Errorf("unable to find matching pod for remote execution: %v", err)
	}

	return podName, http.StatusOK, nil
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
