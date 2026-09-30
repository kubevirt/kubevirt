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
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"kubevirt.io/client-go/log"

	synccontrollermetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-synchronization-controller"
	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

const (
	// frameDataSize is the size of data per frame (32KB)
	frameDataSize = 32 * 1024

	// maxConcurrentChannelsPerTunnel limits parallel Accept/inbound channel handlers
	// (NBD opens multiple streams; unbounded Accept would allow FD/goroutine exhaustion).
	maxConcurrentChannelsPerTunnel = 64

	tlsHandshakeTimeout = 30 * time.Second
	targetDialTimeout   = 30 * time.Second

	defaultChannelIdleTimeout       = 5 * time.Minute
	defaultChannelIdleCheckInterval = 1 * time.Minute
)

// frameStream is a bidirectional MigrationTunnel stream (client or server side).
type frameStream interface {
	Send(*syncv1.MigrationFrame) error
	Recv() (*syncv1.MigrationFrame, error)
}

// MigrationTunnelManager manages per-channel gRPC tunnels for cross-cluster migrations.
// Each migration protocol port (channel) gets its own bidi MigrationTunnel stream on the
// existing control-plane ClientConn, avoiding head-of-line blocking across channels.
type MigrationTunnelManager struct {
	tunnels map[string]*migrationTunnel
	mu      sync.RWMutex

	migrationIP    string
	crossClusterIP string

	// tunnelPeers binds migrationID -> gRPC peer address allowed to open MigrationTunnel streams.
	tunnelPeers sync.Map

	// clientTLSConfig terminates TLS when dialing target virt-handler (migration client cert).
	clientTLSConfig *tls.Config
	// serverTLSConfig terminates TLS when accepting source virt-handler (virt-handler server cert).
	serverTLSConfig *tls.Config

	// metricsStop stops the throughput reporter started when the proxy is initialized.
	metricsStop chan struct{}

	logger *log.FilteredLogger
}

// NewMigrationTunnelManager creates a new tunnel manager.
// clientTLSConfig is used to dial target virt-handlers; serverTLSConfig is used to
// accept TLS from source virt-handlers.
func NewMigrationTunnelManager(clientTLSConfig, serverTLSConfig *tls.Config) *MigrationTunnelManager {
	return &MigrationTunnelManager{
		tunnels:         make(map[string]*migrationTunnel),
		clientTLSConfig: clientTLSConfig,
		serverTLSConfig: serverTLSConfig,
		metricsStop:     make(chan struct{}),
		logger:          log.DefaultLogger(),
	}
}

func (m *MigrationTunnelManager) Initialize(migrationIP, crossClusterIP string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.migrationIP = migrationIP
	m.crossClusterIP = crossClusterIP

	if err := synccontrollermetrics.EnsureProxyMetrics(m.metricsStop); err != nil {
		m.logger.Reason(err).Error("failed to register migration proxy metrics")
	}

	m.logger.Infof("Migration tunnel manager initialized - local: %s, peer: %s",
		migrationIP, crossClusterIP)
}

// IsInitialized reports whether local and peer tunnel IPs are set.

func (m *MigrationTunnelManager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.migrationIP != "" && m.crossClusterIP != ""
}

// MigrationIP returns the local (virt-handler-facing) tunnel address under lock.

func (m *MigrationTunnelManager) MigrationIP() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.migrationIP
}

// CrossClusterIP returns the peer-facing tunnel / advertise address under lock.

func (m *MigrationTunnelManager) CrossClusterIP() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.crossClusterIP
}

// StartTargetTunnel prepares the target side to accept per-channel streams from the source
// and forward them to the local virt-handler ports. Lifecycle is owned by stopChan /
// StopTunnel, not a caller context.

func (m *MigrationTunnelManager) StartTargetTunnel(
	migrationID string,
	targetVirtHandlerIP string,
	targetVirtHandlerPorts map[int]int,
) (*migrationTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnelKey := "target:" + migrationID
	portsCopy := copyPortMap(targetVirtHandlerPorts)
	if existing, exists := m.tunnels[tunnelKey]; exists {
		// Refresh dial info — virt-handler IP/ports can change across reconciles.
		existing.mu.Lock()
		existing.targetIP = targetVirtHandlerIP
		existing.targetPorts = portsCopy
		existing.mu.Unlock()
		m.logger.V(4).Infof("Refreshed target tunnel for migration %s → virt-handler %s ports %v",
			migrationID, targetVirtHandlerIP, targetVirtHandlerPorts)
		return existing, nil
	}

	tunnel := &migrationTunnel{
		migrationID:     migrationID,
		isSource:        false,
		stopChan:        make(chan struct{}),
		channelSem:      make(chan struct{}, maxConcurrentChannelsPerTunnel),
		listenerPorts:   make(map[int]int),
		targetIP:        targetVirtHandlerIP,
		targetPorts:     portsCopy,
		clientTLSConfig: m.clientTLSConfig,
		serverTLSConfig: m.serverTLSConfig,
		logger:          m.logger,
	}
	m.tunnels[tunnelKey] = tunnel

	m.logger.V(3).Infof("Started target tunnel for migration %s → virt-handler %s ports %v",
		migrationID, targetVirtHandlerIP, targetVirtHandlerPorts)
	return tunnel, nil
}

// StartSourceTunnel creates listeners on the internal migration network for source
// virt-handler and stores the control-plane gRPC connection used to open one
// MigrationTunnel stream per channel. Lifecycle is owned by stopChan / StopTunnel,
// not a caller context.

func (m *MigrationTunnelManager) StartSourceTunnel(
	migrationID string,
	grpcConn *grpc.ClientConn,
	protocolPorts map[int]int, // map[any TCP port]protocol port — values are the channels to open
) (*migrationTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.serverTLSConfig == nil {
		return nil, fmt.Errorf("migration server TLS config required for source tunnel listeners")
	}

	tunnelKey := "source:" + migrationID
	if existing, exists := m.tunnels[tunnelKey]; exists {
		// Refresh the shared control-plane conn when SyncAddress was re-dialed,
		// and open listeners for any new protocol ports (e.g. hotplug/NBD growth).
		existing.mu.Lock()
		existing.grpcConn = grpcConn
		existing.mu.Unlock()
		if err := m.addSourceListenersLocked(existing, protocolPorts); err != nil {
			return nil, err
		}
		m.logger.V(4).Infof("Refreshed source tunnel for migration %s (ports %v)", migrationID, existing.GetListenerPorts())
		return existing, nil
	}

	tunnel := &migrationTunnel{
		migrationID:     migrationID,
		isSource:        true,
		stopChan:        make(chan struct{}),
		channelSem:      make(chan struct{}, maxConcurrentChannelsPerTunnel),
		listenerPorts:   make(map[int]int),
		grpcConn:        grpcConn,
		clientTLSConfig: m.clientTLSConfig,
		serverTLSConfig: m.serverTLSConfig,
		logger:          m.logger,
	}

	if err := m.addSourceListenersLocked(tunnel, protocolPorts); err != nil {
		tunnel.closeListeners()
		return nil, err
	}

	m.tunnels[tunnelKey] = tunnel
	return tunnel, nil
}

// addSourceListenersLocked opens TLS listeners for protocol ports not already present.
// Caller must hold m.mu.

func (m *MigrationTunnelManager) addSourceListenersLocked(tunnel *migrationTunnel, protocolPorts map[int]int) error {
	tunnel.mu.Lock()
	existingProtocols := make(map[int]struct{}, len(tunnel.listenerPorts))
	for _, protocolPort := range tunnel.listenerPorts {
		existingProtocols[protocolPort] = struct{}{}
	}
	tunnel.mu.Unlock()

	seen := make(map[int]struct{})
	for _, protocolPort := range protocolPorts {
		if _, ok := seen[protocolPort]; ok {
			continue
		}
		seen[protocolPort] = struct{}{}
		if _, ok := existingProtocols[protocolPort]; ok {
			continue
		}

		listenAddr := net.JoinHostPort(m.migrationIP, "0")
		listener, err := tls.Listen("tcp", listenAddr, m.serverTLSConfig.Clone())
		if err != nil {
			return fmt.Errorf("failed to create TLS listener for protocol %d: %v", protocolPort, err)
		}

		listenerPort := listener.Addr().(*net.TCPAddr).Port
		tunnel.mu.Lock()
		tunnel.listenerPorts[listenerPort] = protocolPort
		tunnel.listeners = append(tunnel.listeners, listener)
		tunnel.mu.Unlock()

		go tunnel.acceptConnections(listener, int32(protocolPort))

		m.logger.V(3).Infof("Source tunnel TLS listener for migration %s protocol %d: %s:%d",
			tunnel.migrationID, protocolPort, m.migrationIP, listenerPort)
	}
	return nil
}

// BindTunnelPeer records the gRPC peer host allowed to open MigrationTunnel streams
// for this migration (typically from a validated SyncSourceMigrationStatus).
// If a peer is already bound, a different host is rejected.

func (m *MigrationTunnelManager) BindTunnelPeer(migrationID, peerAddr string) error {
	host := peerHost(peerAddr)
	if migrationID == "" || host == "" {
		return fmt.Errorf("missing migration ID or peer address for tunnel binding")
	}
	if existing, loaded := m.tunnelPeers.LoadOrStore(migrationID, host); loaded {
		expected, _ := existing.(string)
		if expected != "" && expected != host {
			return status.Errorf(codes.PermissionDenied,
				"peer %q not authorized to bind migration %s (expected %q)", host, migrationID, expected)
		}
		return nil
	}
	m.logger.V(4).Infof("Bound tunnel peer %q for migration %s", host, migrationID)
	return nil
}

// AuthorizeTunnelPeer rejects MigrationTunnel streams that are not from the peer
// previously bound for this migration. Unbound migrations are rejected fail-closed.

func (m *MigrationTunnelManager) AuthorizeTunnelPeer(migrationID, peerAddr string) error {
	host := peerHost(peerAddr)
	if migrationID == "" {
		return fmt.Errorf("missing migration ID for tunnel authorization")
	}
	if host == "" {
		return status.Errorf(codes.Unauthenticated, "missing peer address for migration tunnel %s", migrationID)
	}
	existing, ok := m.tunnelPeers.Load(migrationID)
	if !ok {
		return status.Errorf(codes.FailedPrecondition,
			"no peer bound for migration %s; SyncSourceMigrationStatus must run first", migrationID)
	}
	expected, _ := existing.(string)
	if expected != host {
		return status.Errorf(codes.PermissionDenied,
			"peer %q not authorized for migration %s (expected %q)", host, migrationID, expected)
	}
	return nil
}

func (m *MigrationTunnelManager) HandleInboundChannel(stream frameStream, openFrame *syncv1.MigrationFrame) error {
	if openFrame.FrameType != syncv1.FrameType_FRAME_TYPE_OPEN {
		return fmt.Errorf("expected OPEN frame, got %v", openFrame.FrameType)
	}

	migrationID := openFrame.MigrationId
	channelID := openFrame.ChannelId

	m.mu.RLock()
	tunnel := m.tunnels["target:"+migrationID]
	m.mu.RUnlock()
	if tunnel == nil {
		return fmt.Errorf("target tunnel not found for migration %s", migrationID)
	}

	if !tunnel.tryAcquireChannelSlot() {
		return status.Errorf(codes.ResourceExhausted,
			"too many concurrent channels for migration %s", migrationID)
	}
	defer tunnel.releaseChannelSlot()

	// Each inbound stream is its own TCP connection to virt-handler (NBD may open
	// several on protocol port 49153). Do not serialize on channelID.
	now := time.Now()
	ch := &tunnelChannel{
		channelID: channelID,
		stream:    stream,
		stopChan:  make(chan struct{}),
		createdAt: now,
		logger:    tunnel.logger,
	}
	ch.lastActivity.Store(now)
	ch.sequence = 1
	tunnel.addChannel(ch)

	targetIP, targetPort, err := tunnel.lookupTargetDial(channelID)
	if err != nil {
		tunnel.removeChannel(ch)
		return err
	}

	targetAddr := net.JoinHostPort(targetIP, fmt.Sprintf("%d", targetPort))
	if tunnel.clientTLSConfig == nil {
		tunnel.removeChannel(ch)
		synccontrollermetrics.ErrorsInc("target", "connect_error")
		_ = stream.Send(&syncv1.MigrationFrame{
			MigrationId:  migrationID,
			ChannelId:    channelID,
			FrameType:    syncv1.FrameType_FRAME_TYPE_ERROR,
			ErrorMessage: "migration client TLS config required to dial target virt-handler",
		})
		return fmt.Errorf("migration client TLS config required to dial target virt-handler at %s", targetAddr)
	}
	dialer := &net.Dialer{Timeout: targetDialTimeout}
	// Clone so concurrent dials do not share mutable session state.
	conn, err := tls.DialWithDialer(dialer, "tcp", targetAddr, tunnel.clientTLSConfig.Clone())
	if err != nil {
		tunnel.removeChannel(ch)
		synccontrollermetrics.ErrorsInc("target", "connect_error")
		_ = stream.Send(&syncv1.MigrationFrame{
			MigrationId:  migrationID,
			ChannelId:    channelID,
			FrameType:    syncv1.FrameType_FRAME_TYPE_ERROR,
			ErrorMessage: fmt.Sprintf("connection to target virt-handler refused: %v", err),
		})
		return fmt.Errorf("failed to connect to target virt-handler at %s: %v", targetAddr, err)
	}
	ch.conn = conn

	tunnel.logger.V(4).Infof("Target channel %d connected to virt-handler %s for migration %s",
		channelID, targetAddr, migrationID)

	return tunnel.runClaimedChannel("target", ch)
}

func (m *MigrationTunnelManager) StopTunnel(migrationID string) {
	m.mu.Lock()
	sourceTunnel, sourceExists := m.tunnels["source:"+migrationID]
	targetTunnel, targetExists := m.tunnels["target:"+migrationID]
	if sourceExists {
		delete(m.tunnels, "source:"+migrationID)
	}
	if targetExists {
		delete(m.tunnels, "target:"+migrationID)
	}
	m.mu.Unlock()

	m.tunnelPeers.Delete(migrationID)

	if sourceExists {
		m.stopTunnelInternal(sourceTunnel, migrationID, "source")
	}
	if targetExists {
		m.stopTunnelInternal(targetTunnel, migrationID, "target")
	}
	synccontrollermetrics.ClearMigrationBytes(migrationID)
}

func (m *MigrationTunnelManager) stopTunnelInternal(tunnel *migrationTunnel, migrationID, tunnelType string) {
	select {
	case <-tunnel.stopChan:
		// already stopped
	default:
		close(tunnel.stopChan)
	}

	tunnel.closeListeners()

	tunnel.mu.Lock()
	channels := append([]*tunnelChannel(nil), tunnel.channels...)
	tunnel.mu.Unlock()

	for _, ch := range channels {
		ch.close()
	}

	m.logger.V(3).Infof("Stopped %s tunnel for migration %s", tunnelType, migrationID)
}

// Shutdown stops all tunnels

func (m *MigrationTunnelManager) Shutdown() {
	m.mu.Lock()
	migrationIDs := make(map[string]struct{})
	for key := range m.tunnels {
		id := key
		if after, ok := stripTunnelKeyPrefix(key); ok {
			id = after
		}
		migrationIDs[id] = struct{}{}
	}
	m.mu.Unlock()

	for migrationID := range migrationIDs {
		m.StopTunnel(migrationID)
	}

	select {
	case <-m.metricsStop:
	default:
		close(m.metricsStop)
	}

	m.logger.Info("Migration tunnel manager shutdown complete")
}

func peerHost(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func copyPortMap(in map[int]int) map[int]int {
	out := make(map[int]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// GetListenerPorts returns a copy of the map of listener ports for this tunnel

func stripTunnelKeyPrefix(key string) (string, bool) {
	switch {
	case strings.HasPrefix(key, "source:"):
		return strings.TrimPrefix(key, "source:"), true
	case strings.HasPrefix(key, "target:"):
		return strings.TrimPrefix(key, "target:"), true
	default:
		return key, false
	}
}
