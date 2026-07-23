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

package synchronization

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"kubevirt.io/client-go/log"

	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
)

const (
	// proxyDialTimeout is the timeout for establishing outbound connections from the proxy
	// to the target. This prevents the proxy from hanging indefinitely if the target is
	// unreachable. 30 seconds is sufficient for cross-cluster network latency while still
	// failing fast enough to provide useful feedback.
	proxyDialTimeout = 30 * time.Second

	// proxyIdleTimeout is the maximum time a connection can be idle (no data transfer)
	// before being closed. This prevents connections from hanging indefinitely if one
	// side stops sending data without closing the connection properly.
	proxyIdleTimeout = 5 * time.Minute

	// proxyAcceptTimeout is the timeout for Accept() calls in the proxy run loop
	// This allows the run loop to periodically check for shutdown signals
	proxyAcceptTimeout = 1 * time.Second
)

type SyncProxyManager struct {
	// Active proxies - each migration can have multiple proxies (one per port)
	// migrationUID -> map[proxyPort]proxy
	sourceProxies map[string]map[int]*migrationProxy
	targetProxies map[string]map[int]*migrationProxy

	// Network IPs
	migrationIP    string
	crossClusterIP string

	lock           sync.Mutex
	isShuttingDown bool
	logger         *log.FilteredLogger
}

type migrationProxy struct {
	migrationUID  string
	listener      net.Listener
	port          int
	protocolPort  int    // protocol port value (0, 49152, 49153) - preserved for map values
	targetAddress string // where to forward connections (IP:port)
	stopChan      chan struct{}
	runExited     chan struct{}
	manager       *SyncProxyManager
	logger        *log.FilteredLogger
	isTarget      bool // true if target proxy, false if source proxy
}

func NewSyncProxyManager() *SyncProxyManager {
	return &SyncProxyManager{
		sourceProxies: make(map[string]map[int]*migrationProxy),
		targetProxies: make(map[string]map[int]*migrationProxy),
		logger:        log.DefaultLogger(),
	}
}

// Initialize stores network IPs (called at startup if crossClusterNetwork configured)
func (m *SyncProxyManager) Initialize(migrationIP, crossClusterIP string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.migrationIP = migrationIP
	m.crossClusterIP = crossClusterIP

	m.logger.Infof("Migration proxy manager initialized - migration0: %s, crosscluster0: %s",
		migrationIP, crossClusterIP)
}

// getExistingProxyPorts extracts a port map from existing proxies (map[proxyPort]protocolPort)
// Returns an empty map if proxyMap is nil or empty
func getExistingProxyPorts(proxyMap map[int]*migrationProxy) map[int]int {
	result := make(map[int]int)
	for _, proxy := range proxyMap {
		result[proxy.port] = proxy.protocolPort
	}
	return result
}

// portMapsMatch checks if the requested port map matches the existing proxy configuration
// Requested format: {TCP-port: protocol-port}
// We only compare protocol ports, since TCP ports are OS-allocated and may differ
func portMapsMatch(requestedPortMap map[int]int, existingProxies map[int]*migrationProxy) bool {
	// Extract protocol ports from requested map
	requestedProtocolPorts := make(map[int]bool)
	for _, protocolPort := range requestedPortMap {
		requestedProtocolPorts[protocolPort] = true
	}

	// Extract protocol ports from existing proxies
	existingProtocolPorts := make(map[int]bool)
	for _, proxy := range existingProxies {
		existingProtocolPorts[proxy.protocolPort] = true
	}

	// Compare sets
	if len(requestedProtocolPorts) != len(existingProtocolPorts) {
		return false
	}
	for protocolPort := range requestedProtocolPorts {
		if !existingProtocolPorts[protocolPort] {
			return false
		}
	}
	return true
}

// proxiesAreRunning reports whether every proxy in the map still has an active run loop.
func proxiesAreRunning(proxyMap map[int]*migrationProxy) bool {
	if len(proxyMap) == 0 {
		return false
	}
	for _, proxy := range proxyMap {
		select {
		case <-proxy.runExited:
			return false
		default:
		}
	}
	return true
}

// onProxyRunExit removes a proxy from the manager after its run loop exits.
func (m *SyncProxyManager) onProxyRunExit(migrationUID string, proxyPort int, isTarget bool) {
	m.lock.Lock()
	defer m.lock.Unlock()

	var proxyMap map[int]*migrationProxy
	if isTarget {
		proxyMap = m.targetProxies[migrationUID]
	} else {
		proxyMap = m.sourceProxies[migrationUID]
	}
	if proxyMap == nil {
		return
	}

	proxy, ok := proxyMap[proxyPort]
	if !ok {
		return
	}

	select {
	case <-proxy.runExited:
	default:
		return
	}

	delete(proxyMap, proxyPort)
	if len(proxyMap) == 0 {
		if isTarget {
			delete(m.targetProxies, migrationUID)
		} else {
			delete(m.sourceProxies, migrationUID)
		}
	}

	proxyType := "source"
	if isTarget {
		proxyType = "target"
	}
	m.logger.Warningf("Removed dead %s migration proxy for %s on port %d", proxyType, migrationUID, proxyPort)
}

// closeProxyListeners closes listeners for the given migration without removing map
// entries. The run loop is expected to clean up the map via onProxyRunExit.
func (m *SyncProxyManager) closeProxyListeners(migrationUID string, isTarget bool) {
	m.lock.Lock()
	var proxies []*migrationProxy
	if isTarget {
		for _, proxy := range m.targetProxies[migrationUID] {
			proxies = append(proxies, proxy)
		}
	} else {
		for _, proxy := range m.sourceProxies[migrationUID] {
			proxies = append(proxies, proxy)
		}
	}
	m.lock.Unlock()

	for _, proxy := range proxies {
		proxy.listener.Close()
	}
}

// cleanupProxyMap closes and cleans up all proxies in the map
func cleanupProxyMap(proxyMap map[int]*migrationProxy) {
	for _, proxy := range proxyMap {
		close(proxy.stopChan)
		proxy.listener.Close()
	}
}

// validatePortMap validates that all port values are within valid ranges
// TCP ports must be 1-65535, protocol ports must be 0-65535
func validatePortMap(portMap map[int]int) error {
	for tcpPort, protocolPort := range portMap {
		if tcpPort < 1 || tcpPort > 65535 {
			return fmt.Errorf("invalid TCP port value: %d (must be 1-65535)", tcpPort)
		}
		if protocolPort < 0 || protocolPort > 65535 {
			return fmt.Errorf("invalid protocol port value: %d (must be 0-65535)", protocolPort)
		}
	}
	return nil
}

// proxyListenerConfig holds parameters for creating a proxy listener
type proxyListenerConfig struct {
	listenIP      string
	protocolPort  int
	targetAddress string
	migrationUID  string
	isTarget      bool
	manager       *SyncProxyManager
	logger        *log.FilteredLogger
}

// createProxyListener creates a listener on the specified IP and returns the proxy port
func createProxyListener(ctx context.Context, config proxyListenerConfig) (*migrationProxy, error) {
	// Check if context is already cancelled before creating listener
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Create listener with OS-allocated port (0 = kernel assigns available port)
	// Use JoinHostPort to properly handle both IPv4 and IPv6 addresses (IPv6 needs brackets)
	listenAddr := net.JoinHostPort(config.listenIP, "0")
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy listener: %v", err)
	}

	// Extract the actual port assigned by the OS (safe type assertion)
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		listener.Close()
		return nil, fmt.Errorf("listener address is not TCP: %T", listener.Addr())
	}
	proxyPort := tcpAddr.Port

	proxy := &migrationProxy{
		migrationUID:  config.migrationUID,
		listener:      listener,
		port:          proxyPort,
		protocolPort:  config.protocolPort,
		targetAddress: config.targetAddress,
		stopChan:      make(chan struct{}),
		runExited:     make(chan struct{}),
		manager:       config.manager,
		logger:        config.logger,
		isTarget:      config.isTarget,
	}

	go proxy.run()

	return proxy, nil
}

// StartTargetProxies creates target-side proxies for inbound migration
// Takes the DirectMigrationNodePorts map from target virt-handler
// Returns a new port map with proxy ports (map[proxyPort]protocolPort)
func (m *SyncProxyManager) StartTargetProxies(ctx context.Context, migrationUID string, targetIP string, targetPortMap map[int]int) (map[int]int, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.isShuttingDown {
		return nil, fmt.Errorf("proxy manager is shutting down")
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate port values to prevent crashes or security issues
	if err := validatePortMap(targetPortMap); err != nil {
		return nil, err
	}

	// Check if already exists
	if existing, exists := m.targetProxies[migrationUID]; exists {
		if proxiesAreRunning(existing) && portMapsMatch(targetPortMap, existing) {
			result := getExistingProxyPorts(existing)
			m.logger.Infof("StartTargetProxies returning existing proxies: %v", result)
			return result, nil
		}
		// Proxies stopped or port map changed - recreate
		m.logger.Infof("StartTargetProxies recreating proxies")
		cleanupProxyMap(existing)
		delete(m.targetProxies, migrationUID)
	}

	// Create proxy map for this migration
	proxyMap := make(map[int]*migrationProxy)
	resultPortMap := make(map[int]int)

	m.logger.Infof("StartTargetProxies input: migrationUID=%s, targetIP=%s, targetPortMap=%v", migrationUID, targetIP, targetPortMap)

	// Create one proxy listener for each target port
	// Note: targetPortMap format is {TCP-port: protocol-port}
	// where protocol-port is 0 for virtqemud, 49152 for libvirt direct, 49153 for block migration
	// We need to proxy the TCP connections, preserving the protocol-port values in the result
	for targetTCPPort, protocolPort := range targetPortMap {
		// Build target address: target virt-handler IP + target virt-handler TCP port
		// (NOT the protocol port - we're proxying TCP-to-TCP here)
		targetAddress := net.JoinHostPort(targetIP, strconv.Itoa(targetTCPPort))

		// Create listener on crosscluster0 with OS-allocated port
		proxy, err := createProxyListener(ctx, proxyListenerConfig{
			listenIP:      m.crossClusterIP,
			protocolPort:  protocolPort,
			targetAddress: targetAddress,
			migrationUID:  migrationUID,
			isTarget:      true,
			manager:       m,
			logger:        m.logger,
		})
		if err != nil {
			// Clean up any proxies we've already created
			cleanupProxyMap(proxyMap)
			return nil, err
		}

		proxyMap[proxy.port] = proxy
		resultPortMap[proxy.port] = protocolPort

		m.logger.Infof("Started target proxy for %s (protocol %d): %s:%d -> %s", migrationUID, protocolPort, m.crossClusterIP, proxy.port, targetAddress)
	}

	m.targetProxies[migrationUID] = proxyMap
	m.logger.Infof("StartTargetProxies output: resultPortMap=%v", resultPortMap)
	return resultPortMap, nil
}

// StartSourceProxies creates source-side proxies for outbound migration
// Takes the target proxy IP and the DirectMigrationNodePorts map from target sync controller
// Returns a new port map with source proxy ports (map[sourceProxyPort]protocolPort)
func (m *SyncProxyManager) StartSourceProxies(ctx context.Context, migrationUID string, targetProxyIP string, targetProxyPortMap map[int]int) (map[int]int, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.isShuttingDown {
		return nil, fmt.Errorf("proxy manager is shutting down")
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate port values to prevent crashes or security issues
	if err := validatePortMap(targetProxyPortMap); err != nil {
		return nil, err
	}

	// Check if already exists
	if existing, exists := m.sourceProxies[migrationUID]; exists {
		if proxiesAreRunning(existing) && portMapsMatch(targetProxyPortMap, existing) {
			result := getExistingProxyPorts(existing)
			m.logger.Infof("StartSourceProxies returning existing proxies: %v", result)
			return result, nil
		}
		// Proxies stopped or port map changed - recreate
		m.logger.Infof("StartSourceProxies recreating proxies")
		cleanupProxyMap(existing)
		delete(m.sourceProxies, migrationUID)
	}

	// Create proxy map for this migration
	proxyMap := make(map[int]*migrationProxy)
	resultPortMap := make(map[int]int)

	m.logger.Infof("StartSourceProxies input: migrationUID=%s, targetProxyIP=%s, targetProxyPortMap=%v", migrationUID, targetProxyIP, targetProxyPortMap)

	// Create one proxy listener for each target proxy port
	// Note: targetProxyPortMap format is {TCP-port: protocol-port}
	// where protocol-port is 0 for virtqemud, 49152 for libvirt direct, 49153 for block migration
	// We need to proxy the TCP connections, preserving the protocol-port values in the result
	for targetProxyTCPPort, protocolPort := range targetProxyPortMap {
		// Build target proxy address: target sync controller crosscluster IP + target proxy TCP port
		// (NOT the protocol port - we're proxying TCP-to-TCP here)
		targetProxyAddress := net.JoinHostPort(targetProxyIP, strconv.Itoa(targetProxyTCPPort))

		// Create listener on migration0 with OS-allocated port
		proxy, err := createProxyListener(ctx, proxyListenerConfig{
			listenIP:      m.migrationIP,
			protocolPort:  protocolPort,
			targetAddress: targetProxyAddress,
			migrationUID:  migrationUID,
			isTarget:      false,
			manager:       m,
			logger:        m.logger,
		})
		if err != nil {
			// Clean up any proxies we've already created
			cleanupProxyMap(proxyMap)
			return nil, err
		}

		proxyMap[proxy.port] = proxy
		resultPortMap[proxy.port] = protocolPort

		m.logger.Infof("Started source proxy for %s (protocol %d): %s:%d -> %s", migrationUID, protocolPort, m.migrationIP, proxy.port, targetProxyAddress)
	}

	m.sourceProxies[migrationUID] = proxyMap
	m.logger.Infof("StartSourceProxies output: resultPortMap=%v", resultPortMap)
	return resultPortMap, nil
}

// Proxy run loop - accepts connections and forwards to target
func (p *migrationProxy) run() {
	defer func() {
		close(p.runExited)
		p.manager.onProxyRunExit(p.migrationUID, p.port, p.isTarget)
		p.listener.Close()
	}()

	// Set deadline on listener to allow periodic check of stopChan
	// This prevents the run loop from blocking forever on Accept() if shutdown is requested
	for {
		// Set a deadline for Accept() to allow checking stopChan periodically
		if tcpListener, ok := p.listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(proxyAcceptTimeout))
		}

		conn, err := p.listener.Accept()
		if err != nil {
			// Check if we're shutting down
			select {
			case <-p.stopChan:
				return
			default:
				// Check if this was a timeout (expected) vs a real error
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is expected, continue to next iteration
					continue
				}
				// Real error, log and exit
				p.logger.Reason(err).Error("error accepting connection on migration proxy")
				return
			}
		}
		go p.handleConnection(conn)
	}
}

func (p *migrationProxy) handleConnection(inConn net.Conn) {
	defer inConn.Close()

	// Determine proxy type for metrics
	proxyType := "source"
	if p.isTarget {
		proxyType = "target"
	}

	// Track active connection
	virthandler.DecentralizedMigrationProxyActiveConnectionsInc(proxyType)
	defer virthandler.DecentralizedMigrationProxyActiveConnectionsDec(proxyType)

	// Connect to target using plain TCP - this is a transparent proxy that forwards
	// encrypted TLS bytes as-is. The TLS handshake is end-to-end between virt-handlers,
	// not terminated at the proxy.
	outConn, err := net.DialTimeout("tcp", p.targetAddress, proxyDialTimeout)
	if err != nil {
		p.logger.Reason(err).Errorf("Failed to connect to target %s", p.targetAddress)
		virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "connection_failed")
		return
	}
	defer outConn.Close()

	// Set initial idle timeout on both connections to prevent hanging
	// This will be extended as data flows through copyWithIdleTimeout
	inConn.SetDeadline(time.Now().Add(proxyIdleTimeout))
	outConn.SetDeadline(time.Now().Add(proxyIdleTimeout))

	// Helper to signal EOF to peer by closing write side
	closeWrite := func(conn net.Conn) {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.CloseWrite()
		} else {
			conn.Close()
		}
	}

	// Bidirectional copy with byte tracking and idle timeout handling
	// Use io.Copy with a deadline-resetting reader wrapper to prevent idle hangs
	errChan := make(chan error, 2)

	go func() {
		n, err := io.Copy(outConn, NewDeadlineResettingReader(inConn, outConn, proxyIdleTimeout))
		if n > 0 {
			virthandler.DecentralizedMigrationProxyBytesTransferredAdd(proxyType, "outbound", float64(n))
		}
		if err != nil && err != io.EOF {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "idle_timeout")
			} else {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "copy_error")
			}
		}
		closeWrite(outConn)
		errChan <- err
	}()

	go func() {
		n, err := io.Copy(inConn, NewDeadlineResettingReader(outConn, inConn, proxyIdleTimeout))
		if n > 0 {
			virthandler.DecentralizedMigrationProxyBytesTransferredAdd(proxyType, "inbound", float64(n))
		}
		if err != nil && err != io.EOF {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "idle_timeout")
			} else {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "copy_error")
			}
		}
		closeWrite(inConn)
		errChan <- err
	}()

	// Wait for BOTH directions to complete
	<-errChan
	<-errChan
}

func (m *SyncProxyManager) StopSourceProxy(migrationUID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// If shutdown has already cleaned up all proxies, nothing to do
	if m.isShuttingDown {
		return
	}

	proxyMap, exists := m.sourceProxies[migrationUID]
	if !exists {
		return
	}

	// Remove from map first to establish ownership of cleanup
	// This prevents Shutdown from attempting to clean up the same proxies
	delete(m.sourceProxies, migrationUID)

	// Now we own the cleanup responsibility - safe to close resources
	cleanupProxyMap(proxyMap)

	m.logger.Infof("Stopped %d source proxy(s) for %s", len(proxyMap), migrationUID)
}

func (m *SyncProxyManager) StopTargetProxy(migrationUID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	// If shutdown has already cleaned up all proxies, nothing to do
	if m.isShuttingDown {
		return
	}

	proxyMap, exists := m.targetProxies[migrationUID]
	if !exists {
		return
	}

	// Remove from map first to establish ownership of cleanup
	// This prevents Shutdown from attempting to clean up the same proxies
	delete(m.targetProxies, migrationUID)

	// Now we own the cleanup responsibility - safe to close resources
	cleanupProxyMap(proxyMap)

	m.logger.Infof("Stopped %d target proxy(s) for %s", len(proxyMap), migrationUID)
}

// GetTargetProxyPorts returns the port map for target proxies (if they exist)
// Returns map[proxyPort]protocolPort
// getProxyPorts is a helper that extracts proxy ports from a proxy map
func getProxyPorts(proxyMap map[int]*migrationProxy) map[int]int {
	if proxyMap == nil {
		return nil
	}

	result := make(map[int]int)
	for _, proxy := range proxyMap {
		result[proxy.port] = proxy.protocolPort
	}
	return result
}

func (m *SyncProxyManager) GetTargetProxyPorts(migrationUID string) map[int]int {
	m.lock.Lock()
	defer m.lock.Unlock()

	proxyMap, exists := m.targetProxies[migrationUID]
	if !exists {
		return nil
	}

	return getProxyPorts(proxyMap)
}

func (m *SyncProxyManager) GetSourceProxyPorts(migrationUID string) map[int]int {
	m.lock.Lock()
	defer m.lock.Unlock()

	proxyMap, exists := m.sourceProxies[migrationUID]
	if !exists {
		return nil
	}

	return getProxyPorts(proxyMap)
}

func (m *SyncProxyManager) Shutdown() {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Idempotent: if already shutting down, nothing to do
	if m.isShuttingDown {
		return
	}

	m.isShuttingDown = true

	// Stop all source proxies
	for migrationUID, proxyMap := range m.sourceProxies {
		cleanupProxyMap(proxyMap)
		delete(m.sourceProxies, migrationUID)
	}

	// Stop all target proxies
	for migrationUID, proxyMap := range m.targetProxies {
		cleanupProxyMap(proxyMap)
		delete(m.targetProxies, migrationUID)
	}

	m.logger.Info("Migration proxy manager shutdown complete")
}
