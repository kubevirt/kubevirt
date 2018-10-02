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

package migrationproxy

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"kubevirt.io/kubevirt/pkg/log"
)

type ProxyManager interface {
	StartTargetListener(key string, targetUnixFile string) error
	GetTargetListenerPort(key string) int
	StopTargetListener(key string)

	StartSourceListener(key string, targetAddress string) error
	GetSourceListenerFile(key string) string
	StopSourceListener(key string)
}

type migrationProxyManager struct {
	virtShareDir  string
	sourceProxies map[string]*migrationProxy
	targetProxies map[string]*migrationProxy
	managerLock   sync.Mutex
}

type migrationProxy struct {
	unixSocketPath string
	tcpBindAddress string
	tcpBindPort    int
	targetAddress  string
	targetProtocol string
	stopChan       chan struct{}
	listenErrChan  chan error
	fdChan         chan net.Conn

	listener net.Listener
}

func NewMigrationProxyManager(virtShareDir string) ProxyManager {
	return &migrationProxyManager{
		virtShareDir:  virtShareDir,
		sourceProxies: make(map[string]*migrationProxy),
		targetProxies: make(map[string]*migrationProxy),
	}
}

func SourceUnixFile(virtShareDir string, key string) string {
	return filepath.Join(virtShareDir, "migrationproxy", key+"-source.sock")
}

func (m *migrationProxyManager) StartTargetListener(key string, targetUnixFile string) error {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxy, exists := m.targetProxies[key]

	if exists {
		if curProxy.targetAddress == targetUnixFile {
			// No Op, already exists
			return nil
		} else {
			// stop the current proxy and point it somewhere new.
			curProxy.StopListening()
		}
	}

	// 0 means random port is used
	proxy := NewTargetProxy("0.0.0.0", 0, targetUnixFile)

	err := proxy.StartListening()
	if err != nil {
		proxy.StopListening()
		return err
	}

	log.Log.Infof("Proxy Target listening on port %d for key %s", proxy.tcpBindPort, key)

	m.targetProxies[key] = proxy
	return nil

}

func (m *migrationProxyManager) GetSourceListenerFile(key string) string {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxy, exists := m.sourceProxies[key]
	if exists {
		return curProxy.unixSocketPath
	}
	return ""
}

func (m *migrationProxyManager) GetTargetListenerPort(key string) int {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxy, exists := m.targetProxies[key]
	if exists {
		return curProxy.tcpBindPort
	}
	return 0
}

func (m *migrationProxyManager) StopTargetListener(key string) {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxy, exists := m.targetProxies[key]
	if exists {
		curProxy.StopListening()
		delete(m.targetProxies, key)
		log.Log.Infof("Stopping proxy target %s listening on %d", key, curProxy.tcpBindPort)
	}
}

func (m *migrationProxyManager) StartSourceListener(key string, targetAddress string) error {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxy, exists := m.sourceProxies[key]

	if exists {
		if curProxy.targetAddress == targetAddress {
			// No Op, already exists
			return nil
		} else {
			// stop the current proxy and point it somewhere new.
			curProxy.StopListening()
		}
	}
	filePath := SourceUnixFile(m.virtShareDir, key)

	os.RemoveAll(filePath)
	proxy := NewSourceProxy(filePath, targetAddress)

	err := proxy.StartListening()
	if err != nil {
		proxy.StopListening()
		return err
	}

	log.Log.Infof("Proxy Source listening on unix file %s for key %s", filePath, key)
	m.sourceProxies[key] = proxy
	return nil
}

func (m *migrationProxyManager) StopSourceListener(key string) {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxy, exists := m.sourceProxies[key]
	if exists {
		curProxy.StopListening()
		delete(m.sourceProxies, key)
	}
	filePath := SourceUnixFile(m.virtShareDir, key)
	os.RemoveAll(filePath)
}

// SRC POD ENV(migration unix socket) <-> HOST ENV (tcp client) <-----> HOST ENV (tcp server) <-> TARGET POD ENV (libvirtd unix socket)

// Source proxy exposes a unix socket server and pipes to an outbound TCP connection.
func NewSourceProxy(unixSocketPath string, tcpTargetAddress string) *migrationProxy {
	return &migrationProxy{
		unixSocketPath: unixSocketPath,
		targetAddress:  tcpTargetAddress,
		targetProtocol: "tcp",
		stopChan:       make(chan struct{}),
		fdChan:         make(chan net.Conn, 1),
		listenErrChan:  make(chan error, 1),
	}
}

// Target proxy listens on a tcp socket and pipes to a libvirtd unix socket
func NewTargetProxy(tcpBindAddress string, tcpBindPort int, libvirtdSocketPath string) *migrationProxy {

	return &migrationProxy{
		tcpBindAddress: tcpBindAddress,
		tcpBindPort:    tcpBindPort,
		targetAddress:  libvirtdSocketPath,
		targetProtocol: "unix",
		stopChan:       make(chan struct{}),
		fdChan:         make(chan net.Conn, 1),
		listenErrChan:  make(chan error, 1),
	}

}

func (m *migrationProxy) createTcpListener() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", m.tcpBindAddress, m.tcpBindPort))
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}

	if m.tcpBindPort == 0 {
		// update the random port that was selected
		m.tcpBindPort = listener.Addr().(*net.TCPAddr).Port
	}

	m.listener = listener
	return nil
}

func (m *migrationProxy) createUnixListener() error {

	os.RemoveAll(m.unixSocketPath)
	err := os.MkdirAll(filepath.Dir(m.unixSocketPath), 0755)
	if err != nil {
		log.Log.Reason(err).Error("unable to create directory for unix socket")
		return err
	}

	listener, err := net.Listen("unix", m.unixSocketPath)
	if err != nil {
		log.Log.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}

	m.listener = listener
	return nil

}

func (m *migrationProxy) StopListening() {
	close(m.stopChan)
	m.listener.Close()
}

func handleConnection(fd net.Conn, targetAddress string, targetProtocol string, stopChan chan struct{}) {
	defer fd.Close()

	outBoundErr := make(chan error)
	inBoundErr := make(chan error)
	conn, err := net.Dial(targetProtocol, targetAddress)
	if err != nil {
		log.Log.Reason(err).Errorf("unable to create outbound leg of proxy to host %s", targetAddress)
		return
	}

	go func() {
		//from outbound connection to proxy
		n, err := io.Copy(fd, conn)
		log.Log.Infof("%d bytes read from oubound connection", n)
		inBoundErr <- err
	}()
	go func() {
		//from proxy to outbound connection

		n, err := io.Copy(conn, fd)
		log.Log.Infof("%d bytes written oubound connection", n)
		outBoundErr <- err
	}()

	select {
	case err = <-outBoundErr:
		if err != nil {
			log.Log.Reason(err).Errorf("error encountered copying data to outbound proxy connection %s", targetAddress)
		}
	case err = <-inBoundErr:
		if err != nil {
			log.Log.Reason(err).Errorf("error encountered reading data to proxy connection %s", targetAddress)
		}
	case <-stopChan:
		log.Log.Infof("stop channel terminated proxy")
	}
}

func (m *migrationProxy) StartListening() error {

	if m.unixSocketPath != "" {
		err := m.createUnixListener()
		if err != nil {
			return err
		}
	} else {
		err := m.createTcpListener()
		if err != nil {
			return err
		}
	}

	go func(ln net.Listener, fdChan chan net.Conn, listenErr chan error) {
		for {
			fd, err := ln.Accept()
			if err != nil {
				listenErr <- err
				log.Log.Reason(err).Error("proxy unix socket listener returned error.")
				break
			} else {
				fdChan <- fd
			}
		}
	}(m.listener, m.fdChan, m.listenErrChan)

	go func(targetAddress string, targetProtocol string, fdChan chan net.Conn, stopChan chan struct{}, listenErrChan chan error) {
		for {
			select {
			case fd := <-fdChan:
				go handleConnection(fd, targetAddress, targetProtocol, stopChan)
			case <-stopChan:
				return
			case <-listenErrChan:
				return
			}
		}

	}(m.targetAddress, m.targetProtocol, m.fdChan, m.stopChan, m.listenErrChan)

	return nil
}
