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
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

type ConsoleHandler struct {
	podIsolationDetector isolation.PodIsolationDetector
	serialStopChans      map[types.UID](chan struct{})
	vncStopChans         map[types.UID](chan struct{})
	serialLock           *sync.Mutex
	vncLock              *sync.Mutex
	vmiInformer          cache.SharedIndexInformer
}

func NewConsoleHandler(podIsolationDetector isolation.PodIsolationDetector, vmiInformer cache.SharedIndexInformer) *ConsoleHandler {
	return &ConsoleHandler{
		podIsolationDetector: podIsolationDetector,
		serialStopChans:      make(map[types.UID](chan struct{})),
		vncStopChans:         make(map[types.UID](chan struct{})),
		serialLock:           &sync.Mutex{},
		vncLock:              &sync.Mutex{},
		vmiInformer:          vmiInformer,
	}
}

func (t *ConsoleHandler) VNCHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := t.getVMI(request)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}
	unixSocketPath, err := t.getUnixSocketPath(vmi, "virt-vnc")
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed finding unix socket for VNC console")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	uid := vmi.GetUID()
	stopChn := newStopChan(uid, t.vncLock, t.vncStopChans)
	cleanup := func() {
		deleteStopChan(uid, stopChn, t.vncLock, t.vncStopChans)
	}
	t.stream(vmi, request, response, unixSocketPath, stopChn, cleanup)
}

func (t *ConsoleHandler) SerialHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := t.getVMI(request)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}
	unixSocketPath, err := t.getUnixSocketPath(vmi, "virt-serial0")
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed finding unix socket for serial console")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	uid := vmi.GetUID()
	stopCh := newStopChan(uid, t.serialLock, t.serialStopChans)
	cleanup := func() {
		deleteStopChan(uid, stopCh, t.serialLock, t.serialStopChans)
	}
	t.stream(vmi, request, response, unixSocketPath, stopCh, cleanup)
}

func (t *ConsoleHandler) getVMI(request *restful.Request) (*v1.VirtualMachineInstance, int, error) {
	key := fmt.Sprintf("%s/%s", request.PathParameter("namespace"), request.PathParameter("name"))
	vmiObj, vmiExists, err := t.vmiInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if !vmiExists {
		return nil, http.StatusNotFound, fmt.Errorf("VMI %s does not exist", key)
	}
	return vmiObj.(*v1.VirtualMachineInstance), 0, nil
}

func newStopChan(uid types.UID, lock *sync.Mutex, stopChans map[types.UID](chan struct{})) chan struct{} {
	lock.Lock()
	defer lock.Unlock()
	// close current connection, if exists
	if c, ok := stopChans[uid]; ok {
		delete(stopChans, uid)
		close(c)
	}
	// create a stop channel for the new connection
	stopCh := make(chan struct{})
	stopChans[uid] = stopCh
	return stopCh
}

func deleteStopChan(uid types.UID, stopChn chan struct{}, lock *sync.Mutex, stopChans map[types.UID](chan struct{})) {
	lock.Lock()
	defer lock.Unlock()
	// delete the stop channel from the cache if needed
	if c, ok := stopChans[uid]; ok && c == stopChn {
		delete(stopChans, uid)
	}
}

func (t *ConsoleHandler) getUnixSocketPath(vmi *v1.VirtualMachineInstance, socketName string) (string, error) {
	result, err := t.podIsolationDetector.Detect(vmi)
	if err != nil {
		return "", err
	}
	socketDir := path.Join("proc", strconv.Itoa(result.Pid()), "root", "var", "run", "kubevirt-private", string(vmi.GetUID()))
	socketPath := path.Join(socketDir, socketName)
	if _, err = os.Stat(socketPath); os.IsNotExist(err) {
		return "", err
	}
	// See https://github.com/kubevirt/kubevirt/pull/2171
	if err = os.Chmod(socketDir, 0444); err != nil {
		return "", err
	}
	return socketPath, nil
}

type cleanupOnError func()

func (t *ConsoleHandler) stream(vmi *v1.VirtualMachineInstance, request *restful.Request, response *restful.Response, unixSocketPath string, stopCh chan struct{}, cleanup cleanupOnError) {
	var upgrader = kubecli.NewUpgrader()
	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	defer clientSocket.Close()

	log.Log.Object(vmi).Infof("Websocket connection upgraded")
	log.Log.Object(vmi).Infof("Connecting to %s", unixSocketPath)

	fd, err := net.Dial("unix", unixSocketPath)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("failed to dial unix socket %s", unixSocketPath)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer fd.Close()

	log.Log.Object(vmi).Infof("Connected to %s", unixSocketPath)

	errCh := make(chan error)
	go func() {
		_, err := kubecli.CopyTo(clientSocket, fd)
		log.Log.Object(vmi).Reason(err).Error("error encountered reading from unix socket")
		errCh <- err
	}()

	go func() {
		_, err := kubecli.CopyFrom(fd, clientSocket)
		log.Log.Object(vmi).Reason(err).Error("error encountered reading from client (virt-api) websocket")
		errCh <- err
	}()

	select {
	case <-stopCh:
		response.WriteHeader(http.StatusOK)
	case err := <-errCh:
		if err != nil && err != io.EOF {
			log.Log.Object(vmi).Reason(err).Error("Error in proxing websocket and unix socket")
			response.WriteHeader(http.StatusInternalServerError)
		} else {
			response.WriteHeader(http.StatusOK)
		}
		cleanup()
	}
}
