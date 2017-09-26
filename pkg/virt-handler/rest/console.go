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
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/libvirt/libvirt-go"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/errors"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Console struct {
	connection cli.Connection
}

func NewConsoleResource(connection cli.Connection) *Console {
	return &Console{connection: connection}
}

func (t *Console) Console(request *restful.Request, response *restful.Response) {
	console := request.QueryParameter("console")
	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	vm := v1.NewVMReferenceFromNameWithNS(namespace, vmName)
	logger := log.Log.Object(vm)
	domain, err := t.connection.LookupDomainByName(cache.VMNamespaceKeyFunc(vm))
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Reason(err).Error("Domain not found.")
			response.WriteError(http.StatusNotFound, err)
			return
		} else {
			response.WriteError(http.StatusInternalServerError, err)
			logger.Reason(err).Error("Failed to look up domain.")
			return
		}
	}
	defer domain.Free()

	uid, err := domain.GetUUIDString()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		logger.Reason(err).Error("Failed to look up domain UID.")
		return
	}
	vm.GetObjectMeta().SetUID(types.UID(uid))
	logger = log.Log.Object(vm)

	logger.Infof("Opening connection to console %s", console)

	consoleStream, err := t.connection.NewStream(0)
	if err != nil {
		logger.Reason(err).Error("Creating a consoleStream failed.")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	defer consoleStream.Close()

	logger.V(3).Info("Stream created.")

	err = domain.OpenConsole(console, consoleStream.UnderlyingStream(), libvirt.DOMAIN_CONSOLE_FORCE)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		logger.Reason(err).Error("Failed to open console.")
		return
	}
	logger.V(3).Info("Connection to console created.")

	errorChan := make(chan error)

	ws, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		logger.Reason(err).Error("Failed to upgrade websocket connection.")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer ws.Close()

	wsReadWriter := &TextReadWriter{ws}

	go func() {
		_, err := io.Copy(consoleStream, wsReadWriter)
		errorChan <- err
	}()

	go func() {
		_, err := io.Copy(wsReadWriter, consoleStream)
		errorChan <- err
	}()

	err = <-errorChan

	if err != nil {
		logger.Reason(err).Error("Proxying data between libvirt and the websocket failed.")
	}

	logger.V(3).Info("Done.")
	response.WriteHeader(http.StatusOK)
}

type TextReadWriter struct {
	*websocket.Conn
}

func (s *TextReadWriter) Write(p []byte) (int, error) {
	err := s.Conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, s.err(err)
	}
	return len(p), nil
}

func (s *TextReadWriter) Read(p []byte) (int, error) {
	_, r, err := s.Conn.NextReader()
	if err != nil {
		return 0, s.err(err)
	}
	n, err := r.Read(p)
	return n, s.err(err)
}

func (s *TextReadWriter) err(err error) error {
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
