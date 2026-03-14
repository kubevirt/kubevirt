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
 * Copyright The KubeVirt Authors.
 *
 */

package launcher_clients

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/notify-server/pipe"
)

var IrrecoverableError = errors.New("IrrecoverableError")

type LauncherClientsManager interface {
	GetVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error)
	GetLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error)
	GetLauncherClientInfo(vmi *v1.VirtualMachineInstance) *virtcache.LauncherClientInfo
	CloseLauncherClient(vmi *v1.VirtualMachineInstance)
	IsLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error)
}

type launcherClientsManager struct {
	virtShareDir         string
	launcherClients      virtcache.LauncherClientInfoByVMI
	podIsolationDetector isolation.PodIsolationDetector
}

func NewLauncherClientsManager(
	virtShareDir string,
	podIsolationDetector isolation.PodIsolationDetector,
) LauncherClientsManager {

	l := &launcherClientsManager{
		virtShareDir:         virtShareDir,
		launcherClients:      virtcache.LauncherClientInfoByVMI{},
		podIsolationDetector: podIsolationDetector,
	}

	return l
}

// GetVerifiedLauncherClient returns a launcher client for the given VMI after verifying connectivity.
// Returns two types of errors:
//   - Irrecoverable errors (wrapped with IrrecoverableError): Permanent failures such as
//     socket not found (either during initial client creation or after ping failure) or
//     client creation failure. These indicate the VMI launcher is not available and
//     retrying will not help.
//   - Recoverable errors (not wrapped): Transient failures such as ping failures when the
//     socket still exists or pipe initialization that may succeed on retry.
func (l *launcherClientsManager) GetVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error) {
	client, err = l.GetLauncherClient(vmi)
	if err != nil {
		return client, err
	}

	// Verify connectivity.
	// It's possible the pod has already been torn down along with the VirtualMachineInstance.
	err = client.Ping()
	if err == nil {
		return client, nil
	}

	logger := log.Log.Object(vmi)
	logger.Warningf("Ping vmi failed with %s", err.Error())

	_, irrecoverableErr := cmdclient.FindSocket(vmi)
	if irrecoverableErr != nil {
		return client, fmt.Errorf("%w: %w", IrrecoverableError, irrecoverableErr)
	}
	return client, err
}

// GetLauncherClient returns a launcher client for the given VMI.
// Returns two types of errors:
//   - Irrecoverable errors (wrapped with IrrecoverableError): Permanent failures such as
//     socket not found or client creation failure. These indicate the VMI launcher is not
//     available and retrying will not help.
//   - Recoverable errors (not wrapped): Transient failures such as pipe initialization that may succeed on retry.
func (l *launcherClientsManager) GetLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	var err error

	clientInfo, exists := l.launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		return clientInfo.Client, nil
	}

	socketFile, err := cmdclient.FindSocket(vmi)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", IrrecoverableError, err)
	}

	err = virtcache.GhostRecordGlobalStore.Add(vmi.Namespace, vmi.Name, socketFile, vmi.UID)
	if err != nil {
		return nil, err
	}

	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", IrrecoverableError, err)
	}

	domainPipeStopChan := make(chan struct{})
	//we pipe in the domain socket into the VMI's filesystem
	err = l.startDomainNotifyPipe(domainPipeStopChan, vmi)
	if err != nil {
		client.Close()
		close(domainPipeStopChan)
		return nil, err
	}

	l.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
		Client:              client,
		SocketFile:          socketFile,
		DomainPipeStopChan:  domainPipeStopChan,
		NotInitializedSince: time.Now(),
		Ready:               true,
	})

	return client, nil
}

func (l *launcherClientsManager) GetLauncherClientInfo(vmi *v1.VirtualMachineInstance) *virtcache.LauncherClientInfo {
	launcherInfo, exists := l.launcherClients.Load(vmi.UID)
	if !exists {
		return nil
	}
	return launcherInfo
}

func (l *launcherClientsManager) CloseLauncherClient(vmi *v1.VirtualMachineInstance) {
	// UID is required in order to close socket
	if string(vmi.GetUID()) == "" {
		return
	}

	clientInfo, exists := l.launcherClients.Load(vmi.UID)
	if exists {
		clientInfo.Close()
	}

	virtcache.GhostRecordGlobalStore.Delete(vmi.Namespace, vmi.Name)
	l.launcherClients.Delete(vmi.UID)
}

func (l *launcherClientsManager) IsLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error) {
	var socketFile string

	clientInfo, exists := l.launcherClients.Load(vmi.UID)
	if exists {
		if clientInfo.Ready {
			// use cached socket if we previously established a connection
			socketFile = clientInfo.SocketFile
		} else {
			socketFile, err = cmdclient.FindSocket(vmi)
			if err != nil {
				// socket does not exist, but let's see if the pod is still there
				if _, err = cmdclient.FindPodDirOnHost(vmi, cmdclient.SocketDirectoryOnHost); err != nil {
					// no pod meanst that waiting for it to initialize makes no sense
					return true, true, nil
				}
				// pod is still there, if there is no socket let's wait for it to become ready
				if clientInfo.NotInitializedSince.Before(time.Now().Add(-3 * time.Minute)) {
					return true, true, nil
				}
				return false, false, nil
			}
			clientInfo.Ready = true
			clientInfo.SocketFile = socketFile
		}
	} else {
		clientInfo := &virtcache.LauncherClientInfo{
			NotInitializedSince: time.Now(),
			Ready:               false,
		}
		l.launcherClients.Store(vmi.UID, clientInfo)
		// attempt to find the socket if the established connection doesn't currently exist.
		socketFile, err = cmdclient.FindSocket(vmi)
		// no socket file, no VMI, so it's unresponsive
		if err != nil {
			// socket does not exist, but let's see if the pod is still there
			if _, err = cmdclient.FindPodDirOnHost(vmi, cmdclient.SocketDirectoryOnHost); err != nil {
				// no pod means that waiting for it to initialize makes no sense
				return true, true, nil
			}
			return false, false, nil
		}
		clientInfo.Ready = true
		clientInfo.SocketFile = socketFile
	}
	return cmdclient.IsSocketUnresponsive(socketFile), true, nil
}

func handleDomainNotifyPipe(ctx context.Context, ln net.Listener, virtShareDir string, vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)
	fdChan := pipe.ChanFromListener(ctx, logger, ln)

	// Process new connections
	// exit when stop encountered
	go pipe.Pipe(ctx, fdChan, func(conn net.Conn) {
		pipe.Proxy(logger, conn, pipe.NewConnectToNotifyFunc(virtShareDir))
	})
}

func (l *launcherClientsManager) startDomainNotifyPipe(domainPipeStopChan chan struct{}, vmi *v1.VirtualMachineInstance) error {

	res, err := l.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod when setting up notify pipe: %v", err)
	}

	listener, err := pipe.InjectNotify(res, l.virtShareDir, util.IsNonRootVMI(vmi))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-domainPipeStopChan
		cancel()
	}()
	handleDomainNotifyPipe(ctx, listener, l.virtShareDir, vmi)

	return nil
}
