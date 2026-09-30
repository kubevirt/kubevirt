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
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"kubevirt.io/kubevirt/pkg/certificates"
	syncv1 "kubevirt.io/kubevirt/pkg/synchronizer-com/synchronization/v1"
)

type memFrameStream struct {
	sendCh  chan *syncv1.MigrationFrame
	recvCh  chan *syncv1.MigrationFrame
	sendErr error
}

func newMemFrameStream(buf int) *memFrameStream {
	return &memFrameStream{
		sendCh: make(chan *syncv1.MigrationFrame, buf),
		recvCh: make(chan *syncv1.MigrationFrame, buf),
	}
}

func (s *memFrameStream) Send(frame *syncv1.MigrationFrame) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	select {
	case s.sendCh <- frame:
		return nil
	case <-time.After(2 * time.Second):
		return errors.New("send timed out")
	}
}

func (s *memFrameStream) Recv() (*syncv1.MigrationFrame, error) {
	frame, ok := <-s.recvCh
	if !ok {
		return nil, io.EOF
	}
	return frame, nil
}

var _ = Describe("MigrationTunnelManager", func() {
	var manager *MigrationTunnelManager

	BeforeEach(func() {
		manager = NewMigrationTunnelManager(nil, nil)
		manager.Initialize("127.0.0.1", "127.0.0.1")
	})

	AfterEach(func() {
		manager.Shutdown()
	})

	DescribeTable("peerHost",
		func(addr, expected string) {
			Expect(peerHost(addr)).To(Equal(expected))
		},
		Entry("host:port", "10.0.0.1:9185", "10.0.0.1"),
		Entry("bare host", "10.0.0.1", "10.0.0.1"),
		Entry("IPv6 host:port", "[fd00::1]:9185", "fd00::1"),
		Entry("empty", "", ""),
	)

	Describe("BindTunnelPeer / AuthorizeTunnelPeer", func() {
		const migrationID = "mig-auth"

		It("binds a peer and authorizes matching addresses", func() {
			By("binding the peer once")
			Expect(manager.BindTunnelPeer(migrationID, "192.0.2.10:1234")).To(Succeed())

			By("authorizing the same host on a different port")
			Expect(manager.AuthorizeTunnelPeer(migrationID, "192.0.2.10:9999")).To(Succeed())
		})

		It("rejects a different peer on re-bind and authorize", func() {
			By("binding the expected peer")
			Expect(manager.BindTunnelPeer(migrationID, "192.0.2.10:1234")).To(Succeed())

			By("rejecting a different host on re-bind")
			err := manager.BindTunnelPeer(migrationID, "192.0.2.11:1234")
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.PermissionDenied))

			By("rejecting authorize from the wrong host")
			err = manager.AuthorizeTunnelPeer(migrationID, "192.0.2.11:55")
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.PermissionDenied))
		})

		DescribeTable("rejects invalid bind inputs",
			func(migrationID, peerAddr string) {
				Expect(manager.BindTunnelPeer(migrationID, peerAddr)).To(HaveOccurred())
			},
			Entry("empty migration ID", "", "192.0.2.10:1"),
			Entry("empty peer address", "mig-auth", ""),
		)

		DescribeTable("authorize error codes",
			func(setup func(), peerAddr string, expected codes.Code) {
				if setup != nil {
					setup()
				}
				err := manager.AuthorizeTunnelPeer(migrationID, peerAddr)
				Expect(err).To(HaveOccurred())
				Expect(status.Code(err)).To(Equal(expected))
			},
			Entry("unbound migration", nil, "192.0.2.10:1", codes.FailedPrecondition),
			Entry("missing peer address", nil, "", codes.Unauthenticated),
			Entry("wrong peer after bind", func() {
				Expect(manager.BindTunnelPeer(migrationID, "192.0.2.10:1234")).To(Succeed())
			}, "192.0.2.11:55", codes.PermissionDenied),
		)
	})

	It("creates and refreshes StartTargetTunnel dial coordinates", func() {
		By("creating the target tunnel")
		tunnel, err := manager.StartTargetTunnel("mig-target", "10.0.0.5", map[int]int{1: 49152})
		Expect(err).NotTo(HaveOccurred())
		Expect(tunnel.targetIP).To(Equal("10.0.0.5"))

		By("refreshing dial coordinates on the same tunnel")
		same, err := manager.StartTargetTunnel("mig-target", "10.0.0.6", map[int]int{2: 49153})
		Expect(err).NotTo(HaveOccurred())
		Expect(same).To(BeIdenticalTo(tunnel))
		Expect(same.targetIP).To(Equal("10.0.0.6"))
		Expect(same.targetPorts).To(Equal(map[int]int{2: 49153}))
	})

	Describe("HandleInboundChannel", func() {
		DescribeTable("rejects invalid OPEN preconditions",
			func(frame *syncv1.MigrationFrame, substring string) {
				err := manager.HandleInboundChannel(newMemFrameStream(1), frame)
				Expect(err).To(MatchError(ContainSubstring(substring)))
			},
			Entry("non-OPEN frame", &syncv1.MigrationFrame{
				FrameType: syncv1.FrameType_FRAME_TYPE_DATA,
			}, "expected OPEN"),
			Entry("missing target tunnel", &syncv1.MigrationFrame{
				MigrationId: "missing",
				ChannelId:   49152,
				FrameType:   syncv1.FrameType_FRAME_TYPE_OPEN,
			}, "target tunnel not found"),
		)

		It("returns ResourceExhausted when the channel semaphore is full", func() {
			By("starting a target tunnel with a saturated semaphore")
			tunnel, err := manager.StartTargetTunnel("mig-sem", "127.0.0.1", map[int]int{1: 49152})
			Expect(err).NotTo(HaveOccurred())
			tunnel.channelSem = make(chan struct{}, 1)
			Expect(tunnel.tryAcquireChannelSlot()).To(BeTrue())

			By("opening an inbound channel while the semaphore is full")
			err = manager.HandleInboundChannel(newMemFrameStream(1), &syncv1.MigrationFrame{
				MigrationId: "mig-sem",
				ChannelId:   49152,
				FrameType:   syncv1.FrameType_FRAME_TYPE_OPEN,
			})
			Expect(status.Code(err)).To(Equal(codes.ResourceExhausted))
		})

		It("dials the target virt-handler and proxies DATA frames", func() {
			By("creating TLS material for the fake virt-handler")
			tmpDir := GinkgoT().TempDir()
			store, err := certificates.GenerateSelfSignedCert(tmpDir, "testhost", "test")
			Expect(err).NotTo(HaveOccurred())
			cert, err := store.Current()
			Expect(err).NotTo(HaveOccurred())
			serverTLS := &tls.Config{
				Certificates: []tls.Certificate{*cert},
				MinVersion:   tls.VersionTLS12,
			}
			clientTLS := &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			}

			By("starting a TLS listener that stands in for virt-handler")
			ln, err := tls.Listen("tcp", "127.0.0.1:0", serverTLS)
			Expect(err).NotTo(HaveOccurred())
			defer ln.Close()
			port := ln.Addr().(*net.TCPAddr).Port

			accepted := make(chan net.Conn, 1)
			go func() {
				defer GinkgoRecover()
				conn, err := ln.Accept()
				Expect(err).NotTo(HaveOccurred())
				_ = conn.(*tls.Conn).Handshake()
				accepted <- conn
			}()

			By("starting the target tunnel pointed at the fake virt-handler")
			mgr := NewMigrationTunnelManager(clientTLS, serverTLS)
			mgr.Initialize("127.0.0.1", "127.0.0.1")
			defer mgr.Shutdown()
			_, err = mgr.StartTargetTunnel("mig-proxy", "127.0.0.1", map[int]int{port: 49152})
			Expect(err).NotTo(HaveOccurred())

			By("serving an inbound MigrationTunnel stream")
			stream := newMemFrameStream(8)
			done := make(chan error, 1)
			go func() {
				done <- mgr.HandleInboundChannel(stream, &syncv1.MigrationFrame{
					MigrationId: "mig-proxy",
					ChannelId:   49152,
					FrameType:   syncv1.FrameType_FRAME_TYPE_OPEN,
				})
			}()

			var vhConn net.Conn
			Eventually(accepted, 5*time.Second).Should(Receive(&vhConn))
			defer vhConn.Close()

			By("proxying DATA from the stream to virt-handler")
			payload := []byte("hello-tunnel")
			stream.recvCh <- &syncv1.MigrationFrame{
				MigrationId: "mig-proxy",
				ChannelId:   49152,
				FrameType:   syncv1.FrameType_FRAME_TYPE_DATA,
				Data:        payload,
			}
			buf := make([]byte, len(payload))
			Eventually(func(g Gomega) {
				_ = vhConn.SetReadDeadline(time.Now().Add(time.Second))
				n, err := vhConn.Read(buf)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(buf[:n]).To(Equal(payload))
			}, 5*time.Second, 50*time.Millisecond).Should(Succeed())

			By("proxying DATA from virt-handler back onto the stream")
			_, err = vhConn.Write([]byte("from-vh"))
			Expect(err).NotTo(HaveOccurred())
			var frame *syncv1.MigrationFrame
			Eventually(stream.sendCh, 5*time.Second).Should(Receive(&frame))
			Expect(frame.FrameType).To(Equal(syncv1.FrameType_FRAME_TYPE_DATA))
			Expect(frame.Data).To(Equal([]byte("from-vh")))

			By("closing the channel cleanly")
			stream.recvCh <- &syncv1.MigrationFrame{
				MigrationId: "mig-proxy",
				ChannelId:   49152,
				FrameType:   syncv1.FrameType_FRAME_TYPE_CLOSE,
			}
			Eventually(done, 5*time.Second).Should(Receive(BeNil()))
		})
	})

	It("removes peer binding and tears down tunnels on StopTunnel", func() {
		By("creating a target tunnel and binding a peer")
		_, err := manager.StartTargetTunnel("mig-stop", "10.0.0.1", map[int]int{1: 49152})
		Expect(err).NotTo(HaveOccurred())
		Expect(manager.BindTunnelPeer("mig-stop", "192.0.2.1:1")).To(Succeed())

		By("stopping the tunnel")
		manager.StopTunnel("mig-stop")

		By("verifying tunnel and peer binding are gone")
		manager.mu.RLock()
		_, exists := manager.tunnels["target:mig-stop"]
		manager.mu.RUnlock()
		Expect(exists).To(BeFalse())
		_, ok := manager.tunnelPeers.Load("mig-stop")
		Expect(ok).To(BeFalse())
	})
})
