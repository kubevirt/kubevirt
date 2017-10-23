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

package rest

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	k8sv1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Console struct {
	virtClient      kubecli.KubevirtClient
	k8sClient       k8scorev1.CoreV1Interface
	VirtHandlerPort string
}

func NewConsoleResource(virtClient kubecli.KubevirtClient, k8sClient k8scorev1.CoreV1Interface) *Console {
	return &Console{virtClient: virtClient, k8sClient: k8sClient}
}

func (t *Console) Console(request *restful.Request, response *restful.Response) {
	console := request.QueryParameter("console")
	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, err := t.virtClient.VM(namespace).Get(vmName, k8sv1meta.GetOptions{})
	if errors.IsNotFound(err) {
		log.Log.V(3).Infof("VM '%s' does not exist", vmName)
		response.WriteError(http.StatusNotFound, fmt.Errorf("VM does not exist"))
		return
	}
	if err != nil {
		log.Log.Reason(err).Errorf("Error fetching VM '%s'", vmName)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	logger := log.Log.Object(vm)

	if !vm.IsRunning() {
		logger.V(3).Reason(err).Info("VM is not running")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("VM is not running"))
		return
	}

	virtHandlerCon := kubecli.NewVirtHandlerClient(t.virtClient).ForNode(vm.Status.NodeName)
	uri, err := virtHandlerCon.ConsoleURI(vm)
	if err != nil {
		msg := fmt.Sprintf("Looking up the connection details for virt-handler on node %s failed", vm.Status.NodeName)
		logger.Reason(err).Error(msg)
		response.WriteError(http.StatusInternalServerError, fmt.Errorf(msg))
		return
	}

	if t.VirtHandlerPort != "" {
		uri.Hostname()
		uri.Host = uri.Hostname() + ":" + t.VirtHandlerPort
	}
	uri.Scheme = "ws"
	if console != "" {
		uri.RawQuery = "console=" + console
	}
	handlerSocket, resp, err := websocket.DefaultDialer.Dial(uri.String(), nil)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err := fmt.Errorf("%s", buf.String())
			logger.With("statusCode", resp.StatusCode).Reason(err).Error("Failed to connect to virt-handler")
			response.WriteError(resp.StatusCode, err)
		} else {
			logger.Reason(err).Error("Failed to connect to virt-handler")
			response.WriteError(http.StatusInternalServerError, err)
		}
		return
	}
	defer handlerSocket.Close()

	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		logger.Reason(err).Error("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer clientSocket.Close()

	errorChan := make(chan error)

	go func() {
		_, err := io.Copy(clientSocket.UnderlyingConn(), handlerSocket.UnderlyingConn())
		errorChan <- err
	}()

	go func() {
		_, err := io.Copy(handlerSocket.UnderlyingConn(), clientSocket.UnderlyingConn())
		errorChan <- err
	}()

	err = <-errorChan
	if err != nil {
		logger.Reason(err).Error("Proxied Web Socket connection failed")
	}
	response.WriteHeader(http.StatusOK)
}
