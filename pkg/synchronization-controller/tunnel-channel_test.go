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
 */

package synchronization

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"kubevirt.io/client-go/log"
)

var _ = Describe("tunnelChannel", func() {
	DescribeTable("isExpectedProxyCloseErr",
		func(err error, expected bool) {
			Expect(isExpectedProxyCloseErr(err)).To(Equal(expected))
		},
		Entry("nil", nil, true),
		Entry("EOF", io.EOF, true),
		Entry("net.ErrClosed", net.ErrClosed, true),
		Entry("ErrClosedPipe", io.ErrClosedPipe, true),
		Entry("context.Canceled", context.Canceled, true),
		Entry("gRPC Canceled", status.Error(codes.Canceled, "canceled"), true),
		Entry("gRPC Unavailable", status.Error(codes.Unavailable, "gone"), true),
		Entry("closed connection string", errors.New("use of closed network connection"), true),
		Entry("joined expected errors", errors.Join(io.EOF, net.ErrClosed), true),
		Entry("unexpected error", errors.New("boom"), false),
		Entry("joined mixed errors", errors.Join(io.EOF, errors.New("boom")), false),
		Entry("gRPC Internal", status.Error(codes.Internal, "nope"), false),
	)

	It("closes an idle channel after the idle timeout", func() {
		By("starting a claimed channel with stale activity and a short idle timeout")
		tunnel := &migrationTunnel{
			migrationID: "mig-idle",
			stopChan:    make(chan struct{}),
			channelSem:  make(chan struct{}, 1),
			logger:      log.DefaultLogger(),
		}
		client, server := net.Pipe()
		defer client.Close()
		defer server.Close()

		stream := newMemFrameStream(4)
		ch := &tunnelChannel{
			channelID:         1,
			stream:            stream,
			conn:              server,
			stopChan:          make(chan struct{}),
			createdAt:         time.Now(),
			logger:            tunnel.logger,
			idleTimeout:       50 * time.Millisecond,
			idleCheckInterval: 20 * time.Millisecond,
		}
		ch.lastActivity.Store(time.Now().Add(-time.Hour))
		ch.sequence = 1
		tunnel.addChannel(ch)

		done := make(chan error, 1)
		go func() {
			done <- tunnel.runClaimedChannel("source", ch)
		}()

		By("waiting for idle timeout to stop the channel")
		Eventually(func() bool { return ch.stopped.Load() }, 2*time.Second, 20*time.Millisecond).Should(BeTrue())
		Eventually(done, 2*time.Second).Should(Receive(BeNil()))
	})
})
