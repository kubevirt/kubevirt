/*
 * This file is part of the kubevirt project
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
	"net/url"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	k8sv1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api"

	"k8s.io/apimachinery/pkg/api/errors"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
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
		logging.DefaultLogger().Info().V(3).Msgf("VM '%s' does not exist", vmName)
		response.WriteError(http.StatusNotFound, fmt.Errorf("VM does not exist"))
		return
	}
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msgf("Error fetching VM '%s'", vmName)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	log := logging.DefaultLogger().Object(vm)

	if !vm.IsRunning() {
		log.Info().V(3).Reason(err).Msg("VM is not running")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("VM is not running"))
		return
	}

	nodeName := vm.Status.NodeName

	// Get the pod name of virt-handler running on the master node to inspect its logs later on
	handlerNodeSelector := fields.ParseSelectorOrDie("spec.nodeName=" + nodeName)
	labelSelector, err := labels.Parse("daemon in (virt-handler)")
	if err != nil {
		log.Error().Reason(err).Msgf("Unable to parse label selector")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	pods, err := t.k8sClient.Pods(api.NamespaceDefault).List(
		k8sv1meta.ListOptions{
			FieldSelector: handlerNodeSelector.String(),
			LabelSelector: labelSelector.String()})
	if err != nil {
		log.Error().Reason(err).Msgf("Unable to find virt-handler POD")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	if len(pods.Items) != 1 {
		log.Error().Reason(err).Msgf("Expected one virt-handler POD but got %d", len(pods.Items))
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("Expected one virt-handler POD but got %d", len(pods.Items)))
		return
	}

	dstAddr := string(pods.Items[0].Status.PodIP)

	// FIXME, don't hardcode virt-handler port. virt-handler should register itself somehow
	port := "8185"
	if t.VirtHandlerPort != "" {
		port = t.VirtHandlerPort
	}

	u := url.URL{Scheme: "ws", Host: dstAddr + ":" + port, Path: fmt.Sprintf("/api/v1/namespaces/%s/vms/%s/console", namespace, vmName)}
	if console != "" {
		u.RawQuery = "console=" + console
	}
	handlerSocket, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		if resp != nil && resp.StatusCode != http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err := fmt.Errorf("%s", buf.String())
			log.Error().Reason(err).
				With("statusCode", resp.StatusCode).
				Msgf("Failed to connect to virt-handler")
			response.WriteError(resp.StatusCode, err)
		} else {
			log.Error().Reason(err).Msgf("Failed to connect to virt-handler")
			response.WriteError(http.StatusInternalServerError, err)
		}
		return
	}
	defer handlerSocket.Close()

	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Error().Reason(err).Msgf("Failed to upgrade client websocket connection")
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
		log.Error().Reason(err).Msgf("Proxied Web Socket connection failed")
	}
	response.WriteHeader(http.StatusOK)
}
