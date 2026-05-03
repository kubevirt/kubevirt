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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/filewatcher"
)

const (
	defaultKeepaliveMinTime        = 5 * time.Second
	defaultDialTimeout             = 10 * time.Second
	defaultGracefulShutdownTimeout = 10 * time.Second

	defaultTunnelInit          = 1 * time.Second
	defaultTunnelCap           = 5 * time.Minute
	defaultTunnelReset         = 30 * time.Second
	defaultTunnelBackoffFactor = 2.0
	defaultTunnelBackoffJitter = 0.1
)

var h2DummyAddr = &net.TCPAddr{}

type socketWatcher interface {
	Run()
	Close()
	Events() <-chan filewatcher.Event
	Errors() <-chan error
}

type establishAndServeFunc func(ctx context.Context) error

type backupTunnelManager struct {
	targetAddr        string
	serverName        string
	nbdSocket         string
	caCert            string
	backupCert        string
	backupKey         string
	backupName        string
	backupStartTime   *metav1.Time
	sockWatcher       socketWatcher
	registerNBD       RegisterNBDFunc
	establishAndServe establishAndServeFunc

	mu     sync.Mutex
	server *grpc.Server
	cancel context.CancelFunc
}

func newBackupTunnelManager(targetAddr, serverName, nbdSocket, caCert, backupCert, backupKey, backupName string, backupStartTime *metav1.Time, registerNBD RegisterNBDFunc) *backupTunnelManager {
	m := &backupTunnelManager{
		targetAddr:      targetAddr,
		serverName:      serverName,
		nbdSocket:       nbdSocket,
		backupName:      backupName,
		backupStartTime: backupStartTime,
		caCert:          caCert,
		backupCert:      backupCert,
		backupKey:       backupKey,
		sockWatcher:     filewatcher.New(nbdSocket, time.Second),
		registerNBD:     registerNBD,
	}
	m.establishAndServe = m.doEstablishAndServe
	return m
}

func (m *backupTunnelManager) Start() error {
	if _, err := os.Stat(m.nbdSocket); err != nil {
		return fmt.Errorf("cannot initialize backup tunnel: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	m.mu.Lock()
	m.cancel = cancel
	m.mu.Unlock()

	go func() {
		defer cancel()
		m.run(ctx)
	}()
	return nil
}

func (m *backupTunnelManager) Stop() {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	m.stopServer()
}

func (m *backupTunnelManager) IsMatch(name string, startTime *metav1.Time) bool {
	return m.backupName == name && m.backupStartTime.Equal(startTime)
}

func (m *backupTunnelManager) run(ctx context.Context) {
	m.sockWatcher.Run()
	defer m.sockWatcher.Close()

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	go func() {
		for {
			select {
			case ev, ok := <-m.sockWatcher.Events():
				if !ok || ev&filewatcher.Remove != 0 {
					log.Log.Infof("NBD socket %s removed, stopping backup tunnel", m.nbdSocket)
					cancel(fmt.Errorf("NBD socket %s removed", m.nbdSocket))
					return
				}
			case err, ok := <-m.sockWatcher.Errors():
				if !ok {
					return
				}
				log.Log.Reason(err).Error("backup tunnel file watcher error")
				if _, statErr := os.Stat(m.nbdSocket); statErr != nil {
					log.Log.Reason(statErr).Errorf("NBD socket %s not accessible after watcher error, stopping", m.nbdSocket)
					cancel(fmt.Errorf("NBD socket %s not accessible after watcher error: %w", m.nbdSocket, err))
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	delayFn := wait.Backoff{
		Duration: defaultTunnelInit,
		Cap:      defaultTunnelCap,
		Factor:   defaultTunnelBackoffFactor,
		Jitter:   defaultTunnelBackoffJitter,
	}.DelayWithReset(&clock.RealClock{}, defaultTunnelReset)

	err := delayFn.Until(ctx, true, true, func(ctx context.Context) (bool, error) {
		if err := m.establishAndServe(ctx); err != nil {
			log.Log.Reason(err).Warning("backup tunnel connection lost, retrying")
		}
		return false, nil
	})

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Log.Reason(err).Error("backup tunnel stopped with terminal error")
	}

	m.stopServer()
}

func (m *backupTunnelManager) stopServer() {
	m.mu.Lock()
	s := m.server
	m.mu.Unlock()
	if s != nil {
		s.Stop()
	}
}

func (m *backupTunnelManager) doEstablishAndServe(ctx context.Context) error {
	url := fmt.Sprintf("https://%s", m.targetAddr)

	tlsConfig, err := m.prepareTLSConfig()
	if err != nil {
		return fmt.Errorf("tls config: %w", err)
	}

	conn, err := m.openConnectTunnel(ctx, url, tlsConfig)
	if err != nil {
		return fmt.Errorf("connect tunnel: %w", err)
	}
	defer conn.Close()

	srv := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             defaultKeepaliveMinTime,
			PermitWithoutStream: true,
		}),
	)
	m.registerNBD(srv, m.nbdSocket)

	m.mu.Lock()
	m.server = srv
	m.mu.Unlock()

	log.Log.Infof("backup tunnel: connected via CONNECT to %s, serving NBD", url)

	serveDone := make(chan struct{})
	go m.manageGracefulShutdown(ctx, srv, serveDone)

	closed := make(chan struct{})
	wrapped := &closeNotifyConn{Conn: conn, closed: closed}
	if err := srv.Serve(&oneConnListener{
		conn:   wrapped,
		closed: closed,
		addr:   conn.LocalAddr(),
	}); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
		return err
	}

	close(serveDone)
	m.mu.Lock()
	m.server = nil
	m.mu.Unlock()

	return nil
}

func (m *backupTunnelManager) prepareTLSConfig() (*tls.Config, error) {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(m.caCert)); !ok {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	clientCert, err := tls.X509KeyPair([]byte(m.backupCert), []byte(m.backupKey))
	if err != nil {
		return nil, fmt.Errorf("failed to load client keypair: %w", err)
	}

	return &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{clientCert},
		ServerName:   m.serverName,
		// We force HTTP/2 here to yield a bidirectional stream because
		// with HTTP/1.1 http.Client exposes an io.ReadCloser which makes writes impossible
		NextProtos: []string{"h2"},
	}, nil
}

func (m *backupTunnelManager) openConnectTunnel(ctx context.Context, targetURL string, tlsConfig *tls.Config) (net.Conn, error) {
	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
		DialTLSContext: func(ctx context.Context, network string, addr string, cfg *tls.Config) (net.Conn, error) {
			dialer := &tls.Dialer{
				NetDialer: &net.Dialer{Timeout: defaultDialTimeout},
				Config:    cfg,
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	// We use pipe to couple the gRPC server writes to the HTTP/2 request body
	pr, pw := io.Pipe()

	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, targetURL, pr)
	if err != nil {
		pr.Close()
		pw.Close()
		return nil, err
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		pr.Close()
		pw.Close()
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("CONNECT roundtrip: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		pr.Close()
		pw.Close()
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("server rejected CONNECT: %s", resp.Status)
	}

	return &h2ClientConn{r: resp.Body, w: pw, t: transport}, nil
}

func (m *backupTunnelManager) manageGracefulShutdown(ctx context.Context, srv *grpc.Server, serveDone <-chan struct{}) {
	select {
	case <-ctx.Done():
	case <-serveDone:
		return
	}

	gracefulDone := make(chan struct{})
	go func() {
		defer close(gracefulDone)
		srv.GracefulStop()
	}()

	select {
	case <-gracefulDone:
	case <-time.After(defaultGracefulShutdownTimeout):
		srv.Stop()
	}
}

type h2ClientConn struct {
	r io.ReadCloser
	w io.WriteCloser
	t *http2.Transport
}

func (c *h2ClientConn) Read(b []byte) (int, error)  { return c.r.Read(b) }
func (c *h2ClientConn) Write(b []byte) (int, error) { return c.w.Write(b) }
func (c *h2ClientConn) Close() error {
	c.t.CloseIdleConnections()
	return errors.Join(c.r.Close(), c.w.Close())
}
func (c *h2ClientConn) LocalAddr() net.Addr                { return h2DummyAddr }
func (c *h2ClientConn) RemoteAddr() net.Addr               { return h2DummyAddr }
func (c *h2ClientConn) SetDeadline(_ time.Time) error      { return nil }
func (c *h2ClientConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *h2ClientConn) SetWriteDeadline(_ time.Time) error { return nil }

type closeNotifyConn struct {
	net.Conn
	once   sync.Once
	closed chan struct{}
}

func (c *closeNotifyConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

// oneConnListener hands out exactly one connection on the first Accept call,
// then blocks subsequent Accept calls until the connection closes, at which
// point it returns net.ErrClosed so gRPC's internal accept loop exits cleanly.
type oneConnListener struct {
	mu     sync.Mutex
	conn   net.Conn
	closed chan struct{}
	addr   net.Addr
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	c := l.conn
	if c != nil {
		l.conn = nil
	}
	l.mu.Unlock()

	if c != nil {
		return c, nil
	}

	<-l.closed
	return nil, net.ErrClosed
}

func (l *oneConnListener) Close() error   { return nil }
func (l *oneConnListener) Addr() net.Addr { return l.addr }
