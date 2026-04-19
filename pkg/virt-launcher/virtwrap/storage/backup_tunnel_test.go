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

package storage

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/http2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/filewatcher"
)

var _ = Describe("Backup Tunnel", func() {
	Context("newBackupTunnelManager", func() {
		It("should create a manager with the provided fields", func() {
			startTime := metav1.Now()
			m := newBackupTunnelManager(
				"127.0.0.1:443",
				"server",
				"/tmp/nbd.sock",
				"ca",
				"cert",
				"key",
				"test-backup",
				&startTime,
				nil,
			)
			Expect(m).ToNot(BeNil())
			Expect(m.targetAddr).To(Equal("127.0.0.1:443"))
			Expect(m.serverName).To(Equal("server"))
			Expect(m.nbdSocket).To(Equal("/tmp/nbd.sock"))
			Expect(m.backupName).To(Equal("test-backup"))
			Expect(m.backupStartTime).To(Equal(&startTime))
			Expect(m.caCert).To(Equal("ca"))
			Expect(m.backupCert).To(Equal("cert"))
			Expect(m.backupKey).To(Equal("key"))
		})
	})

	Context("IsMatch", func() {
		var (
			now     metav1.Time
			after   metav1.Time
			manager *backupTunnelManager
		)

		BeforeEach(func() {
			now = metav1.Now()
			after = metav1.NewTime(now.Add(time.Minute))
			manager = &backupTunnelManager{
				backupName:      "test-backup",
				backupStartTime: &now,
			}
		})

		DescribeTable("should correctly identify matches",
			func(checkName string, checkTime *metav1.Time, expected bool) {
				Expect(manager.IsMatch(checkName, checkTime)).To(Equal(expected))
			},
			Entry("when name and time match exactly", "test-backup", &now, true),
			Entry("when name matches but times differ", "test-backup", &after, false),
			Entry("when time matches but names differ", "other-backup", &now, false),
			Entry("when name matches but check time is nil", "test-backup", nil, false),
		)
	})

	Context("prepareTLSConfig", func() {
		generateTestCerts := func() (string, string, string) {
			caKey, err := certutil.NewECDSAPrivateKey()
			Expect(err).ToNot(HaveOccurred())

			ca, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "test-ca"}, caKey, time.Hour)
			Expect(err).ToNot(HaveOccurred())
			caCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})

			leafKey, err := certutil.NewECDSAPrivateKey()
			Expect(err).ToNot(HaveOccurred())

			leaf, err := certutil.NewSignedCert(certutil.Config{
				CommonName: "test-client",
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}, leafKey, ca, caKey, time.Hour)
			Expect(err).ToNot(HaveOccurred())
			clientCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf.Raw})

			leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
			Expect(err).ToNot(HaveOccurred())
			clientKey := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})

			return string(caCert), string(clientCert), string(clientKey)
		}

		It("should return an error when the CA cert PEM is invalid", func() {
			m := newBackupTunnelManager(
				"127.0.0.1:443",
				"server",
				"/tmp/nbd.sock",
				"ca",
				"cert",
				"key",
				"test-backup",
				nil,
				nil,
			)
			_, err := m.prepareTLSConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse CA certificate"))
		})

		It("should return an error when the CA cert PEM slice is empty", func() {
			m := newBackupTunnelManager(
				"127.0.0.1:443",
				"server",
				"/tmp/nbd.sock",
				"",
				"cert",
				"key",
				"test-backup",
				nil,
				nil,
			)
			_, err := m.prepareTLSConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse CA certificate"))
		})

		It("should return an error when the client keypair is invalid", func() {
			caCert, _, _ := generateTestCerts()
			m := newBackupTunnelManager(
				"127.0.0.1:443",
				"server",
				"/tmp/nbd.sock",
				caCert,
				"not-a-cert",
				"not-a-key",
				"test-backup",
				nil,
				nil,
			)
			_, err := m.prepareTLSConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load client keypair"))
		})

		It("should build a valid TLS config when all inputs are correct", func() {
			caCert, clientCert, clientKey := generateTestCerts()
			m := newBackupTunnelManager(
				"127.0.0.1:443",
				"server",
				"/tmp/nbd.sock",
				caCert,
				clientCert,
				clientKey,
				"test-backup",
				nil,
				nil,
			)
			cfg, err := m.prepareTLSConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.ServerName).To(Equal("server"))
			Expect(cfg.Certificates).To(HaveLen(1))
			Expect(cfg.RootCAs).ToNot(BeNil())
			Expect(cfg.NextProtos).To(ConsistOf("h2"))
		})
	})

	Context("closeNotifyConn", func() {
		It("should close the underlying connection and signals the channel exactly once", func() {
			server, client := net.Pipe()
			defer server.Close()

			closed := make(chan struct{})
			notify := &closeNotifyConn{Conn: client, closed: closed}

			Expect(notify.Close()).To(Succeed())

			Eventually(closed).Should(BeClosed())
			Expect(notify.Close()).To(Or(Succeed(), HaveOccurred()))
			Consistently(closed).Should(BeClosed())
		})
	})

	Context("oneConnListener", func() {
		It("should return the connection on the first Accept call", func() {
			_, client := net.Pipe()
			defer client.Close()

			closed := make(chan struct{})
			l := &oneConnListener{
				conn:   client,
				closed: closed,
				addr:   client.LocalAddr(),
			}

			conn, err := l.Accept()
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).To(Equal(client))
		})

		It("should block on the second Accept until the closed channel is shut", func() {
			_, client := net.Pipe()
			defer client.Close()

			closed := make(chan struct{})
			l := &oneConnListener{
				conn:   client,
				closed: closed,
				addr:   client.LocalAddr(),
			}

			conn, err := l.Accept()
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())

			result := make(chan error, 1)
			go func() {
				_, acceptErr := l.Accept()
				result <- acceptErr
			}()

			Consistently(result, 100*time.Millisecond).ShouldNot(Receive())

			close(closed)
			Eventually(result).Should(Receive(Equal(net.ErrClosed)))
		})

		It("should return the stored address", func() {
			_, client := net.Pipe()
			defer client.Close()

			addr := client.LocalAddr()
			l := &oneConnListener{addr: addr}
			Expect(l.Addr()).To(Equal(addr))
		})
	})

	Context("openConnectTunnel", func() {
		var (
			srv       *httptest.Server
			tlsConfig *tls.Config
			manager   *backupTunnelManager
		)

		startServer := func(handler http.Handler) {
			srv = httptest.NewUnstartedServer(handler)
			srv.TLS = &tls.Config{NextProtos: []string{"h2"}}
			Expect(http2.ConfigureServer(srv.Config, &http2.Server{})).To(Succeed())
			srv.StartTLS()
			tlsConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
			tlsConfig.NextProtos = []string{"h2"}
			manager = &backupTunnelManager{}
		}

		AfterEach(func() {
			if srv != nil {
				srv.Close()
			}
		})

		It("should return an error when the server rejects CONNECT with a non-200", func() {
			startServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "forbidden", http.StatusForbidden)
			}))

			conn, err := manager.openConnectTunnel(context.Background(), srv.URL, tlsConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("403"))
			Expect(conn).To(BeNil())
		})

		It("should return a working bidirectional conn when the server responds with 200", func() {
			startServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				buf := make([]byte, 4)
				io.ReadFull(r.Body, buf)
				w.Write(buf)
				w.(http.Flusher).Flush()
			}))

			conn, err := manager.openConnectTunnel(context.Background(), srv.URL, tlsConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			defer conn.Close()

			_, err = conn.Write([]byte("test"))
			Expect(err).ToNot(HaveOccurred())

			buf := make([]byte, 4)
			_, err = io.ReadFull(conn, buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf)).To(Equal("test"))
		})

		It("should return an error when the context is already cancelled", func() {
			startServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			conn, err := manager.openConnectTunnel(ctx, srv.URL, tlsConfig)
			Expect(err).To(HaveOccurred())
			Expect(conn).To(BeNil())
		})
	})

	Context("run", func() {
		var (
			watcher *fakeSocketWatcher
			manager *backupTunnelManager
		)

		BeforeEach(func() {
			watcher = newFakeSocketWatcher()
			manager = &backupTunnelManager{
				nbdSocket:   "/test/nbd.sock",
				sockWatcher: watcher,
				establishAndServe: func(ctx context.Context) error {
					<-ctx.Done()
					return ctx.Err()
				},
			}
		})

		It("should exit when a socket Remove event is received", func() {
			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(context.Background())
			}()

			watcher.events <- filewatcher.Remove
			Eventually(done).Should(BeClosed())
		})

		It("should exit when the parent context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())

			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(ctx)
			}()

			cancel()
			Eventually(done).Should(BeClosed())
		})

		It("should exit when the events channel is closed", func() {
			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(context.Background())
			}()

			close(watcher.events)
			Eventually(done).Should(BeClosed())
		})

		It("should exit when watcher error is received and socket is not accessible", func() {
			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(context.Background())
			}()

			watcher.errors <- fmt.Errorf("test error")
			Eventually(done).Should(BeClosed())
		})

		It("should call establishAndServe", func() {
			called := make(chan struct{})
			manager.establishAndServe = func(ctx context.Context) error {
				close(called)
				<-ctx.Done()
				return ctx.Err()
			}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(ctx)
			}()

			Eventually(called).Should(BeClosed())
			cancel()
			Eventually(done).Should(BeClosed())
		})

		It("should close the socket watcher on exit", func() {
			ctx, cancel := context.WithCancel(context.Background())

			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(ctx)
			}()

			cancel()
			Eventually(done).Should(BeClosed())
			Expect(watcher.closed).To(BeTrue())
		})

		It("should retry establishAndServe after failure", func() {
			calls := make(chan struct{}, 10)
			manager.establishAndServe = func(ctx context.Context) error {
				calls <- struct{}{}
				return fmt.Errorf("connection failed")
			}

			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				manager.run(ctx)
			}()

			Eventually(calls).Should(Receive())
			Eventually(calls, 5*time.Second).Should(Receive())

			cancel()
			Eventually(done).Should(BeClosed())
		})
	})

})

type fakeSocketWatcher struct {
	events chan filewatcher.Event
	errors chan error
	closed bool
}

func newFakeSocketWatcher() *fakeSocketWatcher {
	return &fakeSocketWatcher{
		events: make(chan filewatcher.Event, 10),
		errors: make(chan error, 10),
	}
}

func (f *fakeSocketWatcher) Run()                             {}
func (f *fakeSocketWatcher) Close()                           { f.closed = true }
func (f *fakeSocketWatcher) Events() <-chan filewatcher.Event { return f.events }
func (f *fakeSocketWatcher) Errors() <-chan error             { return f.errors }
