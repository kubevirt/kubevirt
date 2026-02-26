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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/http2"

	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

var _ = Describe("Backup Tunnel", func() {
	Context("newBackupTunnelManager", func() {
		It("should create a manager with the provided fields", func() {
			m := newBackupTunnelManager(
				"127.0.0.1:443", "server", "/tmp/nbd.sock",
				[]byte("ca"), []byte("cert"), []byte("key"),
			)
			Expect(m).ToNot(BeNil())
			Expect(m.targetAddr).To(Equal("127.0.0.1:443"))
			Expect(m.serverName).To(Equal("server"))
			Expect(m.nbdSocket).To(Equal("/tmp/nbd.sock"))
			Expect(m.caCert).To(Equal([]byte("ca")))
			Expect(m.backupCert).To(Equal([]byte("cert")))
			Expect(m.backupKey).To(Equal([]byte("key")))
		})
	})

	Context("prepareTLSConfig", func() {
		generateTestCerts := func() (caCert, clientCert, clientKey []byte) {
			caKey, err := certutil.NewECDSAPrivateKey()
			Expect(err).ToNot(HaveOccurred())

			ca, err := certutil.NewSelfSignedCACert(certutil.Config{CommonName: "test-ca"}, caKey, time.Hour)
			Expect(err).ToNot(HaveOccurred())
			caCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw})

			leafKey, err := certutil.NewECDSAPrivateKey()
			Expect(err).ToNot(HaveOccurred())

			leaf, err := certutil.NewSignedCert(certutil.Config{
				CommonName: "test-client",
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}, leafKey, ca, caKey, time.Hour)
			Expect(err).ToNot(HaveOccurred())
			clientCert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leaf.Raw})

			leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
			Expect(err).ToNot(HaveOccurred())
			clientKey = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})

			return
		}

		It("should return an error when the CA cert PEM is invalid", func() {
			m := newBackupTunnelManager(
				"127.0.0.1:443", "server", "/tmp/nbd.sock",
				[]byte("not-a-cert"), []byte("cert"), []byte("key"),
			)
			_, err := m.prepareTLSConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse CA certificate"))
		})

		It("should return an error when the CA cert PEM slice is empty", func() {
			m := newBackupTunnelManager(
				"127.0.0.1:443", "server", "/tmp/nbd.sock",
				[]byte{}, []byte("cert"), []byte("key"),
			)
			_, err := m.prepareTLSConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse CA certificate"))
		})

		It("should return an error when the client keypair is invalid", func() {
			caCert, _, _ := generateTestCerts()
			m := newBackupTunnelManager(
				"127.0.0.1:443", "server", "/tmp/nbd.sock",
				caCert, []byte("not-a-cert"), []byte("not-a-key"),
			)
			_, err := m.prepareTLSConfig()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load client keypair"))
		})

		It("should build a valid TLS config when all inputs are correct", func() {
			caCert, clientCert, clientKey := generateTestCerts()
			m := newBackupTunnelManager(
				"127.0.0.1:443", "myserver", "/tmp/nbd.sock",
				caCert, clientCert, clientKey,
			)
			cfg, err := m.prepareTLSConfig()
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg).ToNot(BeNil())
			Expect(cfg.ServerName).To(Equal("myserver"))
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

	Context("watchSocket", func() {
		var (
			manager  *backupTunnelManager
			sockPath string
		)

		BeforeEach(func() {
			tmpDir := GinkgoT().TempDir()
			sockPath = filepath.Join(tmpDir, "nbd.sock")
			manager = &backupTunnelManager{nbdSocket: sockPath}
		})

		It("should return a channel that is closed when the socket is removed", func() {
			f, err := os.Create(sockPath)
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ch, err := manager.watchSocket(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.Remove(sockPath)).To(Succeed())
			Eventually(ch, 5*time.Second).Should(BeClosed())
		})

		It("should return a channel that is closed when context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())

			ch, err := manager.watchSocket(ctx)
			Expect(err).ToNot(HaveOccurred())
			cancel()
			Eventually(ch, 3*time.Second).Should(BeClosed())
		})

		It("should return an error if the socket directory does not exist", func() {
			manager.nbdSocket = "/doesnotexist/nbd.sock"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, err := manager.watchSocket(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to watch directory"))
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
})
