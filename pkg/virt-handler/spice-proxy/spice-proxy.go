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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package spice_proxy

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"fmt"
	"io"
	"net"
	"sync"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

var (
	proxyRange = []int{30000, 32000}
)

type UnixToTcpProxyManager interface {
	StartListener(vmNamespace, vmName string, vmi *v1.VirtualMachineInstance) error
	GetPort(vmNamespace, vmName string) (int32, error)
	StopListener(vmNamespace, vmName string)
}

type spiceProxyManager struct {
	proxies              map[string]*spiceProxy
	allocatedPort        map[int32]bool
	managerLock          sync.Mutex
	podIsolationDetector isolation.PodIsolationDetector
}

type spiceProxy struct {
	unixSocketPath string
	tcpBindAddress string
	tcpBindPort    int32
	stopChan       chan struct{}
	listenErrChan  chan error
	fdChan         chan net.Conn
	listener       net.Listener
}

func NewSpiceProxyManager(podIsolationDetector isolation.PodIsolationDetector) UnixToTcpProxyManager {
	return &spiceProxyManager{proxies: make(map[string]*spiceProxy), allocatedPort: make(map[int32]bool), podIsolationDetector: podIsolationDetector}
}

func (spm *spiceProxyManager) StartListener(vmNamespace, vmName string, vmi *v1.VirtualMachineInstance) error {
	spm.managerLock.Lock()
	defer spm.managerLock.Unlock()

	if _, isExist := spm.proxies[uniqueVmName(vmNamespace, vmName)]; isExist {
		return nil
	}

	port, err := spm.getFreePort()
	if err != nil {
		return err
	}

	res, err := spm.podIsolationDetector.Detect(vmi)
	if err != nil {
		return err
	}

	// Get the libvirt connection socket file on the destination pod.
	spiceSocketFile := fmt.Sprintf("/proc/%d/root/var/run/kubevirt-private/%s/virt-spice", res.Pid(), vmi.UID)
	spm.proxies[uniqueVmName(vmNamespace, vmName)], err = NewSpiceProxy(spiceSocketFile, port)
	if err != nil {
		return err
	}

	spm.allocatedPort[port] = true

	return nil
}

func (spm *spiceProxyManager) StopListener(vmNamespace, vmName string) {
	spm.managerLock.Lock()
	defer spm.managerLock.Unlock()

	if proxy, isExist := spm.proxies[uniqueVmName(vmNamespace, vmName)]; !isExist {
		log.Log.Warningf("failed to find proxy process for vm %s", uniqueVmName(vmNamespace, vmName))
		return
	} else {
		proxy.Stop()
		err := proxy.listener.Close()
		if err != nil {
			log.Log.Warningf("failed to cluster listener: %v", err)
		}
		delete(spm.allocatedPort, proxy.tcpBindPort)
		delete(spm.proxies, uniqueVmName(vmNamespace, vmName))
	}

}

func (spm *spiceProxyManager) GetPort(vmNamespace, vmName string) (int32, error) {
	if proxy, isExist := spm.proxies[uniqueVmName(vmNamespace, vmName)]; isExist {
		return proxy.tcpBindPort, nil
	}

	return -1, fmt.Errorf("failed to find proxy for vm %s in namespace %s", vmName, vmNamespace)
}

func (spm *spiceProxyManager) getFreePort() (int32, error) {
	for port := proxyRange[0]; port <= proxyRange[1]; port++ {
		if _, isExist := spm.allocatedPort[int32(port)]; !isExist {
			return int32(port), nil
		}
	}

	return 0, fmt.Errorf("failed to find a free port for the spice proxy")
}

// proxy exposes a unix socket server and pipes to an outbound TCP connection.
func NewSpiceProxy(unixSocketPath string, tcpTargetPort int32) (*spiceProxy, error) {
	spiceProxy := &spiceProxy{
		unixSocketPath: unixSocketPath,
		tcpBindAddress: "0.0.0.0",
		tcpBindPort:    tcpTargetPort,
		stopChan:       make(chan struct{}),
		fdChan:         make(chan net.Conn, 1),
		listenErrChan:  make(chan error, 1),
	}

	var err error
	spiceProxy.listener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", spiceProxy.tcpBindAddress, spiceProxy.tcpBindPort))
	if err != nil {
		return spiceProxy, err
	}

	go spiceProxy.Start()
	return spiceProxy, err
}

func (sp *spiceProxy) Start() {
	defer sp.listener.Close()

	go func() {
		for {
			tcpConn, err := sp.listener.Accept()
			if err != nil {
				return
			}

			fd, err := net.Dial("unix", sp.unixSocketPath)
			if err != nil {
				log.Log.Reason(err).Errorf("unable to accept incoming connect")
				return
			}

			go handleConnection(fd, tcpConn, sp.stopChan)
		}
	}()

	select {
	case <-sp.stopChan:

	}
}

func (sp *spiceProxy) Stop() {
	close(sp.stopChan)
}

func handleConnection(fd net.Conn, tcpConn net.Conn, stopChan chan struct{}) {
	defer fd.Close()
	defer tcpConn.Close()

	tcpBoundErr := make(chan error)
	unixBoundErr := make(chan error)

	go func() {
		//from unix connection to tcp
		n, err := io.Copy(fd, tcpConn)
		log.Log.Infof("%d bytes read from oubound connection", n)
		unixBoundErr <- err
	}()
	go func() {
		//from tcp to unixconnection
		n, err := io.Copy(tcpConn, fd)
		log.Log.Infof("%d bytes written oubound connection", n)
		tcpBoundErr <- err
	}()

	var err error
	select {
	case err = <-tcpBoundErr:
		if err != nil {
			log.Log.Reason(err).Errorf("error encountered copying data to tcp connection %s", tcpConn.RemoteAddr())
		}
	case err = <-unixBoundErr:
		if err != nil {
			log.Log.Reason(err).Errorf("error encountered reading data to proxy connection %s", tcpConn.RemoteAddr())
		}
	case <-stopChan:
		log.Log.Infof("stop channel terminated proxy")
	}
}

func uniqueVmName(vmNamespace, vmName string) string {
	return fmt.Sprintf("%s/%s", vmNamespace, vmName)
}
