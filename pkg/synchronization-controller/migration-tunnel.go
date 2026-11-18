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
	"net"
	"sync"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"kubevirt.io/client-go/log"

	synccontrollermetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-synchronization-controller"
	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

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

func (t *migrationTunnel) GetListenerPorts() map[int]int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return copyPortMap(t.listenerPorts)
}

// HandleInboundChannel serves a MigrationTunnel stream opened by the source for one channel.
// Runs until the stream or underlying virt-handler connection closes.

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
