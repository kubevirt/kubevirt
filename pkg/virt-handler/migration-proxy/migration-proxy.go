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

package migrationproxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	LibvirtDirectMigrationPort = 49152
	LibvirtBlockMigrationPort  = 49153

	// Pod-relative path for virt-handler safepath operations (JoinNoFollow).
	// QEMU and virt-launcher use util.VirtShareDir (/var/run/kubevirt); both refer
	// to the same location when /var/run symlinks to ../run in the container.
	VirtLauncherKubevirtRunDir = "/run/kubevirt"
)

var migrationPortsRange = []int{LibvirtDirectMigrationPort, LibvirtBlockMigrationPort}

type ProxyManager interface {
	StartTargetListener(key string, mountRoot *safepath.Path, targetUnixFiles []string) error
	GetTargetListenerPorts(key string) map[string]int
	StopTargetListener(key string)

	StartSourceListener(key string, mountRoot *safepath.Path, targetAddress string, destSrcPortMap map[string]int) error
	GetSourceListenerFiles(key string) []string
	StopSourceListener(key string)

	OpenListenerCount() int

	InitiateGracefulShutdown()
}

type migrationProxyManager struct {
	sourceProxies      map[string][]*migrationProxy
	targetProxies      map[string][]*migrationProxy
	managerLock        sync.Mutex
	serverTLSConfig    *tls.Config
	migrationTLSConfig *tls.Config

	isShuttingDown bool
	config         *virtconfig.ClusterConfig
}

type MigrationProxyListener interface {
	Start() error
	Stop()
}

type migrationProxy struct {
	mountRoot      *safepath.Path
	relativePath   string
	unixSocketPath string
	tcpBindAddress string
	tcpBindPort    int
	targetAddress  string
	targetProtocol string
	stopChan       chan struct{}
	listenErrChan  chan error
	fdChan         chan net.Conn

	listener           net.Listener
	serverTLSConfig    *tls.Config
	migrationTLSConfig *tls.Config

	logger *log.FilteredLogger
}

func (m *migrationProxyManager) InitiateGracefulShutdown() {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	m.isShuttingDown = true
}

func (m *migrationProxyManager) OpenListenerCount() int {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	return len(m.sourceProxies) + len(m.targetProxies)
}

func GetMigrationPortsList(isBlockMigration bool) (ports []int) {
	ports = append(ports, migrationPortsRange[0])
	if isBlockMigration {
		ports = append(ports, migrationPortsRange[1])
	}
	return
}

func NewMigrationProxyManager(serverTLSConfig *tls.Config, migrationTLSConfig *tls.Config, config *virtconfig.ClusterConfig) ProxyManager {
	return &migrationProxyManager{
		sourceProxies:      make(map[string][]*migrationProxy),
		targetProxies:      make(map[string][]*migrationProxy),
		serverTLSConfig:    serverTLSConfig,
		migrationTLSConfig: migrationTLSConfig,
		config:             config,
	}
}

func MigrationSocketsDir(virtShareDir string) string {
	return filepath.Join(virtShareDir, "migrationproxy")
}

func SourceUnixFile(baseDir string, key string) string {
	return filepath.Join(MigrationSocketsDir(baseDir), key+"-source.sock")
}

func (m *migrationProxyManager) StartTargetListener(key string, mountRoot *safepath.Path, targetUnixFiles []string) error {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	if m.isShuttingDown {
		return fmt.Errorf("unable to process new migration connections during virt-handler shutdown")
	}

	isExistingProxy := func(curProxies []*migrationProxy, targetUnixFiles []string) bool {
		// make sure that all elements in the existing proxy match to the provided targetUnixFiles
		if len(curProxies) != len(targetUnixFiles) {
			return false
		}
		existingSocketFiles := make(map[string]bool)
		for _, file := range targetUnixFiles {
			existingSocketFiles[file] = true
		}
		for _, curProxy := range curProxies {
			if _, ok := existingSocketFiles[curProxy.relativePath]; !ok {
				return false
			}
		}
		return true
	}
	curProxies, exists := m.targetProxies[key]

	if exists {
		if isExistingProxy(curProxies, targetUnixFiles) {
			// No Op, already exists
			return nil
		} else {
			// stop the current proxy and point it somewhere new.
			for _, curProxy := range curProxies {
				curProxy.logger.Infof("Manager stopping proxy on target node due to new unix filepath location")
				curProxy.Stop()
			}
		}
	}

	zeroAddress := ip.GetIPZeroAddress()
	proxiesList := []*migrationProxy{}
	serverTLSConfig := m.serverTLSConfig
	if m.config.GetMigrationConfiguration().DisableTLS != nil && *m.config.GetMigrationConfiguration().DisableTLS {
		serverTLSConfig = nil
	}
	for _, targetUnixFile := range targetUnixFiles {
		proxy := NewTargetProxy(
			zeroAddress,
			0, // random port is used
			serverTLSConfig,
			mountRoot,
			targetUnixFile,
			key)

		err := proxy.Start()
		if err != nil {
			proxy.Stop()
			// close all already created proxies for this key
			for _, curProxy := range proxiesList {
				curProxy.Stop()
			}
			return err
		}
		proxiesList = append(proxiesList, proxy)
		proxy.logger.Infof("Manager created proxy on target")
	}
	m.targetProxies[key] = proxiesList
	return nil
}

func (m *migrationProxyManager) GetSourceListenerFiles(key string) []string {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxies, exists := m.sourceProxies[key]
	socketsList := []string{}
	if exists {
		for _, curProxy := range curProxies {
			socketsList = append(socketsList, curProxy.relativePath)
		}
	}
	return socketsList
}

func ConstructProxyKey(id string, port int) string {
	key := id
	if port != 0 {
		key += fmt.Sprintf("-%d", port)
	}
	return key
}

func (m *migrationProxyManager) GetTargetListenerPorts(key string) map[string]int {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	getPortFromSocket := func(id string, path string) int {
		for _, port := range migrationPortsRange {
			key := ConstructProxyKey(id, port)
			if strings.Contains(path, key) {
				return port
			}
		}
		return 0
	}

	curProxies, exists := m.targetProxies[key]
	targetSrcPortMap := make(map[string]int)

	if exists {
		for _, curProxy := range curProxies {
			port := strconv.Itoa(curProxy.tcpBindPort)
			targetSrcPortMap[port] = getPortFromSocket(key, curProxy.relativePath)
		}
	}
	return targetSrcPortMap
}

func (m *migrationProxyManager) StopTargetListener(key string) {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxies, exists := m.targetProxies[key]
	if exists {
		for _, curProxy := range curProxies {
			curProxy.logger.Info("Manager stopping proxy on target node")
			curProxy.Stop()
			delete(m.targetProxies, key)
		}
	}
}

func (m *migrationProxyManager) StartSourceListener(key string, mountRoot *safepath.Path, targetAddress string, destSrcPortMap map[string]int) error {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	if m.isShuttingDown {
		return fmt.Errorf("unable to process new migration connections during virt-handler shutdown")
	}

	isExistingProxy := func(curProxies []*migrationProxy, targetAddress string, destSrcPortMap map[string]int) bool {
		if len(curProxies) != len(destSrcPortMap) {
			return false
		}
		destSrcLookup := make(map[string]int)
		for dest, src := range destSrcPortMap {
			addr := net.JoinHostPort(targetAddress, dest)
			destSrcLookup[addr] = src
		}
		for _, curProxy := range curProxies {
			if _, ok := destSrcLookup[curProxy.targetAddress]; !ok {
				return false
			}
		}
		return true
	}

	curProxies, exists := m.sourceProxies[key]

	if exists {
		if isExistingProxy(curProxies, targetAddress, destSrcPortMap) {
			// No Op, already exists
			return nil
		} else {
			// stop the current proxy and point it somewhere new.
			for _, curProxy := range curProxies {
				curProxy.logger.Infof("Manager is stopping proxy on source node due to new target location")
				curProxy.Stop()
			}
		}
	}
	migrationTLSConfig := m.migrationTLSConfig
	if m.config.GetMigrationConfiguration().DisableTLS != nil && *m.config.GetMigrationConfiguration().DisableTLS {
		migrationTLSConfig = nil
	}
	proxiesList := []*migrationProxy{}
	for destPort, srcPort := range destSrcPortMap {
		proxyKey := ConstructProxyKey(key, srcPort)
		targetFullAddr := net.JoinHostPort(targetAddress, destPort)
		relativePath := SourceUnixFile(VirtLauncherKubevirtRunDir, proxyKey)

		proxy := NewSourceProxy(mountRoot, relativePath, targetFullAddr, migrationTLSConfig, key)

		err := proxy.Start()
		if err != nil {
			proxy.Stop()
			// close all already created proxies for this key
			for _, curProxy := range proxiesList {
				curProxy.Stop()
			}
			return err
		}
		proxiesList = append(proxiesList, proxy)
		proxy.logger.Infof("Manager created proxy on source node")
	}
	m.sourceProxies[key] = proxiesList
	return nil
}

func (m *migrationProxyManager) StopSourceListener(key string) {
	m.managerLock.Lock()
	defer m.managerLock.Unlock()

	curProxies, exists := m.sourceProxies[key]
	if exists {
		for _, curProxy := range curProxies {
			curProxy.logger.Infof("Manager stopping proxy on source node")
			curProxy.Stop()
			removeSourceUnixSocket(curProxy)
		}
		delete(m.sourceProxies, key)
	}
}

// When mountRoot is nil, relativePath is treated as an absolute host path. This is
// only used by virt-launcher, which runs inside the pod mount namespace.
func removeSourceUnixSocket(proxy *migrationProxy) {
	if proxy.mountRoot == nil {
		os.RemoveAll(proxy.relativePath)
		return
	}

	socketPath, err := safepath.JoinNoFollow(proxy.mountRoot, proxy.relativePath)
	if err != nil {
		log.Log.Reason(err).Errorf("unable to resolve source unix socket %q for cleanup", proxy.relativePath)
		return
	}
	if err := safepath.UnlinkAtNoFollow(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Log.Reason(err).Errorf("unable to remove source unix socket %q", proxy.relativePath)
	}
}

// SRC POD ENV(migration unix socket) <-> HOST ENV (tcp client) <-----> HOST ENV (tcp server) <-> TARGET POD ENV (virtqemud unix socket)

// Source proxy exposes a unix socket server and pipes to an outbound TCP connection.
// When mountRoot is nil, relativePath is treated as an absolute host path. This is
// only used by virt-launcher, which runs inside the pod mount namespace.
func NewSourceProxy(mountRoot *safepath.Path, relativePath string, tcpTargetAddress string, migrationTLSConfig *tls.Config, vmiUID string) *migrationProxy {
	return &migrationProxy{
		mountRoot:          mountRoot,
		relativePath:       relativePath,
		targetAddress:      tcpTargetAddress,
		targetProtocol:     "tcp",
		stopChan:           make(chan struct{}),
		fdChan:             make(chan net.Conn, 1),
		listenErrChan:      make(chan error, 1),
		migrationTLSConfig: migrationTLSConfig,
		logger:             log.Log.With("uid", vmiUID).With("listening", filepath.Base(relativePath)).With("outbound", tcpTargetAddress),
	}
}

// Target proxy listens on a tcp socket and pipes to a virtqemud unix socket
func NewTargetProxy(
	tcpBindAddress string,
	tcpBindPort int,
	serverTLSConfig *tls.Config,
	mountRoot *safepath.Path,
	relativePath string,
	vmiUID string) *migrationProxy {
	return &migrationProxy{
		tcpBindAddress:  tcpBindAddress,
		tcpBindPort:     tcpBindPort,
		targetProtocol:  "unix",
		mountRoot:       mountRoot,
		relativePath:    relativePath,
		stopChan:        make(chan struct{}),
		fdChan:          make(chan net.Conn, 1),
		listenErrChan:   make(chan error, 1),
		serverTLSConfig: serverTLSConfig,
		logger:          log.Log.With("uid", vmiUID).With("outbound", filepath.Base(relativePath)),
	}

}

func (m *migrationProxy) createTcpListener() error {
	var listener net.Listener
	var err error

	laddr := net.JoinHostPort(m.tcpBindAddress, strconv.Itoa(m.tcpBindPort))
	if m.serverTLSConfig != nil {
		listener, err = tls.Listen("tcp", laddr, m.serverTLSConfig)
	} else {
		listener, err = net.Listen("tcp", laddr)
	}
	if err != nil {
		m.logger.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}

	if m.tcpBindPort == 0 {
		// update the random port that was selected
		m.tcpBindPort = listener.Addr().(*net.TCPAddr).Port
		// Add the listener to the log output once we know the port
		m.logger = m.logger.With("listening", fmt.Sprintf("%s:%d", m.tcpBindAddress, m.tcpBindPort))
	}

	m.listener = listener
	return nil
}

func (m *migrationProxy) createUnixListener() error {
	if m.mountRoot == nil {
		return m.createUnixListenerUnsafe()
	}

	socketDirPath := filepath.Dir(m.relativePath)
	socketName := filepath.Base(m.relativePath)

	socketDir, err := safepath.JoinNoFollow(m.mountRoot, socketDirPath)
	if err != nil {
		m.logger.Reason(err).Error("unable to resolve directory for unix socket")
		return err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, socketName)
	if err != nil {
		m.logger.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}

	socketPath, err := safepath.JoinNoFollow(socketDir, socketName)
	if err != nil {
		listener.Close()
		m.logger.Reason(err).Error("unable to resolve unix socket path")
		return err
	}
	if err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath); err != nil {
		listener.Close()
		log.Log.Reason(err).Error("failed to change ownership on migration unix socket")
		return err
	}

	m.listener = listener
	return nil
}

func (m *migrationProxy) createUnixListenerUnsafe() error {
	os.RemoveAll(m.relativePath)
	err := util.MkdirAllWithNosec(filepath.Dir(m.relativePath))
	if err != nil {
		m.logger.Reason(err).Error("unable to create directory for unix socket")
		return err
	}

	listener, err := net.Listen("unix", m.relativePath)
	if err != nil {
		m.logger.Reason(err).Error("failed to create unix socket for proxy service")
		return err
	}
	if err := diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(m.relativePath); err != nil {
		log.Log.Reason(err).Error("failed to change ownership on migration unix socket")
		return err
	}

	m.listener = listener
	return nil
}

func (m *migrationProxy) Stop() {

	close(m.stopChan)
	if m.listener != nil {
		m.logger.Infof("proxy stopped listening")
		m.listener.Close()
	}
}

func (m *migrationProxy) handleConnection(fd net.Conn) {
	defer fd.Close()

	outBoundErr := make(chan error, 1)
	inBoundErr := make(chan error, 1)

	var conn net.Conn
	var err error
	if m.targetProtocol == "tcp" && m.migrationTLSConfig != nil {
		conn, err = tls.Dial(m.targetProtocol, m.targetAddress, m.migrationTLSConfig)
	} else if m.targetProtocol == "tcp" {
		conn, err = net.Dial(m.targetProtocol, m.targetAddress)
	} else if m.targetProtocol == "unix" {
		if m.mountRoot == nil {
			m.logger.Error("mount root is unavailable")
			return
		}

		socketPath, resolveErr := safepath.JoinNoFollow(m.mountRoot, m.relativePath)
		if resolveErr != nil {
			m.logger.Reason(resolveErr).Error("unable to join relative path with mount root")
			return
		}

		err = socketPath.ExecuteNoFollow(func(safePath string) error {
			var dialErr error
			conn, dialErr = net.Dial("unix", safePath)
			if dialErr != nil {
				return fmt.Errorf("failed to dial unix socket %s: %w", safePath, dialErr)
			}
			return nil
		})
	} else {
		m.logger.Errorf("unsupported target protocol: %s", m.targetProtocol)
		return
	}

	if err != nil {
		m.logger.Reason(err).Error("unable to create outbound leg of proxy to host")
		return
	}

	go func() {
		//from outbound connection to proxy
		n, err := io.Copy(fd, conn)
		m.logger.Infof("%d bytes copied outbound to inbound", n)
		inBoundErr <- err
	}()
	go func() {
		//from proxy to outbound connection
		n, err := io.Copy(conn, fd)
		m.logger.Infof("%d bytes copied from inbound to outbound", n)
		outBoundErr <- err
	}()

	select {
	case err = <-outBoundErr:
		if err != nil {
			m.logger.Reason(err).Errorf("error encountered copying data to outbound connection")
		}
	case err = <-inBoundErr:
		if err != nil {
			m.logger.Reason(err).Errorf("error encountered copying data into inbound connection")
		}
	case <-m.stopChan:
		m.logger.Info("stop channel terminated proxy")
	}
}

func (m *migrationProxy) Start() error {

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

	go func(ln net.Listener, fdChan chan net.Conn, listenErr chan error, stopChan chan struct{}) {
		for {
			fd, err := ln.Accept()
			if err != nil {
				listenErr <- err

				select {
				case <-stopChan:
					// If the stopChan is closed, then this is expected. Log at a lesser debug level
					m.logger.Reason(err).V(3).Infof("stopChan is closed. Listener exited with expected error.")
				default:
					m.logger.Reason(err).Error("proxy unix socket listener returned error.")
				}
				break
			} else {
				fdChan <- fd
			}
		}
	}(m.listener, m.fdChan, m.listenErrChan, m.stopChan)

	go func(m *migrationProxy) {
		for {
			select {
			case fd := <-m.fdChan:
				go m.handleConnection(fd)
			case <-m.stopChan:
				return
			case <-m.listenErrChan:
				return
			}
		}

	}(m)

	m.logger.Infof("proxy started listening")
	return nil
}
