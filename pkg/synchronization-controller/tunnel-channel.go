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
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"kubevirt.io/client-go/log"

	synccontrollermetrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-synchronization-controller"
	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

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
