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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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

// migrationTunnel holds source listeners or target dial info for one migration.
type migrationTunnel struct {
	migrationID string
	isSource    bool

	// Active TCP↔gRPC proxies. NBD (49153) may open multiple connections on the
	// same protocol port, so this is a list — not one slot per protocol port.
	channels []*tunnelChannel
	mu       sync.Mutex

	stopChan chan struct{}

	// channelSem limits concurrent channel handlers for this tunnel.
	channelSem chan struct{}

	// Source: listeners for virt-handler and the shared control-plane conn.
	listenerPorts map[int]int // map[listen TCP port]protocol port
	listeners     []net.Listener
	grpcConn      *grpc.ClientConn

	// Target: where to forward opened channels.
	targetIP    string
	targetPorts map[int]int // map[TCP port]protocol port

	clientTLSConfig *tls.Config
	serverTLSConfig *tls.Config
	logger          *log.FilteredLogger
}

// tunnelChannel is one migration protocol connection with its own gRPC stream.
type tunnelChannel struct {
	channelID     int32
	stream        frameStream
	cancelStream  context.CancelFunc // set for client-opened streams
	conn          net.Conn           // virt-handler side (source local or target dial)
	stopChan      chan struct{}
	stopped       atomic.Bool
	sequence      uint64
	lastActivity  atomic.Value // time.Time
	bytesSent     atomic.Uint64
	bytesReceived atomic.Uint64
	createdAt     time.Time
	logger        *log.FilteredLogger

	// Optional overrides for unit tests; zero means use package defaults.
	idleTimeout       time.Duration
	idleCheckInterval time.Duration
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

// Initialize stores network IPs (called at startup if crossClusterNetwork configured)
// and registers proxy metrics once the proxy is actually enabled.
func (m *MigrationTunnelManager) Initialize(migrationIP, crossClusterIP string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.migrationIP = migrationIP
	m.crossClusterIP = crossClusterIP

	if err := synccontrollermetrics.EnsureProxyMetrics(m.metricsStop); err != nil {
		m.logger.Reason(err).Error("failed to register migration proxy metrics")
	}

	m.logger.Infof("Migration tunnel manager initialized - migration0: %s, crosscluster0: %s",
		migrationIP, crossClusterIP)
}

// IsInitialized reports whether both migration0 and crosscluster0 IPs are set.
func (m *MigrationTunnelManager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.migrationIP != "" && m.crossClusterIP != ""
}

// MigrationIP returns the migration0 address under lock.
func (m *MigrationTunnelManager) MigrationIP() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.migrationIP
}

// CrossClusterIP returns the crosscluster0 address under lock.
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
func (t *migrationTunnel) GetListenerPorts() map[int]int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return copyPortMap(t.listenerPorts)
}

// HandleInboundChannel serves a MigrationTunnel stream opened by the source for one channel.
// Runs until the stream or underlying virt-handler connection closes.
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

func (t *migrationTunnel) lookupTargetDial(channelID int32) (string, int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for tcpPort, protocolPort := range t.targetPorts {
		if int32(protocolPort) == channelID {
			return t.targetIP, tcpPort, nil
		}
	}
	return "", 0, fmt.Errorf("no target virt-handler port for channel %d", channelID)
}

func (t *migrationTunnel) tryAcquireChannelSlot() bool {
	if t.channelSem == nil {
		return true
	}
	select {
	case t.channelSem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (t *migrationTunnel) releaseChannelSlot() {
	if t.channelSem == nil {
		return
	}
	select {
	case <-t.channelSem:
	default:
	}
}

// StopTunnel stops source and target tunnels for this migration
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

func (t *migrationTunnel) closeListeners() {
	t.mu.Lock()
	listeners := t.listeners
	t.listeners = nil
	t.listenerPorts = make(map[int]int)
	t.mu.Unlock()

	for _, l := range listeners {
		_ = l.Close()
	}
}

func (t *migrationTunnel) acceptConnections(listener net.Listener, protocolPort int32) {
	defer listener.Close()

	var tempDelay time.Duration
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-t.stopChan:
				return
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if isTransientAcceptError(err) {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
					if tempDelay > time.Second {
						tempDelay = time.Second
					}
				}
				t.logger.Reason(err).Warningf("Transient accept error for migration %s channel %d; retrying in %v",
					t.migrationID, protocolPort, tempDelay)
				timer := time.NewTimer(tempDelay)
				select {
				case <-timer.C:
				case <-t.stopChan:
					timer.Stop()
					return
				}
				continue
			}
			t.logger.Reason(err).Errorf("Error accepting connection for migration %s channel %d",
				t.migrationID, protocolPort)
			return
		}
		tempDelay = 0

		select {
		case <-t.stopChan:
			conn.Close()
			return
		default:
		}

		if !t.tryAcquireChannelSlot() {
			t.logger.Warningf("Rejecting connection for migration %s channel %d: concurrency limit reached",
				t.migrationID, protocolPort)
			_ = conn.Close()
			synccontrollermetrics.ErrorsInc("source", "connect_error")
			continue
		}

		go func(c net.Conn) {
			defer t.releaseChannelSlot()
			t.handleSourceConnection(c, protocolPort)
		}(conn)
	}
}

// isTransientAcceptError reports OS/net errors that can clear without replacing
// the listener (EMFILE, temporary unavailability, etc.).
func isTransientAcceptError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.EAGAIN, syscall.EINTR, syscall.EMFILE, syscall.ENFILE, syscall.ENOBUFS:
			return true
		}
		// EWOULDBLOCK may alias EAGAIN (Linux); keep a separate check for other platforms.
		if errno == syscall.EWOULDBLOCK {
			return true
		}
	}
	return false
}

func (t *migrationTunnel) handleSourceConnection(conn net.Conn, protocolPort int32) {
	t.mu.Lock()
	grpcConn := t.grpcConn
	t.mu.Unlock()
	if grpcConn == nil {
		t.logger.Errorf("No gRPC connection for source tunnel migration %s", t.migrationID)
		conn.Close()
		return
	}

	// Bound the handshake so slow/malicious clients cannot hold Accept slots forever.
	_ = conn.SetDeadline(time.Now().Add(tlsHandshakeTimeout))
	// Complete TLS before opening the gRPC stream / dialing the target. Otherwise
	// target-side traffic can close the conn mid-handshake.
	if tlsConn, ok := conn.(*tls.Conn); ok {
		if err := tlsConn.Handshake(); err != nil {
			t.logger.Reason(err).Errorf("TLS handshake failed for migration %s channel %d from %s",
				t.migrationID, protocolPort, conn.RemoteAddr())
			conn.Close()
			synccontrollermetrics.ErrorsInc("source", "connect_error")
			return
		}
	}
	_ = conn.SetDeadline(time.Time{})

	// Open the stream before registering the channel so StopTunnel can cancel it.
	streamCtx, cancel := context.WithCancel(context.Background())
	client := syncv1.NewSynchronizeClient(grpcConn)
	stream, err := client.MigrationTunnel(streamCtx)
	if err != nil {
		cancel()
		synccontrollermetrics.ErrorsInc("source", "connect_error")
		t.logger.Reason(err).Errorf("Failed to open MigrationTunnel stream for channel %d", protocolPort)
		conn.Close()
		return
	}

	openFrame := &syncv1.MigrationFrame{
		MigrationId: t.migrationID,
		ChannelId:   protocolPort,
		FrameType:   syncv1.FrameType_FRAME_TYPE_OPEN,
		Sequence:    1,
	}
	if err := stream.Send(openFrame); err != nil {
		cancel()
		synccontrollermetrics.ErrorsInc("source", "send_error")
		t.logger.Reason(err).Errorf("Failed to send OPEN for channel %d", protocolPort)
		conn.Close()
		return
	}

	t.logger.V(4).Infof("Opened per-channel stream for migration %s channel %d", t.migrationID, protocolPort)

	if err := t.runChannel("source", protocolPort, stream, cancel, conn); err != nil {
		t.logger.Reason(err).Errorf("Channel %d ended with error", protocolPort)
	}
}

// runChannel registers conn on a dedicated stream and proxies until the channel ends.
func (t *migrationTunnel) runChannel(
	proxyType string,
	channelID int32,
	stream frameStream,
	cancelStream context.CancelFunc,
	conn net.Conn,
) error {
	now := time.Now()
	ch := &tunnelChannel{
		channelID:    channelID,
		stream:       stream,
		cancelStream: cancelStream,
		conn:         conn,
		stopChan:     make(chan struct{}),
		createdAt:    now,
		logger:       t.logger,
	}
	ch.lastActivity.Store(now)
	ch.sequence = 1
	t.addChannel(ch)

	return t.runClaimedChannel(proxyType, ch)
}

// runClaimedChannel proxies an already-registered channel until either side closes.
func (t *migrationTunnel) runClaimedChannel(proxyType string, ch *tunnelChannel) error {
	synccontrollermetrics.ActiveConnectionsInc(proxyType)
	defer synccontrollermetrics.ActiveConnectionsDec(proxyType)
	defer func() {
		ch.closeWithStats(t.migrationID, proxyType)
		t.removeChannel(ch)
	}()

	errChan := make(chan error, 2)
	go func() {
		errChan <- ch.proxyConnToStream(t.migrationID, proxyType)
	}()
	go func() {
		errChan <- ch.proxyStreamToConn(t.migrationID, proxyType)
	}()
	go ch.monitorIdle(t.migrationID)

	select {
	case <-t.stopChan:
		ch.close()
		return nil
	case err := <-errChan:
		ch.close()
		// Drain the other side without blocking forever
		select {
		case <-errChan:
		case <-time.After(time.Second):
		}
		if err != nil && !isExpectedProxyCloseErr(err) {
			return err
		}
		return nil
	}
}

// isExpectedProxyCloseErr reports errors that are normal when a tunnel channel
// tears down (peer CLOSE/EOF, local close of the TCP conn or gRPC stream).
// For joined errors, every component must be an expected close.
func isExpectedProxyCloseErr(err error) bool {
	if err == nil {
		return true
	}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		errs := multi.Unwrap()
		if len(errs) == 0 {
			return true
		}
		for _, e := range errs {
			if !isExpectedProxyCloseErr(e) {
				return false
			}
		}
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, context.Canceled) {
		return true
	}
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Canceled, codes.Unavailable:
			return true
		}
	}
	// net.OpError / older stdlib strings that do not always unwrap to net.ErrClosed
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection")
}

func (c *tunnelChannel) proxyConnToStream(migrationID, proxyType string) error {
	buf := make([]byte, frameDataSize)
	for {
		select {
		case <-c.stopChan:
			return nil
		default:
		}

		n, err := c.conn.Read(buf)
		if err != nil {
			closeFrame := &syncv1.MigrationFrame{
				MigrationId: migrationID,
				ChannelId:   c.channelID,
				FrameType:   syncv1.FrameType_FRAME_TYPE_CLOSE,
				Sequence:    atomic.AddUint64(&c.sequence, 1),
			}
			// Keep the Read error as the cause of teardown; Join any CLOSE Send
			// failure so it is not silently discarded.
			err = errors.Join(err, c.stream.Send(closeFrame))
			if !isExpectedProxyCloseErr(err) && !c.stopped.Load() {
				synccontrollermetrics.ErrorsInc(proxyType, "send_error")
			}
			return err
		}

		frame := &syncv1.MigrationFrame{
			MigrationId: migrationID,
			ChannelId:   c.channelID,
			FrameType:   syncv1.FrameType_FRAME_TYPE_DATA,
			Data:        append([]byte(nil), buf[:n]...),
			Sequence:    atomic.AddUint64(&c.sequence, 1),
		}
		if err := c.stream.Send(frame); err != nil {
			if !isExpectedProxyCloseErr(err) && !c.stopped.Load() {
				synccontrollermetrics.ErrorsInc(proxyType, "send_error")
			}
			return err
		}

		c.lastActivity.Store(time.Now())
		c.bytesSent.Add(uint64(n))
		// Source: count once as payload enters the tunnel from virt-handler.
		if proxyType == "source" {
			if channelType := synccontrollermetrics.ChannelTypeFromID(c.channelID); channelType != "" {
				synccontrollermetrics.BytesTransferredAdd(migrationID, proxyType, channelType, float64(n))
			}
		}
	}
}

func (c *tunnelChannel) proxyStreamToConn(migrationID, proxyType string) error {
	for {
		select {
		case <-c.stopChan:
			return nil
		default:
		}

		frame, err := c.stream.Recv()
		if err != nil {
			if !isExpectedProxyCloseErr(err) && !c.stopped.Load() {
				synccontrollermetrics.ErrorsInc(proxyType, "recv_error")
			}
			return err
		}

		switch frame.FrameType {
		case syncv1.FrameType_FRAME_TYPE_DATA:
			if _, err := c.conn.Write(frame.Data); err != nil {
				if !isExpectedProxyCloseErr(err) && !c.stopped.Load() {
					synccontrollermetrics.ErrorsInc(proxyType, "send_error")
				}
				return err
			}
			c.lastActivity.Store(time.Now())
			c.bytesReceived.Add(uint64(len(frame.Data)))
			// Target: count once as payload leaves the tunnel toward virt-handler.
			if proxyType == "target" {
				if channelType := synccontrollermetrics.ChannelTypeFromID(c.channelID); channelType != "" {
					synccontrollermetrics.BytesTransferredAdd(migrationID, proxyType, channelType, float64(len(frame.Data)))
				}
			}

		case syncv1.FrameType_FRAME_TYPE_CLOSE, syncv1.FrameType_FRAME_TYPE_ERROR:
			c.logger.V(4).Infof("Channel %d closed by remote (%v): %s", c.channelID, frame.FrameType, frame.ErrorMessage)
			return io.EOF

		case syncv1.FrameType_FRAME_TYPE_OPEN:
			// Already handled as the first frame on the target; ignore duplicates.
			c.logger.V(4).Infof("Ignoring extra OPEN on channel %d migration %s", c.channelID, migrationID)

		default:
			c.logger.V(4).Infof("Ignoring frame type %v on channel %d", frame.FrameType, c.channelID)
		}
	}
}

func (c *tunnelChannel) monitorIdle(migrationID string) {
	idleTimeout := c.idleTimeout
	if idleTimeout == 0 {
		idleTimeout = defaultChannelIdleTimeout
	}
	checkInterval := c.idleCheckInterval
	if checkInterval == 0 {
		checkInterval = defaultChannelIdleCheckInterval
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lastActivity, _ := c.lastActivity.Load().(time.Time)
			if !lastActivity.IsZero() && time.Since(lastActivity) > idleTimeout {
				c.logger.V(3).Infof("Channel %d idle timeout for migration %s", c.channelID, migrationID)
				c.close()
				return
			}
		case <-c.stopChan:
			return
		}
	}
}

func (c *tunnelChannel) isActive() bool {
	return c != nil && !c.stopped.Load()
}

func (t *migrationTunnel) addChannel(ch *tunnelChannel) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.channels = append(t.channels, ch)
}

func (t *migrationTunnel) removeChannel(ch *tunnelChannel) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, existing := range t.channels {
		if existing == ch {
			t.channels = append(t.channels[:i], t.channels[i+1:]...)
			return
		}
	}
}

func (c *tunnelChannel) close() {
	if !c.stopped.CompareAndSwap(false, true) {
		return
	}
	close(c.stopChan)
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.cancelStream != nil {
		c.cancelStream()
	}
	if clientStream, ok := c.stream.(syncv1.Synchronize_MigrationTunnelClient); ok {
		_ = clientStream.CloseSend()
	}
}

func (c *tunnelChannel) closeWithStats(migrationID, proxyType string) {
	if !c.stopped.Load() {
		duration := time.Since(c.createdAt)
		c.logger.V(3).Infof("Closed migration tunnel channel: migration=%s channel=%d proxy=%s duration=%v sent=%d received=%d",
			migrationID, c.channelID, proxyType, duration.Round(time.Second), c.bytesSent.Load(), c.bytesReceived.Load())
	}
	c.close()
}
