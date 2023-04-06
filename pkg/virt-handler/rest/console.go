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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/emicklei/go-restful/v3"
	"github.com/mdlayher/vsock"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/certificate"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

//const failedRetrieveVMI = "Failed to retrieve VMI"

type ConsoleHandler struct {
	podIsolationDetector isolation.PodIsolationDetector
	serialStopChans      map[types.UID](chan struct{})
	vncStopChans         map[types.UID](chan struct{})
	serialLock           *sync.Mutex
	vncLock              *sync.Mutex
	vmiInformer          cache.SharedIndexInformer
	usbredir             map[types.UID]UsbredirHandlerVMI
	usbredirLock         *sync.Mutex
	certManager          certificate.Manager
}

type UsbredirHandlerVMI struct {
	stopChans map[int](chan struct{})
}

func NewConsoleHandler(podIsolationDetector isolation.PodIsolationDetector, vmiInformer cache.SharedIndexInformer, certManager certificate.Manager) *ConsoleHandler {
	return &ConsoleHandler{
		podIsolationDetector: podIsolationDetector,
		serialStopChans:      make(map[types.UID](chan struct{})),
		vncStopChans:         make(map[types.UID](chan struct{})),
		serialLock:           &sync.Mutex{},
		vncLock:              &sync.Mutex{},
		usbredirLock:         &sync.Mutex{},
		vmiInformer:          vmiInformer,
		usbredir:             make(map[types.UID]UsbredirHandlerVMI),
		certManager:          certManager,
	}
}

func (t *ConsoleHandler) USBRedirHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, t.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedRetrieveVMI)
		response.WriteError(code, err)
		return
	}

	uid := vmi.GetUID()
	stopChan := make(chan struct{})
	var slotId int
	var unixSocketPath string
	ok := func() bool {
		// For simplicity, we handle one usbredir request at the time, for all VMIs
		// handled by virt-handler
		t.usbredirLock.Lock()
		defer t.usbredirLock.Unlock()

		if _, exists := t.usbredir[uid]; !exists {
			// Initialize
			t.usbredir[uid] = UsbredirHandlerVMI{
				stopChans: make(map[int](chan struct{})),
			}
		}

		usbHandler := t.usbredir[uid]
		// Find the first USB device slot available
		for slotId = 0; slotId < v1.UsbClientPassthroughMaxNumberOf; slotId++ {
			if _, inUse := usbHandler.stopChans[slotId]; !inUse {
				break
			}
		}

		if slotId == v1.UsbClientPassthroughMaxNumberOf {
			log.Log.Object(vmi).Reason(err).Errorf("All USB devices are in use.")
			response.WriteError(http.StatusServiceUnavailable, err)
			return false
		}

		unixSocketPath, err = t.getUnixSocketPath(vmi, fmt.Sprintf("virt-usbredir-%d", slotId))
		if err != nil {
			log.Log.Object(vmi).Reason(err).Error("Failed on finding unix socket for USBRedir")
			response.WriteError(http.StatusBadRequest, err)
			return false
		}

		usbHandler.stopChans[slotId] = stopChan
		return true
	}()

	if !ok {
		return
	}

	defer func() {
		t.usbredirLock.Lock()
		defer t.usbredirLock.Unlock()
		usbHandler := t.usbredir[uid]
		delete(usbHandler.stopChans, slotId)
	}()
	t.stream(vmi, request, response, unixSocketDialer(vmi, unixSocketPath), stopChan)
}

func (t *ConsoleHandler) VNCHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, t.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedRetrieveVMI)
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
	defer deleteStopChan(uid, stopChn, t.vncLock, t.vncStopChans)
	t.stream(vmi, request, response, unixSocketDialer(vmi, unixSocketPath), stopChn)
}

func (t *ConsoleHandler) SerialHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, t.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedRetrieveVMI)
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
	defer deleteStopChan(uid, stopCh, t.serialLock, t.serialStopChans)
	t.stream(vmi, request, response, unixSocketDialer(vmi, unixSocketPath), stopCh)
}

func (t *ConsoleHandler) VSOCKHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, t.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedRetrieveVMI)
		response.WriteError(code, err)
		return
	}
	log.Log.Object(vmi).Info("In VSOCKHandler")
	if !util.IsAutoAttachVSOCK(vmi) {
		response.WriteError(http.StatusBadRequest, errors.New("VM doesn't have VSOCK enabled"))
		return
	}
	if vmi.Status.VSOCKCID == nil {
		// This should not happen.
		err := errors.New("VSOCK CID is nil")
		log.Log.Object(vmi).Error(err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	portParam := request.QueryParameter("port")
	tlsParam := request.QueryParameter("tls")
	if tlsParam == "" {
		tlsParam = "false"
	}
	port, err := strconv.ParseUint(portParam, 10, 32)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed parsing the path parameter port %s", portParam)
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	useTLS, err := strconv.ParseBool(tlsParam)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed parsing the path parameter useTLS %s", tlsParam)
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	cid := *vmi.Status.VSOCKCID
	t.stream(vmi, request, response, func() (net.Conn, error) {
		log.Log.Object(vmi).Infof("Connecting to %d:%d", cid, port)
		conn, err := vsock.Dial(cid, uint32(port), &vsock.Config{})
		if err != nil {
			log.Log.Object(vmi).Reason(err).Errorf("failed to dial vsock %d:%d", cid, port)
			return nil, err
		}
		if !useTLS {
			log.Log.Object(vmi).Infof("Connected to %d:%d", cid, port)
			return conn, nil
		}
		tlsConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS13,
			GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return t.certManager.Current(), nil
			},
		})
		if err := tlsConn.Handshake(); err != nil {
			log.Log.Object(vmi).Reason(err).Errorf("Failed to connect to %d:%d over TLS", cid, port)
			return nil, err
		}
		log.Log.Object(vmi).Infof("Connected to %d:%d over TLS", cid, port)
		return tlsConn, nil
	}, make(chan struct{})) // It is legitimate and up to the guest-application to accept multiple connections.
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
	socketDir := path.Join("/proc", strconv.Itoa(result.Pid()), "root", "var", "run", "kubevirt-private", string(vmi.GetUID()))
	socketPath := path.Join(socketDir, socketName)
	if _, err = os.Stat(socketPath); errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return socketPath, nil
}

func unixSocketDialer(vmi *v1.VirtualMachineInstance, unixSocketPath string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		log.Log.Object(vmi).Infof("Connecting to %s", unixSocketPath)
		fd, err := net.Dial("unix", unixSocketPath)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Errorf("failed to dial unix socket %s", unixSocketPath)
			return nil, err
		}
		log.Log.Object(vmi).Infof("Connected to %s", unixSocketPath)
		return fd, nil
	}
}

func (t *ConsoleHandler) stream(vmi *v1.VirtualMachineInstance, request *restful.Request, response *restful.Response, dial func() (net.Conn, error), stopCh chan struct{}) {
	var upgrader = kubecli.NewUpgrader()
	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	defer clientSocket.Close()

	log.Log.Object(vmi).Infof("Websocket connection upgraded")

	conn, err := dial()
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	errCh := make(chan error, 2)
	go func() {
		_, err := kubecli.CopyTo(clientSocket, conn)
		log.Log.Object(vmi).Reason(err).Error("error encountered reading from unix socket")
		errCh <- err
	}()

	go func() {
		_, err := kubecli.CopyFrom(conn, clientSocket)
		log.Log.Object(vmi).Reason(err).Error("error encountered reading from client (virt-api) websocket")
		errCh <- err
	}()

	select {
	case <-stopCh:
		break
	case err := <-errCh:
		if err != nil && err != io.EOF {
			log.Log.Object(vmi).Reason(err).Error("Error in proxing websocket and unix socket")
			response.WriteHeader(http.StatusInternalServerError)
		}
	}
}
