/*
 * This file is part of the kubevirt project
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

package exec

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/apimachinery/pkg/util/httpstream/spdy"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// The vendored SPDY round tripper reads the upgrade response directly from its
// connection without observing request cancellation. Use the standard HTTP
// transport for the handshake so a missing upgrade response cannot outlive the
// exec context. It also retains the kubeconfig's TLS, authentication and proxy
// settings. Once upgraded, StreamWithContext owns and closes the SPDY connection.
func newCancellableSPDYExecutor(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
	config = rest.CopyConfig(config)
	config.NextProtos = []string{"http/1.1"} // Connection upgrades require HTTP/1.1.
	transport, err := rest.TransportFor(config)
	if err != nil {
		return nil, err
	}
	upgrader := &cancellableSPDYUpgrader{transport: transport}
	return remotecommand.NewSPDYExecutorForTransports(upgrader, upgrader, method, url)
}

type cancellableSPDYUpgrader struct {
	transport  http.RoundTripper
	connection net.Conn
	body       io.ReadWriteCloser
}

func (s *cancellableSPDYUpgrader) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) { s.connection = info.Conn },
	}))
	req.Header.Add(httpstream.HeaderConnection, httpstream.HeaderUpgrade)
	req.Header.Add(httpstream.HeaderUpgrade, spdy.HeaderSpdy31)
	resp, err := s.transport.RoundTrip(req)
	if err == nil && resp.StatusCode == http.StatusSwitchingProtocols {
		if body, ok := resp.Body.(io.ReadWriteCloser); ok {
			s.body = &cancellableUpgradeBody{
				ReadWriteCloser: body,
				stop:            context.AfterFunc(req.Context(), func() { body.Close() }),
			}
			// client-go closes the HTTP body after NewConnection. Ownership of the
			// upgraded stream instead passes to the returned SPDY connection.
			resp.Body = io.NopCloser(s.body)
		}
	}
	return resp, err
}

func (s *cancellableSPDYUpgrader) NewConnection(resp *http.Response) (httpstream.Connection, error) {
	if resp.StatusCode != http.StatusSwitchingProtocols ||
		!strings.EqualFold(resp.Header.Get(httpstream.HeaderUpgrade), spdy.HeaderSpdy31) ||
		!strings.Contains(strings.ToLower(resp.Header.Get(httpstream.HeaderConnection)), "upgrade") {
		defer resp.Body.Close()
		if s.body != nil {
			defer s.body.Close()
		}
		if resp.StatusCode == http.StatusSwitchingProtocols {
			return nil, fmt.Errorf("unexpected SPDY upgrade protocol %q", resp.Header.Get(httpstream.HeaderUpgrade))
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		if err != nil {
			return nil, fmt.Errorf("unable to read SPDY upgrade response: %w", err)
		}
		return nil, fmt.Errorf("unable to upgrade connection (%s): %s", resp.Status, strings.TrimSpace(string(body)))
	}
	connection := s.body
	if connection == nil || s.connection == nil {
		if connection != nil {
			connection.Close()
		}
		resp.Body.Close()
		return nil, fmt.Errorf("SPDY upgrade response does not provide a bidirectional stream")
	}
	conn, err := spdy.NewClientConnectionWithPings(&upgradedConnection{Conn: s.connection, body: connection}, 5*time.Second)
	if err != nil {
		connection.Close()
	}
	return conn, err
}

// Read through the response body to retain any bytes buffered after the upgrade.
// The underlying connection supplies addresses and deadlines required by SPDY.
type upgradedConnection struct {
	net.Conn
	body io.ReadWriteCloser
}

func (c *upgradedConnection) Read(p []byte) (int, error)  { return c.body.Read(p) }
func (c *upgradedConnection) Write(p []byte) (int, error) { return c.body.Write(p) }
func (c *upgradedConnection) Close() error                { return c.body.Close() }

type cancellableUpgradeBody struct {
	io.ReadWriteCloser
	stop func() bool
}

func (b *cancellableUpgradeBody) Close() error {
	b.stop()
	return b.ReadWriteCloser.Close()
}
