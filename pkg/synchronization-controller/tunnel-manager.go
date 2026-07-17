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
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

const (
	// frameDataSize is the size of data per frame (32KB)
	frameDataSize = 32 * 1024

	// channelIdleTimeout is the maximum time a channel can be idle before being closed
	channelIdleTimeout = 5 * time.Minute
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

	// clientTLSConfig terminates TLS when dialing target virt-handler (migration client cert).
	clientTLSConfig *tls.Config
	// serverTLSConfig terminates TLS when accepting source virt-handler (virt-handler server cert).
	serverTLSConfig *tls.Config

	logger *log.FilteredLogger
}

// migrationTunnel holds source listeners or target dial info for one migration.
type migrationTunnel struct {
	migrationID string
	isSource    bool

	channels map[int32]*tunnelChannel
	mu       sync.RWMutex

	stopChan chan struct{}

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
	channelID    int32
	stream       frameStream
	cancelStream context.CancelFunc // set for client-opened streams
	conn         net.Conn           // virt-handler side (source local or target dial)
	stopChan     chan struct{}
	stopped      atomic.Bool
	sequence     uint64
	lastActivity atomic.Value // time.Time
	bytesSent    atomic.Uint64
	bytesReceived atomic.Uint64
	createdAt    time.Time
	logger       *log.FilteredLogger
}

// NewMigrationTunnelManager creates a new tunnel manager.
// clientTLSConfig is used to dial target virt-handlers; serverTLSConfig is used to
// accept TLS from source virt-handlers.
func NewMigrationTunnelManager(clientTLSConfig, serverTLSConfig *tls.Config) *MigrationTunnelManager {
	return &MigrationTunnelManager{
		tunnels:         make(map[string]*migrationTunnel),
		clientTLSConfig: clientTLSConfig,
		serverTLSConfig: serverTLSConfig,
		logger:          log.DefaultLogger(),
	}
}

// Initialize stores network IPs (called at startup if crossClusterNetwork configured)
func (m *MigrationTunnelManager) Initialize(migrationIP, crossClusterIP string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.migrationIP = migrationIP
	m.crossClusterIP = crossClusterIP

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
// and forward them to the local virt-handler ports.
func (m *MigrationTunnelManager) StartTargetTunnel(
	_ context.Context,
	migrationID string,
	targetVirtHandlerIP string,
	targetVirtHandlerPorts map[int]int,
) (*migrationTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnelKey := "target:" + migrationID
	if existing, exists := m.tunnels[tunnelKey]; exists {
		m.logger.Infof("Target tunnel already exists for migration %s", migrationID)
		return existing, nil
	}

	tunnel := &migrationTunnel{
		migrationID:     migrationID,
		isSource:        false,
		channels:        make(map[int32]*tunnelChannel),
		stopChan:        make(chan struct{}),
		listenerPorts:   make(map[int]int),
		targetIP:        targetVirtHandlerIP,
		targetPorts:     targetVirtHandlerPorts,
		clientTLSConfig: m.clientTLSConfig,
		serverTLSConfig: m.serverTLSConfig,
		logger:          m.logger,
	}
	m.tunnels[tunnelKey] = tunnel

	m.logger.Infof("Started target tunnel for migration %s → virt-handler %s ports %v",
		migrationID, targetVirtHandlerIP, targetVirtHandlerPorts)
	return tunnel, nil
}

// StartSourceTunnel creates listeners on the internal migration network for source
// virt-handler and stores the control-plane gRPC connection used to open one
// MigrationTunnel stream per channel.
func (m *MigrationTunnelManager) StartSourceTunnel(
	_ context.Context,
	migrationID string,
	grpcConn *grpc.ClientConn,
	protocolPorts map[int]int, // map[any TCP port]protocol port — values are the channels to open
) (*migrationTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnelKey := "source:" + migrationID
	if existing, exists := m.tunnels[tunnelKey]; exists {
		m.logger.Infof("Source tunnel already exists for migration %s", migrationID)
		return existing, nil
	}

	tunnel := &migrationTunnel{
		migrationID:     migrationID,
		isSource:        true,
		channels:        make(map[int32]*tunnelChannel),
		stopChan:        make(chan struct{}),
		listenerPorts:   make(map[int]int),
		grpcConn:        grpcConn,
		clientTLSConfig: m.clientTLSConfig,
		serverTLSConfig: m.serverTLSConfig,
		logger:          m.logger,
	}

	if m.serverTLSConfig == nil {
		return nil, fmt.Errorf("migration server TLS config required for source tunnel listeners")
	}

	// Deduplicate protocol ports (map values) so we open one listener per channel.
	seen := make(map[int]struct{})
	for _, protocolPort := range protocolPorts {
		if _, ok := seen[protocolPort]; ok {
			continue
		}
		seen[protocolPort] = struct{}{}

		listenAddr := net.JoinHostPort(m.migrationIP, "0")
		// Terminate TLS from source virt-handler (same as target virt-handler migration proxy).
		listener, err := tls.Listen("tcp", listenAddr, m.serverTLSConfig)
		if err != nil {
			tunnel.closeListeners()
			return nil, fmt.Errorf("failed to create TLS listener for protocol %d: %v", protocolPort, err)
		}

		listenerPort := listener.Addr().(*net.TCPAddr).Port
		tunnel.listenerPorts[listenerPort] = protocolPort
		tunnel.listeners = append(tunnel.listeners, listener)

		go tunnel.acceptConnections(listener, int32(protocolPort))

		m.logger.Infof("Source tunnel TLS listener for migration %s protocol %d: %s:%d",
			migrationID, protocolPort, m.migrationIP, listenerPort)
	}

	m.tunnels[tunnelKey] = tunnel
	return tunnel, nil
}

// GetListenerPorts returns the map of listener ports for this tunnel
func (t *migrationTunnel) GetListenerPorts() map[int]int {
	return t.listenerPorts
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

	targetPort, err := tunnel.lookupTargetPort(channelID)
	if err != nil {
		return err
	}

	targetAddr := net.JoinHostPort(tunnel.targetIP, fmt.Sprintf("%d", targetPort))
	var conn net.Conn
	if tunnel.clientTLSConfig != nil {
		conn, err = tls.Dial("tcp", targetAddr, tunnel.clientTLSConfig)
	} else {
		conn, err = net.Dial("tcp", targetAddr)
	}
	if err != nil {
		virthandler.DecentralizedMigrationProxyErrorsInc("target", "connect_error")
		_ = stream.Send(&syncv1.MigrationFrame{
			MigrationId:  migrationID,
			ChannelId:    channelID,
			FrameType:    syncv1.FrameType_FRAME_TYPE_ERROR,
			ErrorMessage: fmt.Sprintf("connection to target virt-handler refused: %v", err),
		})
		return fmt.Errorf("failed to connect to target virt-handler at %s: %v", targetAddr, err)
	}

	tunnel.logger.Infof("Target channel %d connected to virt-handler %s for migration %s",
		channelID, targetAddr, migrationID)

	return tunnel.runChannel("target", channelID, stream, nil, conn)
}

func (t *migrationTunnel) lookupTargetPort(channelID int32) (int, error) {
	for tcpPort, protocolPort := range t.targetPorts {
		if int32(protocolPort) == channelID {
			return tcpPort, nil
		}
	}
	return 0, fmt.Errorf("no target virt-handler port for channel %d", channelID)
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

	if sourceExists {
		m.stopTunnelInternal(sourceTunnel, migrationID, "source")
	}
	if targetExists {
		m.stopTunnelInternal(targetTunnel, migrationID, "target")
	}
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
	channels := make([]*tunnelChannel, 0, len(tunnel.channels))
	for _, ch := range tunnel.channels {
		channels = append(channels, ch)
	}
	tunnel.mu.Unlock()

	for _, ch := range channels {
		ch.close()
	}

	m.logger.Infof("Stopped %s tunnel for migration %s", tunnelType, migrationID)
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

		go t.handleSourceConnection(conn, protocolPort)
	}
}

// isTransientAcceptError reports OS/net errors that can clear without replacing
// the listener (EMFILE, temporary unavailability, etc.).
func isTransientAcceptError(err error) bool {
	if err == nil {
		return false
	}
	if ne, ok := err.(interface{ Temporary() bool }); ok && ne.Temporary() {
		return true
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
	if t.grpcConn == nil {
		t.logger.Errorf("No gRPC connection for source tunnel migration %s", t.migrationID)
		conn.Close()
		return
	}

	// Open the stream before registering the channel so StopTunnel can cancel it.
	streamCtx, cancel := context.WithCancel(context.Background())
	client := syncv1.NewSynchronizeClient(t.grpcConn)
	stream, err := client.MigrationTunnel(streamCtx)
	if err != nil {
		cancel()
		virthandler.DecentralizedMigrationProxyErrorsInc("source", "connect_error")
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
		virthandler.DecentralizedMigrationProxyErrorsInc("source", "send_error")
		t.logger.Reason(err).Errorf("Failed to send OPEN for channel %d", protocolPort)
		conn.Close()
		return
	}

	t.logger.Infof("Opened per-channel stream for migration %s channel %d", t.migrationID, protocolPort)

	if err := t.runChannel("source", protocolPort, stream, cancel, conn); err != nil {
		t.logger.Reason(err).Errorf("Channel %d ended with error", protocolPort)
	}
}

// runChannel proxies bytes between conn and stream until either side closes.
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

	t.mu.Lock()
	if prev, ok := t.channels[channelID]; ok {
		prev.close()
	}
	t.channels[channelID] = ch
	t.mu.Unlock()

	virthandler.DecentralizedMigrationProxyActiveConnectionsInc(proxyType)
	defer virthandler.DecentralizedMigrationProxyActiveConnectionsDec(proxyType)
	defer func() {
		ch.closeWithStats(t.migrationID, proxyType)
		t.mu.Lock()
		if current, ok := t.channels[channelID]; ok && current == ch {
			delete(t.channels, channelID)
		}
		t.mu.Unlock()
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
		if err != nil && err != io.EOF {
			return err
		}
		return nil
	}
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
			_ = c.stream.Send(closeFrame)
			if err != io.EOF {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "send_error")
				return err
			}
			return io.EOF
		}

		frame := &syncv1.MigrationFrame{
			MigrationId: migrationID,
			ChannelId:   c.channelID,
			FrameType:   syncv1.FrameType_FRAME_TYPE_DATA,
			// gRPC/protobuf serializes Data before the next Read reuses buf.
			Data:     buf[:n],
			Sequence: atomic.AddUint64(&c.sequence, 1),
		}
		if err := c.stream.Send(frame); err != nil {
			virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "send_error")
			return err
		}

		c.lastActivity.Store(time.Now())
		c.bytesSent.Add(uint64(n))
		virthandler.DecentralizedMigrationProxyBytesTransferredAdd(proxyType, "receive", float64(n))
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
			if err != io.EOF {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "send_error")
			}
			return err
		}

		switch frame.FrameType {
		case syncv1.FrameType_FRAME_TYPE_DATA:
			if _, err := c.conn.Write(frame.Data); err != nil {
				virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "send_error")
				return err
			}
			c.lastActivity.Store(time.Now())
			c.bytesReceived.Add(uint64(len(frame.Data)))
			virthandler.DecentralizedMigrationProxyBytesTransferredAdd(proxyType, "send", float64(len(frame.Data)))

		case syncv1.FrameType_FRAME_TYPE_CLOSE, syncv1.FrameType_FRAME_TYPE_ERROR:
			c.logger.Infof("Channel %d closed by remote (%v): %s", c.channelID, frame.FrameType, frame.ErrorMessage)
			return io.EOF

		case syncv1.FrameType_FRAME_TYPE_OPEN:
			// Already handled as the first frame on the target; ignore duplicates.
			c.logger.V(4).Infof("Ignoring extra OPEN on channel %d migration %s", c.channelID, migrationID)

		default:
			c.logger.Infof("Ignoring frame type %v on channel %d", frame.FrameType, c.channelID)
		}
	}
}

func (c *tunnelChannel) monitorIdle(migrationID string) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lastActivity, _ := c.lastActivity.Load().(time.Time)
			if !lastActivity.IsZero() && time.Since(lastActivity) > channelIdleTimeout {
				c.logger.Infof("Channel %d idle timeout for migration %s", c.channelID, migrationID)
				c.close()
				return
			}
		case <-c.stopChan:
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
		c.logger.Infof("Closed migration tunnel channel: migration=%s channel=%d proxy=%s duration=%v sent=%d received=%d",
			migrationID, c.channelID, proxyType, duration.Round(time.Second), c.bytesSent.Load(), c.bytesReceived.Load())
	}
	c.close()
}
