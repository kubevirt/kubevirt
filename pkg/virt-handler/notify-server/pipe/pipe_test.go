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
package pipe

import (
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/notify-server/auth"
)

var _ = Describe("InjectNotify", func() {

	var pod *isolation.MockIsolationResult
	var socketPath string

	BeforeEach(func() {
		tmp := GinkgoT().TempDir()
		safeTmp, err := safepath.JoinAndResolveWithRelativeRoot(tmp)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(tmp, "dir"), 0777)).To(Succeed())

		pod = isolation.NewMockIsolationResult(gomock.NewController(GinkgoT()))
		pod.EXPECT().MountRoot().Return(safeTmp, nil)

		socketPath = filepath.Join(tmp, "dir", "domain-notify-pipe.sock")
	})
	It("should return working listener", func() {
		listener, err := InjectNotify(pod, "dir", false)
		Expect(err).ToNot(HaveOccurred())
		defer listener.Close()

		resultChan := make(chan error)
		go func() {
			conn, err := listener.Accept()
			Expect(conn.Close()).To(Succeed())
			resultChan <- err
		}()

		conn, err := net.Dial("unix", socketPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(conn.Close()).To(Succeed())
		Eventually(resultChan).Should(Receive(BeNil()))
	})

	It("should make it accessible for nonroot", func() {
		original := diskutils.DefaultOwnershipManager
		DeferCleanup(func() {
			diskutils.DefaultOwnershipManager = original
		})
		ownershipManager := diskutils.NewMockOwnershipManagerInterface(gomock.NewController(GinkgoT()))
		diskutils.DefaultOwnershipManager = ownershipManager
		ownershipManager.EXPECT().SetFileOwnership(gomock.Any()).Times(1)

		listener, err := InjectNotify(pod, "dir", true)
		Expect(err).ToNot(HaveOccurred())
		defer listener.Close()
	})
})

var _ = Describe("NewConnectToNotifyFunc", func() {
	var (
		virtShareDir string
		notifyLn     net.Listener
	)

	BeforeEach(func() {
		virtShareDir = GinkgoT().TempDir()
		var err error
		notifyLn, err = net.Listen("unix", filepath.Join(virtShareDir, "domain-notify.sock"))
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(notifyLn.Close)
	})

	It("should send the client identity ahead of the client data", func() {
		identity := auth.Identity{Namespace: "default", Name: "testvmi", UID: "1234"}

		conn, err := NewConnectToNotifyFunc(virtShareDir, identity)()
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()
		_, err = conn.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())

		notifyConn, err := notifyLn.Accept()
		Expect(err).ToNot(HaveOccurred())
		defer notifyConn.Close()
		Expect(auth.ReadIdentity(notifyConn)).To(Equal(identity))
		Expect(readMsg(notifyConn, 5)).To(Equal("Hello"))
	})

	It("should fail and close the connection when the identity is incomplete", func() {
		_, err := NewConnectToNotifyFunc(virtShareDir, auth.Identity{Namespace: "default", Name: "testvmi"})()
		Expect(err).To(MatchError(ContainSubstring("incomplete identity")))

		notifyConn, err := notifyLn.Accept()
		Expect(err).ToNot(HaveOccurred())
		defer notifyConn.Close()
		_, err = readMsg(notifyConn, 1)
		Expect(err).To(MatchError(io.EOF))
	})

	It("should fail when the notify server is not listening", func() {
		_, err := NewConnectToNotifyFunc(GinkgoT().TempDir(), auth.Identity{Namespace: "default", Name: "testvmi", UID: "1234"})()
		Expect(err).To(MatchError(ContainSubstring("error connecting to domain-notify.sock")))
	})
})

var _ = Describe("Serve", func() {
	It("should proxy accepted connections until ctx is canceled", func() {
		pipePath := filepath.Join(GinkgoT().TempDir(), "domain-notify-pipe.sock")
		listener, err := net.Listen("unix", pipePath)
		Expect(err).ToNot(HaveOccurred())

		notifyS, notify := net.Pipe()
		defer notify.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		Serve(ctx, log.Log, listener, func() (net.Conn, error) { return notifyS, nil })

		client, err := net.Dial("unix", pipePath)
		Expect(err).ToNot(HaveOccurred())
		defer client.Close()
		_, err = client.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())
		Expect(readMsg(notify, 5)).To(Equal("Hello"))

		cancel()
		Eventually(func() error {
			conn, err := net.Dial("unix", pipePath)
			if err == nil {
				conn.Close()
			}
			return err
		}).Should(HaveOccurred())
	})
})

var _ = Describe("Proxy", func() {
	It("should proxy", func() {
		pipeS, pipe := net.Pipe()
		notifyS, notify := net.Pipe()

		exit := make(chan struct{})
		go func() {
			Proxy(log.Log, pipeS, func() (net.Conn, error) { return notifyS, nil })
			exit <- struct{}{}
		}()

		_, err := pipe.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())

		Expect(func() (string, error) {
			return readMsg(notify, 5)
		}()).Should(Equal("Hello"))

		_, err = notify.Write([]byte("Hello back"))
		Expect(err).ToNot(HaveOccurred())

		Expect(func() (string, error) {
			return readMsg(pipe, 10)
		}()).Should(Equal("Hello back"))

		// Closing the pipe should close the notify connection and proxy should exit
		Expect(pipe.Close()).To(Succeed())
		Eventually(exit).Should(Receive())
		Eventually(func() error {
			_, err := readMsg(notify, 5)
			return err
		}).Should(MatchError(io.EOF))
	})
})

var _ = Describe("ChanFromListener", func() {
	It("should work", func() {
		tmp := GinkgoT().TempDir()
		safeTmp, err := safepath.JoinAndResolveWithRelativeRoot(tmp)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(tmp, "dir"), 0777)).To(Succeed())

		pod := isolation.NewMockIsolationResult(gomock.NewController(GinkgoT()))
		pod.EXPECT().MountRoot().Return(safeTmp, nil)

		socketPath := filepath.Join(tmp, "dir", "domain-notify-pipe.sock")

		listener, err := InjectNotify(pod, "dir", false)
		Expect(err).ToNot(HaveOccurred())
		defer listener.Close()
		ctx, cancelListener := context.WithCancel(context.Background())
		pipeChan := ChanFromListener(ctx, log.Log, listener)

		conn, err := net.Dial("unix", socketPath)
		Expect(err).ToNot(HaveOccurred())
		_, err = conn.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())

		var pipeConn net.Conn
		var open bool
		select {
		case <-time.After(time.Second):
			Fail("Expected to recieve new connection from channel")
		case pipeConn, open = <-pipeChan:
			if !open {
				Fail("Expected to not close channel")
			}
			msg, err := readMsg(pipeConn, 5)
			Expect(err).ToNot(HaveOccurred())
			Expect(msg).To(Equal("Hello"))
		}

		cancelListener()

		// Should close listener
		Eventually(func() error {
			_, err = net.Dial("unix", socketPath)
			return err
		}).Should(HaveOccurred())

		// will keep current connections

		_, err = conn.Write([]byte("Hello"))
		Expect(err).ToNot(HaveOccurred())

		msg, err := readMsg(pipeConn, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(msg).To(Equal("Hello"))

	})

})

var _ = Describe("Pipe", func() {
	It("should work", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		connChan := make(chan net.Conn, 1)
		go func() {
			Pipe(ctx, connChan, func(c net.Conn) {
				_, err := c.Write([]byte("Hello"))
				Expect(err).ToNot(HaveOccurred())
			})
		}()

		pipeConn, pipeMirror := net.Pipe()
		connChan <- pipeConn

		msg, err := readMsg(pipeMirror, 5)
		Expect(err).ToNot(HaveOccurred())
		Expect(msg).To(Equal("Hello"))
	})

	It("should return when ctx is canceled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		exit := make(chan struct{})
		go func() {
			Pipe(ctx, make(chan net.Conn, 1), func(c net.Conn) {})
			exit <- struct{}{}
		}()
		Consistently(exit).ShouldNot(Receive())
		cancel()
		Eventually(exit).Should(Receive())
	})

	It("should return when chanel is closed", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		exit := make(chan struct{})
		pipeChan := make(chan net.Conn, 1)
		go func() {
			Pipe(ctx, pipeChan, func(c net.Conn) {})
			exit <- struct{}{}
		}()
		Consistently(exit).ShouldNot(Receive())
		close(pipeChan)
		Eventually(exit).Should(Receive())
	})
})

func readMsg(conn net.Conn, n int) (string, error) {
	buf := make([]byte, n)
	_, err := conn.Read(buf)
	if err != nil {
		return "", err
	}
	return string(buf), err

}
