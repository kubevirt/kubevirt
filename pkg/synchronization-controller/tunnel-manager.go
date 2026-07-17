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
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	virthandler "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler"
	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

const (
	// frameDataSize is the size of data per frame (32KB)
	frameDataSize = 32 * 1024

	// channelBufferSize is the buffer size for channel queues
	channelBufferSize = 16

	// channelIdleTimeout is the maximum time a channel can be idle before being closed
	channelIdleTimeout = 5 * time.Minute

	// frameWriteTimeout is the timeout for writing a frame to the send queue
	frameWriteTimeout = 30 * time.Second
)

// MigrationTunnelManager manages gRPC tunnels for cross-cluster migrations
type MigrationTunnelManager struct {
	// Tunnels keyed by migrationID
	tunnels map[string]*migrationTunnel
	mu      sync.RWMutex

	// Network IPs
	migrationIP    string
	crossClusterIP string

	// TLS configuration for connecting to virt-handler
	tlsConfig *tls.Config

	logger *log.FilteredLogger
}

// sendStream is an interface for sending frames
type sendStream interface {
	Send(*syncv1.MigrationFrame) error
}

// recvStream is an interface for receiving frames
type recvStream interface {
	Recv() (*syncv1.MigrationFrame, error)
}

// migrationTunnel represents dual unidirectional gRPC streams for a migration
// Uses two streams: one for sending (on our outbound connection) and one for receiving (on peer's outbound connection)
type migrationTunnel struct {
	migrationID string
	sendStream  sendStream // Stream we opened to send frames to peer
	recvStream  recvStream // Stream peer opened to send frames to us
	isSource    bool       // true if source tunnel, false if target tunnel

	// Channels keyed by channel_id (protocol port: 0, 49152, 49153)
	channels map[int32]*tunnelChannel
	mu       sync.RWMutex

	// Communication channels
	sendChan chan *syncv1.MigrationFrame
	errChan  chan error
	stopChan chan struct{}

	// Context cancellation for send stream
	cancelSendStream context.CancelFunc

	// Listener ports (for virt-handler connections)
	listenerPorts map[int]int // map[TCP port]protocol port

	// Target virt-handler connection info (for target tunnel)
	targetIP    string
	targetPorts map[int]int // map[TCP port]protocol port

	// TLS configuration for connecting to target virt-handler
	tlsConfig *tls.Config

	logger *log.FilteredLogger
}

// tunnelChannel represents a single multiplexed channel within a tunnel
type tunnelChannel struct {
	channelID     int32
	localConn     net.Conn // Incoming connection (from source virt-handler)
	targetConn    net.Conn // Outgoing connection (to target virt-handler), only for target tunnel
	sendQueue     chan []byte
	recvQueue     chan []byte
	sequence      uint64
	lastActivity  atomic.Value // time.Time
	stopChan      chan struct{}
	stopped       atomic.Bool
	mu            sync.Mutex
	bytesSent     atomic.Uint64 // bytes sent to remote peer (source→target or target→source)
	bytesReceived atomic.Uint64 // bytes received from remote peer
	createdAt     time.Time
}

// NewMigrationTunnelManager creates a new tunnel manager
func NewMigrationTunnelManager(tlsConfig *tls.Config) *MigrationTunnelManager {
	return &MigrationTunnelManager{
		tunnels:   make(map[string]*migrationTunnel),
		tlsConfig: tlsConfig,
		logger:    log.DefaultLogger(),
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
	ctx context.Context,
	migrationID string,
	targetVirtHandlerIP string,
	targetVirtHandlerPorts map[int]int, // map[TCP port]protocol port
) (*migrationTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use "target:" prefix to distinguish from source tunnel
	tunnelKey := "target:" + migrationID

	// Check if tunnel already exists
	if existing, exists := m.tunnels[tunnelKey]; exists {
		m.logger.Infof("Target tunnel already exists for migration %s", migrationID)
		return existing, nil
	}

	// Create tunnel structure
	// sendStream and recvStream will be set when streams are established
	tunnel := &migrationTunnel{
		migrationID:   migrationID,
		sendStream:    nil, // Will be set when we open stream on our outbound connection to target virt-handler
		recvStream:    nil, // Will be set when source virt-handler opens stream on its connection to us
		isSource:      false,
		channels:      make(map[int32]*tunnelChannel),
		sendChan:      make(chan *syncv1.MigrationFrame, channelBufferSize),
		errChan:       make(chan error, 1),
		stopChan:      make(chan struct{}),
		listenerPorts: make(map[int]int), // only used for source tunnel, we open these when we have received the target virt-handler ports.
		targetIP:      targetVirtHandlerIP,
		targetPorts:   targetVirtHandlerPorts,
		tlsConfig:     m.tlsConfig,
		logger:        m.logger,
	}

	// Store tunnel with "target:" prefix
	m.tunnels[tunnelKey] = tunnel

	// Start source receiving frames handler. (source -> target)
	go tunnel.sourceToTarget()
	// recvStream will be set later via SetReceiveStream, which starts runReceive

	return tunnel, nil
}

// StartSourceTunnel creates a source-side tunnel for outbound migration
// Connects to target sync controller's crosscluster0 via existing gRPC connection
// Creates listeners on migration0 for source virt-handler to connect to
func (m *MigrationTunnelManager) StartSourceTunnel(
	ctx context.Context,
	migrationID string,
	grpcConn *grpc.ClientConn,
	targetProxyPorts map[int]int, // map[TCP port]protocol port from target sync controller
) (*migrationTunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use "source:" prefix to distinguish from target tunnel
	tunnelKey := "source:" + migrationID

	// Check if tunnel already exists
	if existing, exists := m.tunnels[tunnelKey]; exists {
		m.logger.Infof("Source tunnel already exists for migration %s", migrationID)
		return existing, nil
	}

	// Create outbound gRPC stream for SENDING to target
	// Use a cancellable context that lives until the tunnel is stopped
	// Do NOT use a timeout - the stream must stay open for the entire migration
	streamCtx, cancel := context.WithCancel(context.Background())
	client := syncv1.NewSynchronizeClient(grpcConn)
	sendStream, err := client.MigrationTunnel(streamCtx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create outbound migration tunnel stream: %v", err)
	}

	// Create tunnel structure
	// recvStream will be set when target calls MigrationTunnel on its connection back to us
	tunnel := &migrationTunnel{
		migrationID:      migrationID,
		sendStream:       sendStream,
		cancelSendStream: cancel,
		recvStream:       nil, // Will be set by SetReceiveStream when target opens its stream
		isSource:         true,
		channels:         make(map[int32]*tunnelChannel),
		sendChan:         make(chan *syncv1.MigrationFrame, channelBufferSize),
		errChan:          make(chan error, 1),
		stopChan:         make(chan struct{}),
		listenerPorts:    make(map[int]int),
		tlsConfig:        m.tlsConfig,
		logger:           m.logger,
	}

	// Create listeners on migration0 for source virt-handler
	for targetProxyPort, protocolPort := range targetProxyPorts {
		listenAddr := net.JoinHostPort(m.migrationIP, "0") // OS-allocated port
		listener, err := net.Listen("tcp", listenAddr)
		if err != nil {
			// Clean up
			if sendStream != nil {
				sendStream.CloseSend()
			}
			for _, ch := range tunnel.channels {
				if ch.localConn != nil {
					ch.localConn.Close()
				}
			}
			return nil, fmt.Errorf("failed to create listener: %v", err)
		}

		tcpAddr := listener.Addr().(*net.TCPAddr)
		listenerPort := tcpAddr.Port

		tunnel.listenerPorts[listenerPort] = protocolPort

		// Start accepting connections
		go tunnel.acceptConnections(listener, int32(protocolPort), "", targetProxyPort)

		m.logger.Infof("Source tunnel listener for migration %s protocol %d: %s:%d (target proxy port: %d)",
			migrationID, protocolPort, m.migrationIP, listenerPort, targetProxyPort)
	}

	// Store tunnel with "source:" prefix
	m.tunnels[tunnelKey] = tunnel

	// Start stream handlers
	go tunnel.sourceToTarget()
	// recvStream will be set via SetReceiveStream when target opens its stream to us

	return tunnel, nil
}

// GetListenerPorts returns the map of listener ports for this tunnel
func (t *migrationTunnel) GetListenerPorts() map[int]int {
	return t.listenerPorts
}

// SetReceiveStream sets the receive stream (called when peer opens their send stream to us)
// isSource indicates if this is being called on the source side (true) or target side (false)
func (m *MigrationTunnelManager) SetReceiveStream(migrationID string, stream recvStream, isSource bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use appropriate prefix based on which side we're on
	tunnelKey := "target:" + migrationID
	if isSource {
		tunnelKey = "source:" + migrationID
	}

	tunnel, exists := m.tunnels[tunnelKey]
	if !exists {
		return fmt.Errorf("tunnel not found for migration %s (key: %s)", migrationID, tunnelKey)
	}

	tunnel.recvStream = stream
	m.logger.Infof("Set receive stream for migration %s (key: %s)", migrationID, tunnelKey)

	// Note: We don't start runReceive() here because the gRPC handler
	// will handle the receive loop itself by calling stream.Recv() in a loop
	// and forwarding frames to handleIncomingFrame()

	return nil
}

// OpenSendStream opens the send stream on the provided connection
// This is called by target to open its send stream back to source
func (m *MigrationTunnelManager) OpenSendStream(ctx context.Context, migrationID string, grpcConn *grpc.ClientConn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Target opens send stream, so use "target:" prefix
	tunnelKey := "target:" + migrationID

	tunnel, exists := m.tunnels[tunnelKey]
	if !exists {
		return fmt.Errorf("tunnel not found for migration %s (key: %s)", migrationID, tunnelKey)
	}

	// Check if send stream already exists
	if tunnel.sendStream != nil {
		m.logger.V(2).Infof("Send stream already exists for migration %s, skipping", migrationID)
		return nil
	}

	// Create outbound gRPC stream for SENDING
	// Use a cancellable context that lives until the tunnel is stopped
	// Do NOT use a timeout - the stream must stay open for the entire migration
	streamCtx, cancel := context.WithCancel(context.Background())
	client := syncv1.NewSynchronizeClient(grpcConn)
	sendStream, err := client.MigrationTunnel(streamCtx)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create send migration tunnel stream: %v", err)
	}

	tunnel.sendStream = sendStream
	tunnel.cancelSendStream = cancel
	m.logger.Infof("Opened send stream for migration %s", migrationID)

	// Start send loop
	go tunnel.sourceToTarget()

	return nil
}

// StopTunnel stops a tunnel and cleans up all resources
// Stops both source and target tunnels for this migration
func (m *MigrationTunnelManager) StopTunnel(migrationID string) {
	m.mu.Lock()

	// Stop both source and target tunnels if they exist
	sourceTunnel, sourceExists := m.tunnels["source:"+migrationID]
	targetTunnel, targetExists := m.tunnels["target:"+migrationID]

	if sourceExists {
		delete(m.tunnels, "source:"+migrationID)
	}
	if targetExists {
		delete(m.tunnels, "target:"+migrationID)
	}

	m.mu.Unlock()

	// Clean up source tunnel
	if sourceExists {
		m.stopTunnelInternal(sourceTunnel, migrationID, "source")
	}

	// Clean up target tunnel
	if targetExists {
		m.stopTunnelInternal(targetTunnel, migrationID, "target")
	}
}

// stopTunnelInternal performs the actual cleanup of a tunnel
func (m *MigrationTunnelManager) stopTunnelInternal(tunnel *migrationTunnel, migrationID, tunnelType string) {

	// Signal shutdown
	close(tunnel.stopChan)

	// Close all channels
	tunnel.mu.Lock()
	for _, ch := range tunnel.channels {
		ch.close()
	}
	tunnel.mu.Unlock()

	// Cancel send stream context
	if tunnel.cancelSendStream != nil {
		tunnel.cancelSendStream()
	}

	// Close send stream (client streams have CloseSend)
	if tunnel.sendStream != nil {
		if clientStream, ok := tunnel.sendStream.(syncv1.Synchronize_MigrationTunnelClient); ok {
			clientStream.CloseSend()
		}
	}

	m.logger.Infof("Stopped %s tunnel for migration %s", tunnelType, migrationID)
}

// Shutdown stops all tunnels
func (m *MigrationTunnelManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for migrationID, tunnel := range m.tunnels {
		close(tunnel.stopChan)

		tunnel.mu.Lock()
		for _, ch := range tunnel.channels {
			ch.close()
		}
		tunnel.mu.Unlock()

		if tunnel.cancelSendStream != nil {
			tunnel.cancelSendStream()
		}

		if tunnel.sendStream != nil {
			if clientStream, ok := tunnel.sendStream.(syncv1.Synchronize_MigrationTunnelClient); ok {
				clientStream.CloseSend()
			}
		}

		delete(m.tunnels, migrationID)
	}

	m.logger.Info("Migration tunnel manager shutdown complete")
}

// acceptConnections accepts incoming connections on a listener
func (t *migrationTunnel) acceptConnections(listener net.Listener, protocolPort int32, targetIP string, targetPort int) {
	defer listener.Close()

	for {
		select {
		case <-t.stopChan:
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-t.stopChan:
				return
			default:
				t.logger.Reason(err).Errorf("Error accepting connection for migration %s", t.migrationID)
				continue
			}
		}

		// Handle connection in new goroutine
		go t.handleNewConnection(conn, protocolPort, targetIP, targetPort)
	}
}

// handleNewConnection handles a new virt-handler connection
func (t *migrationTunnel) handleNewConnection(conn net.Conn, protocolPort int32, targetIP string, targetPort int) {
	t.logger.Infof("New connection for migration %s channel %d", t.migrationID, protocolPort)

	var targetConn net.Conn

	// For target tunnel: connect to target virt-handler NOW (lazy connection)
	// For source tunnel: targetIP is empty, so we skip this
	if !t.isSource && targetIP != "" {
		targetAddr := net.JoinHostPort(targetIP, fmt.Sprintf("%d", targetPort))
		t.logger.Infof("Connecting to target virt-handler at %s for channel %d", targetAddr, protocolPort)

		var err error
		// Use TLS to connect to target virt-handler
		if t.tlsConfig != nil {
			targetConn, err = tls.Dial("tcp", targetAddr, t.tlsConfig)
		} else {
			targetConn, err = net.Dial("tcp", targetAddr)
		}
		if err != nil {
			t.logger.Reason(err).Errorf("Failed to connect to target virt-handler at %s", targetAddr)
			conn.Close()
			return
		}

		t.logger.Infof("Successfully connected to target virt-handler at %s for channel %d", targetAddr, protocolPort)
	}

	// Create channel
	// For target: conn = source virt-handler connection, targetConn = target virt-handler connection
	// For source: conn = source virt-handler connection, targetConn = nil
	now := time.Now()
	channel := &tunnelChannel{
		channelID:  protocolPort,
		localConn:  conn,       // Incoming connection (from source virt-handler)
		targetConn: targetConn, // Outgoing connection (to target virt-handler), nil for source tunnel
		sendQueue:  make(chan []byte, channelBufferSize),
		recvQueue:  make(chan []byte, channelBufferSize),
		stopChan:   make(chan struct{}),
		createdAt:  now,
	}
	channel.lastActivity.Store(now)

	// Register channel
	t.mu.Lock()
	t.channels[protocolPort] = channel
	t.mu.Unlock()

	// Track active connection
	proxyType := "source"
	if !t.isSource {
		proxyType = "target"
	}
	virthandler.DecentralizedMigrationProxyActiveConnectionsInc(proxyType)
	defer virthandler.DecentralizedMigrationProxyActiveConnectionsDec(proxyType)

	// Send OPEN frame
	openFrame := &syncv1.MigrationFrame{
		MigrationId: t.migrationID,
		ChannelId:   protocolPort,
		FrameType:   syncv1.FrameType_FRAME_TYPE_OPEN,
		Sequence:    atomic.AddUint64(&channel.sequence, 1),
	}

	select {
	case t.sendChan <- openFrame:
	case <-time.After(frameWriteTimeout):
		t.logger.Errorf("Timeout sending OPEN frame for channel %d", protocolPort)
		channel.close()
		return
	case <-t.stopChan:
		channel.close()
		return
	}

	// Start I/O goroutines
	t.logger.Infof("Starting sendLoop and recvLoop for migration=%s channel=%d isSource=%v", t.migrationID, protocolPort, t.isSource)
	go channel.sendLoop(t.migrationID, t.sendChan, t.stopChan, t.isSource, t.logger)
	go channel.recvLoop(t.isSource, t.logger)

	// Monitor idle timeout
	go channel.monitorIdle(t.migrationID, t.sendChan, t.logger)
}

// runSource runs the stream send/recv loops for source tunnel
// sourceToTarget sends frames from sendChan to the peer via sendStream
func (t *migrationTunnel) sourceToTarget() {
	proxyType := "target"

	for {
		select {
		case frame := <-t.sendChan: // received frame from source proxy. (source -> target)
			if t.sendStream == nil {
				// Have not connected to target virt-handler yet. Setup connections now.
				for _, targetPort := range t.targetPorts {
					if t.tlsConfig == nil {
						t.logger.Errorf("TLS config is nil for migration %s", t.migrationID)
						virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "connect_error")
						continue
					}
					targetAddr := net.JoinHostPort(t.targetIP, fmt.Sprintf("%d", targetPort))
					targetConn, err := tls.Dial("tcp", targetAddr, t.tlsConfig)
					if err != nil {
						t.logger.Reason(err).Errorf("Failed to connect to target virt-handler at %s", targetAddr)
						virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "connect_error")
						continue
					}
					t.sendStream = targetConn
				}
			}
			if err := t.sendStream.Send(frame); err != nil {
				if err != io.EOF {
					t.logger.Reason(err).Errorf("Error sending frame for migration %s", t.migrationID)
					virthandler.DecentralizedMigrationProxyErrorsInc(proxyType, "send_error")
				}
				t.errChan <- err
				return
			}
		case <-t.stopChan:
			return
		}
	}
}

// runReceive receives frames from peer via recvStream and routes them to channels
// handleIncomingFrame routes an incoming frame to the appropriate channel
func (t *migrationTunnel) handleIncomingFrame(frame *syncv1.MigrationFrame) {
	t.logger.Infof("handleIncomingFrame: migration=%s channelId=%d frameType=%v isSource=%v",
		t.migrationID, frame.ChannelId, frame.FrameType, t.isSource)

	t.mu.Lock()
	channel := t.channels[frame.ChannelId]

	// If channel doesn't exist and this is an OPEN frame, create it now
	if channel == nil && frame.FrameType == syncv1.FrameType_FRAME_TYPE_OPEN {
		t.logger.Infof("Creating new channel for OPEN frame: channelId=%d isSource=%v", frame.ChannelId, t.isSource)

		now := time.Now()
		channel = &tunnelChannel{
			channelID:  frame.ChannelId,
			localConn:  nil, // No local connection - this is a remote-initiated channel
			targetConn: nil, // Will connect to target virt-handler below
			sendQueue:  make(chan []byte, channelBufferSize),
			recvQueue:  make(chan []byte, channelBufferSize),
			stopChan:   make(chan struct{}),
			createdAt:  now,
		}
		channel.lastActivity.Store(now)

		t.channels[frame.ChannelId] = channel

		// Start recvLoop for this channel (only recvLoop, no sendLoop since no localConn)
		go channel.recvLoop(t.isSource, t.logger)

		// Track active connection
		proxyType := "source"
		if !t.isSource {
			proxyType = "target"
		}
		virthandler.DecentralizedMigrationProxyActiveConnectionsInc(proxyType)
	}

	t.mu.Unlock()

	if channel == nil {
		t.logger.Errorf("Received frame for unknown channel %d (frameType=%v, have %d channels)",
			frame.ChannelId, frame.FrameType, len(t.channels))
		return
	}

	// Handle frame based on type
	switch frame.FrameType {
	case syncv1.FrameType_FRAME_TYPE_OPEN:
		// On target side: connect to target virt-handler NOW (lazy connection)
		if !t.isSource && channel.targetConn == nil {
			targetIP := t.targetIP
			var targetPort int

			// Find the target port for this channel (frame.ChannelId is the protocol port)
			for tcpPort, protocolPort := range t.targetPorts {
				if int32(protocolPort) == frame.ChannelId {
					targetPort = tcpPort
					break
				}
			}

			if targetPort != 0 {
				targetAddr := net.JoinHostPort(targetIP, fmt.Sprintf("%d", targetPort))
				t.logger.Infof("Connecting to target virt-handler at %s for channel %d", targetAddr, frame.ChannelId)

				var targetConn net.Conn
				var err error
				// Use TLS to connect to target virt-handler
				if t.tlsConfig != nil {
					targetConn, err = tls.Dial("tcp", targetAddr, t.tlsConfig)
				} else {
					targetConn, err = net.Dial("tcp", targetAddr)
				}
				if err != nil {
					t.logger.Reason(err).Errorf("Failed to connect to target virt-handler at %s", targetAddr)
					// Send ERROR frame back
					errorFrame := &syncv1.MigrationFrame{
						MigrationId:  t.migrationID,
						ChannelId:    frame.ChannelId,
						FrameType:    syncv1.FrameType_FRAME_TYPE_ERROR,
						ErrorMessage: fmt.Sprintf("connection to target virt-handler refused: %v", err),
						Sequence:     atomic.AddUint64(&channel.sequence, 1),
					}
					select {
					case t.sendChan <- errorFrame:
					case <-time.After(frameWriteTimeout):
					case <-t.stopChan:
					}
					channel.close()
					return
				}

				// Store the target connection
				channel.targetConn = targetConn

				// Start goroutine to read from target virt-handler and send to source
				go func() {
					buf := make([]byte, frameDataSize)
					for {
						select {
						case <-channel.stopChan:
							return
						case <-t.stopChan:
							return
						default:
						}

						n, err := targetConn.Read(buf)
						if err != nil {
							if err != io.EOF {
								t.logger.Reason(err).Errorf("Error reading from target virt-handler channel %d", frame.ChannelId)
							}
							// Send CLOSE frame
							closeFrame := &syncv1.MigrationFrame{
								MigrationId: t.migrationID,
								ChannelId:   frame.ChannelId,
								FrameType:   syncv1.FrameType_FRAME_TYPE_CLOSE,
								Sequence:    atomic.AddUint64(&channel.sequence, 1),
							}
							select {
							case t.sendChan <- closeFrame:
							case <-time.After(frameWriteTimeout):
							case <-t.stopChan:
							}
							channel.closeWithStats(t.migrationID, t.isSource, t.logger)
							return
						}

						// Send DATA frame with data from target virt-handler
						frame := &syncv1.MigrationFrame{
							MigrationId: t.migrationID,
							ChannelId:   frame.ChannelId,
							FrameType:   syncv1.FrameType_FRAME_TYPE_DATA,
							Data:        append([]byte(nil), buf[:n]...),
							Sequence:    atomic.AddUint64(&channel.sequence, 1),
						}

						channel.lastActivity.Store(time.Now())

						select {
						case t.sendChan <- frame:
						case <-time.After(frameWriteTimeout):
							t.logger.Errorf("Timeout sending DATA frame from target virt-handler for channel %d", frame.ChannelId)
							channel.close()
							return
						case <-t.stopChan:
							channel.close()
							return
						}
					}
				}()
			}
		}

		t.logger.V(4).Infof("Channel %d opened", frame.ChannelId)

	case syncv1.FrameType_FRAME_TYPE_DATA:
		// Determine direction and protocol for logging
		direction := "target→source"
		if t.isSource {
			direction = "target→source"
		} else {
			direction = "source→target"
		}
		protocol := "unknown"
		switch frame.ChannelId {
		case 0:
			protocol = "virtqemud"
		case 49152:
			protocol = "libvirt"
		case 49153:
			protocol = "block-migration"
		default:
			protocol = fmt.Sprintf("port-%d", frame.ChannelId)
		}

		t.logger.V(4).Infof("Received %d bytes on migration tunnel: migration=%s protocol=%s direction=%s seq=%d",
			len(frame.Data), t.migrationID, protocol, direction, frame.Sequence)

		// Forward data to channel
		select {
		case channel.recvQueue <- frame.Data:
			channel.lastActivity.Store(time.Now())
		case <-channel.stopChan:
		case <-t.stopChan:
		}

	case syncv1.FrameType_FRAME_TYPE_CLOSE:
		// Remote closed channel
		t.logger.Infof("Channel %d closed by remote", frame.ChannelId)
		channel.close()

	case syncv1.FrameType_FRAME_TYPE_ERROR:
		// Remote reported error
		t.logger.Errorf("Channel %d error from remote: %s", frame.ChannelId, frame.ErrorMessage)
		channel.close()
	}
}

// tunnelChannel methods

func (c *tunnelChannel) sendLoop(migrationID string, sendChan chan *syncv1.MigrationFrame, stopChan chan struct{}, isSource bool, logger *log.FilteredLogger) {
	buf := make([]byte, frameDataSize)

	proxyType := "source"
	if !isSource {
		proxyType = "target"
	}

	logger.Infof("sendLoop started for migration=%s channel=%d isSource=%v", migrationID, c.channelID, isSource)

	for {
		select {
		case <-c.stopChan:
			logger.Infof("sendLoop stopped via stopChan for migration=%s channel=%d", migrationID, c.channelID)
			return
		case <-stopChan:
			logger.Infof("sendLoop stopped via tunnel stopChan for migration=%s channel=%d", migrationID, c.channelID)
			return
		default:
		}

		logger.V(4).Infof("sendLoop about to Read from localConn for migration=%s channel=%d", migrationID, c.channelID)
		n, err := c.localConn.Read(buf)
		if err != nil {
			if err != io.EOF {
				logger.Reason(err).Errorf("Error reading from local connection channel %d", c.channelID)
			}
			// Send CLOSE frame
			closeFrame := &syncv1.MigrationFrame{
				MigrationId: migrationID,
				ChannelId:   c.channelID,
				FrameType:   syncv1.FrameType_FRAME_TYPE_CLOSE,
				Sequence:    atomic.AddUint64(&c.sequence, 1),
			}
			select {
			case sendChan <- closeFrame:
			case <-time.After(frameWriteTimeout):
			case <-stopChan:
			}
			c.closeWithStats(migrationID, isSource, logger)
			return
		}

		// Determine direction and protocol for logging
		direction := "source→target"
		if !isSource {
			direction = "target→source"
		}
		protocol := "unknown"
		switch c.channelID {
		case 0:
			protocol = "virtqemud"
		case 49152:
			protocol = "libvirt"
		case 49153:
			protocol = "block-migration"
		default:
			protocol = fmt.Sprintf("port-%d", c.channelID)
		}

		logger.Infof("sendLoop READ %d bytes from localConn: migration=%s protocol=%s channel=%d direction=%s",
			n, migrationID, protocol, c.channelID, direction)

		// Track bytes read from virt-handler and sent to remote peer
		virthandler.ProxyBytesTransferredAdd(proxyType, "receive", float64(n))
		c.bytesSent.Add(uint64(n))

		// Send DATA frame
		frame := &syncv1.MigrationFrame{
			MigrationId: migrationID,
			ChannelId:   c.channelID,
			FrameType:   syncv1.FrameType_FRAME_TYPE_DATA,
			Data:        append([]byte(nil), buf[:n]...), // Copy data
			Sequence:    atomic.AddUint64(&c.sequence, 1),
		}

		logger.Infof("sendLoop SENDING %d bytes to sendChan: migration=%s protocol=%s direction=%s seq=%d",
			n, migrationID, protocol, direction, frame.Sequence)

		c.lastActivity.Store(time.Now())

		select {
		case sendChan <- frame:
			logger.Infof("sendLoop SENT frame to sendChan: migration=%s channel=%d seq=%d", migrationID, c.channelID, frame.Sequence)
		case <-time.After(frameWriteTimeout):
			logger.Errorf("Timeout sending DATA frame for channel %d", c.channelID)
			c.close()
			return
		case <-stopChan:
			c.close()
			return
		}
	}
}

func (c *tunnelChannel) recvLoop(isSource bool, logger *log.FilteredLogger) {
	proxyType := "source"
	if !isSource {
		proxyType = "target"
	}

	logger.Infof("recvLoop started for channel=%d isSource=%v", c.channelID, isSource)

	for {
		select {
		case data := <-c.recvQueue:
			logger.Infof("recvLoop received %d bytes from recvQueue for channel=%d", len(data), c.channelID)

			// For target tunnel: write data from source to target virt-handler
			// For source tunnel: write data from target to source virt-handler
			var writeConn net.Conn
			if !isSource && c.targetConn != nil {
				// Target tunnel: write to target virt-handler
				writeConn = c.targetConn
				logger.Infof("recvLoop writing to targetConn for channel=%d", c.channelID)
			} else {
				// Source tunnel or target tunnel before target connected
				writeConn = c.localConn
				logger.Infof("recvLoop writing to localConn for channel=%d", c.channelID)
			}

			n, err := writeConn.Write(data)
			if err != nil {
				logger.Reason(err).Errorf("Error writing to connection channel %d", c.channelID)
				c.close()
				return
			}
			c.lastActivity.Store(time.Now())

			logger.Infof("recvLoop WROTE %d bytes to connection channel=%d", n, c.channelID)

			// Track bytes received from remote peer and written to virt-handler
			virthandler.ProxyBytesTransferredAdd(proxyType, "send", float64(n))
			c.bytesReceived.Add(uint64(n))
		case <-c.stopChan:
			return
		}
	}
}

func (c *tunnelChannel) monitorIdle(migrationID string, sendChan chan *syncv1.MigrationFrame, logger *log.FilteredLogger) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lastActivity := c.lastActivity.Load().(time.Time)
			if time.Since(lastActivity) > channelIdleTimeout {
				logger.Infof("Channel %d idle timeout", c.channelID)
				// Send CLOSE frame
				closeFrame := &syncv1.MigrationFrame{
					MigrationId: migrationID,
					ChannelId:   c.channelID,
					FrameType:   syncv1.FrameType_FRAME_TYPE_CLOSE,
					Sequence:    atomic.AddUint64(&c.sequence, 1),
				}
				select {
				case sendChan <- closeFrame:
				case <-time.After(frameWriteTimeout):
				}
				c.close()
				return
			}
		case <-c.stopChan:
			return
		}
	}
}

func (c *tunnelChannel) close() {
	if c.stopped.CompareAndSwap(false, true) {
		close(c.stopChan)
		if c.localConn != nil {
			c.localConn.Close()
		}
	}
}

// closeWithStats closes the channel and logs transfer statistics
func (c *tunnelChannel) closeWithStats(migrationID string, isSource bool, logger *log.FilteredLogger) {
	if !c.stopped.Load() {
		duration := time.Since(c.createdAt)
		sent := c.bytesSent.Load()
		received := c.bytesReceived.Load()

		// Determine direction for logging
		direction := "source→target"
		if !isSource {
			direction = "target→source"
		}

		// Map channel ID to protocol name
		protocol := "unknown"
		switch c.channelID {
		case 0:
			protocol = "virtqemud"
		case 49152:
			protocol = "libvirt"
		case 49153:
			protocol = "block-migration"
		default:
			protocol = fmt.Sprintf("port-%d", c.channelID)
		}

		logger.Infof("Closed migration tunnel channel: migration=%s protocol=%s direction=%s duration=%v sent=%d bytes received=%d bytes",
			migrationID, protocol, direction, duration.Round(time.Second), sent, received)
	}
	c.close()
}
