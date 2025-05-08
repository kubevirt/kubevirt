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
 * Copyright 2025 The KubeVirt Authors.
 *
 */

package launcher_clients

import (
	goerror "errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
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

	socketFile, err := cmdclient.FindSocketOnHost(vmi)
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
	if exists && clientInfo.Client != nil {
		clientInfo.Client.Close()
		close(clientInfo.DomainPipeStopChan)
	}

	virtcache.GhostRecordGlobalStore.Delete(vmi.Namespace, vmi.Name)
	l.launcherClients.Delete(vmi.UID)
}

// used by unit tests to add mock clients
func (l *launcherClientsManager) addLauncherClient(vmUID types.UID, info *virtcache.LauncherClientInfo) error {
	l.launcherClients.Store(vmUID, info)
	return nil
}

func (l *launcherClientsManager) IsLauncherClientUnresponsive(vmi *v1.VirtualMachineInstance) (unresponsive bool, initialized bool, err error) {
	var socketFile string

	clientInfo, exists := l.launcherClients.Load(vmi.UID)
	if exists {
		fmt.Println("IsLauncherClientUnresponsive exists")
		if clientInfo.Ready == true {
			// use cached socket if we previously established a connection
			socketFile = clientInfo.SocketFile
			fmt.Println("IsLauncherClientUnresponsive Ready")
		} else {
			socketFile, err = cmdclient.FindSocketOnHost(vmi)
			if err != nil {
				// socket does not exist, but let's see if the pod is still there
				if _, err = cmdclient.FindPodDirOnHost(vmi); err != nil {
					// no pod meanst that waiting for it to initialize makes no sense
					fmt.Println("IsLauncherClientUnresponsive no pod meanst that waiting for it to initialize makes no sense")
					return true, true, nil
				}
				// pod is still there, if there is no socket let's wait for it to become ready
				if clientInfo.NotInitializedSince.Before(time.Now().Add(-3 * time.Minute)) {
					fmt.Println("IsLauncherClientUnresponsive NotInitializedSince")
					return true, true, nil
				}
				return false, false, nil
			}
			clientInfo.Ready = true
			clientInfo.SocketFile = socketFile
		}
	} else {
		fmt.Println("IsLauncherClientUnresponsive not exists")
		clientInfo := &virtcache.LauncherClientInfo{
			NotInitializedSince: time.Now(),
			Ready:               false,
		}
		l.launcherClients.Store(vmi.UID, clientInfo)
		// attempt to find the socket if the established connection doesn't currently exist.
		socketFile, err = cmdclient.FindSocketOnHost(vmi)
		// no socket file, no VMI, so it's unresponsive
		if err != nil {
			// socket does not exist, but let's see if the pod is still there
			if _, err = cmdclient.FindPodDirOnHost(vmi); err != nil {
				// no pod meanst that waiting for it to initialize makes no sense
				fmt.Println("IsLauncherClientUnresponsive not exists no pod meanst that waiting for it to initialize makes no sense")
				return true, true, nil
			}
			return false, false, nil
		}
		clientInfo.Ready = true
		clientInfo.SocketFile = socketFile
	}
	fmt.Println("IsLauncherClientUnresponsive all good")
	return cmdclient.IsSocketUnresponsive(socketFile), true, nil
}

func handleDomainNotifyPipe(domainPipeStopChan chan struct{}, ln net.Listener, virtShareDir string, vmi *v1.VirtualMachineInstance) {

	fdChan := make(chan net.Conn, 100)

	// Close listener and exit when stop encountered
	go func() {
		<-domainPipeStopChan
		log.Log.Object(vmi).Infof("closing notify pipe listener for vmi")
		if err := ln.Close(); err != nil {
			log.Log.Object(vmi).Infof("failed closing notify pipe listener for vmi: %v", err)
		}
	}()

	// Listen for new connections,
	go func(vmi *v1.VirtualMachineInstance, ln net.Listener, domainPipeStopChan chan struct{}) {
		for {
			fd, err := ln.Accept()
			if err != nil {
				if goerror.Is(err, net.ErrClosed) {
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
	}(vmi, ln, domainPipeStopChan)

	// Process new connections
	// exit when stop encountered
	go func(vmi *v1.VirtualMachineInstance, fdChan chan net.Conn, domainPipeStopChan chan struct{}) {
		for {
			select {
			case <-domainPipeStopChan:
				return
			case fd := <-fdChan:
				go func(vmi *v1.VirtualMachineInstance) {
					defer fd.Close()

					// pipe the VMI domain-notify.sock to the virt-handler domain-notify.sock
					// so virt-handler receives notifications from the VMI
					conn, err := net.Dial("unix", filepath.Join(virtShareDir, "domain-notify.sock"))
					if err != nil {
						log.Log.Reason(err).Error("error connecting to domain-notify.sock for proxy connection")
						return
					}
					defer conn.Close()

					log.Log.Object(vmi).Infof("Accepted new notify pipe connection for vmi")
					copyErr := make(chan error, 2)
					go func() {
						_, err := io.Copy(fd, conn)
						copyErr <- err
					}()
					go func() {
						_, err := io.Copy(conn, fd)
						copyErr <- err
					}()

					// wait until one of the copy routines exit then
					// let the fd close
					err = <-copyErr
					if err != nil {
						log.Log.Object(vmi).Infof("closing notify pipe connection for vmi with error: %v", err)
					} else {
						log.Log.Object(vmi).Infof("gracefully closed notify pipe connection for vmi")
					}

				}(vmi)
			}
		}
	}(vmi, fdChan, domainPipeStopChan)
}

func (l *launcherClientsManager) startDomainNotifyPipe(domainPipeStopChan chan struct{}, vmi *v1.VirtualMachineInstance) error {

	res, err := l.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod when setting up notify pipe: %v", err)
	}

	// inject the domain-notify.sock into the VMI pod.
	root, err := res.MountRoot()
	if err != nil {
		return err
	}
	socketDir, err := root.AppendAndResolveWithRelativeRoot(l.virtShareDir)
	if err != nil {
		return err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}
	socketPath, err := safepath.JoinNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return err
	}

	if util.IsNonRootVMI(vmi) {
		err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			log.Log.Reason(err).Error("unable to change ownership for domain notify")
			return err
		}
	}

	handleDomainNotifyPipe(domainPipeStopChan, listener, l.virtShareDir, vmi)

	return nil
}
