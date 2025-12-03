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

func (l *launcherClientsManager) GetVerifiedLauncherClient(vmi *v1.VirtualMachineInstance) (client cmdclient.LauncherClient, err error) {
	client, err = l.GetLauncherClient(vmi)
	if err != nil {
		return
	}

	// Verify connectivity.
	// It's possible the pod has already been torn down along with the VirtualMachineInstance.
	err = client.Ping()
	return
}

func (l *launcherClientsManager) GetLauncherClient(vmi *v1.VirtualMachineInstance) (cmdclient.LauncherClient, error) {
	var err error

	clientInfo, exists := l.launcherClients.Load(vmi.UID)
	if exists && clientInfo.Client != nil {
		return clientInfo.Client, nil
	}

	socketFile, err := cmdclient.FindSocket(vmi)
	if err != nil {
		return nil, err
	}

	err = virtcache.GhostRecordGlobalStore.Add(vmi.Namespace, vmi.Name, socketFile, vmi.UID)
	if err != nil {
		return nil, err
	}

	client, err := cmdclient.NewClient(socketFile)
	if err != nil {
		return nil, err
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

	fdChan := make(chan net.Conn, 100)

	// Close listener and exit when stop encountered
	go func() {
		<-ctx.Done()
		log.Log.Object(vmi).Infof("closing notify pipe listener for vmi")
		if err := ln.Close(); err != nil {
			log.Log.Object(vmi).Infof("failed closing notify pipe listener for vmi: %v", err)
		}
	}()

	// Listen for new connections,
	go func(vmi *v1.VirtualMachineInstance, ln net.Listener) {
		for {
			fd, err := ln.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					// As Accept blocks, closing it is our mechanism to exit this loop
					return
				}
				log.Log.Reason(err).Error("Domain pipe accept error encountered.")
				// keep listening until stop invoked
				time.Sleep(1 * time.Second)
			} else {
				fdChan <- fd
			}
		}
	}(vmi, ln)

	// Process new connections
	// exit when stop encountered
	go func(vmi *v1.VirtualMachineInstance, fdChan chan net.Conn, ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case fd := <-fdChan:
				go pipe.Proxy(log.Log.Object(vmi), fd, pipe.NewConnectToNotifyFunc(virtShareDir))
			}
		}
	}(vmi, fdChan, ctx)
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
