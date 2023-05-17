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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virtio_serial

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"net/url"
	"strings"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	metricsScraper "kubevirt.io/kubevirt/pkg/monitoring/domainstats/downwardmetrics"
)

const (
	maxConnectAttempts   = 6
	maxRequestsPerSecond = 5
	maxRequestsBurst     = 1 // must be >= 1, otherwise `rateLimiter.Wait()` will fail
	invalidRequest       = "INVALID REQUEST\n\n"
	emptyMetrics         = "<metrics><!-- host metrics not available --><!-- VM metrics not available --></metrics>"
)

// This is a compile-time assertion to ensure that `maxRequestsBurst` is >= 1, otherwise `rateLimiter.Wait()` will fail
// (will also fail for `maxRequestsBurst` > 256)
const _ = uint8(maxRequestsBurst - 1)

func RunDownwardMetricsVirtioServer(ctx context.Context, nodeName, channelSocketPath, launcherSocketPath string) error {
	report, err := newMetricsReporter(nodeName, launcherSocketPath)
	if err != nil {
		return err
	}

	server := downwardMetricsServer{
		rateLimiter:        rate.NewLimiter(maxRequestsPerSecond, maxRequestsBurst),
		maxConnectAttempts: maxConnectAttempts,
		virtioSerialSocket: channelSocketPath,
		reportFn:           report,
	}
	go server.start(ctx)
	return nil
}

type metricsReporter func() (*api.Metrics, error)

func newMetricsReporter(nodeName, launcherSocketPath string) (metricsReporter, error) {
	exists, err := diskutils.FileExists(launcherSocketPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("virt-launcher socket not found")
	}

	scraper := metricsScraper.NewReporter(nodeName)

	return func() (*api.Metrics, error) {
		return scraper.Report(launcherSocketPath)
	}, nil
}

// The DownwardMetrics server is special, in the sense that the socket is created
// by QEMU as a server (listen) and the DownwardMetrics server connects to it.
// Once the connection is established, the application inside the VM has to open
// the character device and send the requests, and each request is handled sequentially.
// Only one application can open the character device at the same time, and reading/writing
// the device blocks the application
type downwardMetricsServer struct {
	rateLimiter        *rate.Limiter
	maxConnectAttempts uint
	virtioSerialSocket string
	reportFn           metricsReporter
}

func (s *downwardMetricsServer) start(ctx context.Context) {
	conn, err := connect(ctx, s.virtioSerialSocket, s.maxConnectAttempts)
	if err != nil {
		log.Log.Reason(err).Error("failed to connect to virtio-serial socket")
		return
	}

	s.serve(ctx, conn)
}

func (s *downwardMetricsServer) serve(ctx context.Context, conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Log.Reason(err).Warning("virtio-serial failed to close the connection")
		}
	}(conn)

	type reqResult struct {
		request string
		err     error
	}

	// We must provide a space in the buffer to prevent the goroutine from blocking
	// when sending a new request, through the newRequest channel, after the context
	// is canceled
	newRequest := make(chan reqResult, 1)
	reader := bufio.NewReader(conn)

	for {
		// The virtio-serial vhostmd server implementation serves one request at a time,
		// so we make sure the client inside the guest cannot send another request before
		// the previous request was processed
		go func() {
			if err := s.rateLimiter.Wait(ctx); err != nil {
				return // Context canceled, just return
			}

			// This will block until a request is received or `conn` is closed by the
			// parent goroutine or QEMU
			requestLine, err := waitForRequest(reader)
			if err != nil && (errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed)) {
				// `net.ErrClosed`: The parent goroutine closed the connection
				// `io.EOF`: QEMU closes the connection
				close(newRequest) // this is really only required if QEMU closes the connection
				return
			}

			// Non-blocking send, because `newRequest` is a buffered channel
			newRequest <- reqResult{request: requestLine, err: err}
		}()

		select {
		case res, ok := <-newRequest:
			if !ok {
				return // QEMU closed the connection
			}

			if res.err != nil {
				log.Log.Reason(res.err).Warning("virtio-serial socket read failed")
				// Let's clean the remaining data in the connection to prevent the following
				// requests to fail due to left over bytes from the current invalid request
				reader.Reset(conn)
				replyError(conn)
				continue
			}

			response, err := s.handleRequest(res.request)
			if err != nil {
				log.Log.Reason(err).Error("failed to process the request")
				replyError(conn)
				continue
			}

			err = reply(conn, response)
			if err != nil {
				log.Log.Reason(err).Error("failed to send the metrics")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *downwardMetricsServer) handleRequest(requestLine string) ([]byte, error) {
	err := parseRequest(requestLine)
	if err != nil {
		return nil, err
	}

	response, err := s.getXmlMetrics()
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *downwardMetricsServer) getXmlMetrics() ([]byte, error) {
	var xmlMetrics []byte
	metrics, err := s.reportFn()
	if err != nil {
		xmlMetrics = []byte(emptyMetrics)
		log.Log.Reason(err).Error("failed to collect the metrics")
	} else {
		xmlMetrics, err = xml.MarshalIndent(metrics, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode metrics: %v", err)
		}
	}

	// `vm-dump-metrics` expects `\n\n` as a termination symbol
	xmlMetrics = append(xmlMetrics, '\n', '\n')
	return xmlMetrics, nil
}

func connect(ctx context.Context, socketPath string, attempts uint) (net.Conn, error) {
	var conn net.Conn
	var err error

	multiplier := 1
	for i := uint(0); i < attempts; i++ {
		conn, err = net.Dial("unix", socketPath)
		if err == nil {
			break
		}

		// It is only tried again in case the socket doesn't exist or not one is
		// listening on the other end yet
		if !(errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT)) {
			break
		}

		if i == attempts {
			return nil, fmt.Errorf("reached maximum number of connection attempts")
		}

		backoff := time.Duration(multiplier) * time.Second
		multiplier *= 2

		select {
		case <-time.After(backoff): // try again
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return conn, err
}

func waitForRequest(bufReader *bufio.Reader) (string, error) {
	reader := textproto.NewReader(bufReader)

	// First wait for an HTTP-like line, like GET /metrics/XML
	request, err := reader.ReadLine()
	if err != nil {
		return "", err
	}

	// Then wait for a blank line
	blankLine, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	if blankLine != "" {
		return "", errors.New("malformed request missing blank line")
	}

	return request, nil
}

func parseRequest(requestLine string) error {
	method, rawUri, ok := strings.Cut(requestLine, " ")
	if !ok {
		return fmt.Errorf("malformed request: %q", requestLine)
	}

	if method != "GET" {
		return fmt.Errorf("invalid method: %q", method)
	}

	requestUri, err := url.ParseRequestURI(rawUri)
	if err != nil {
		return err
	}

	// Currently this is the only valid request
	if requestUri.Path != "/metrics/XML" {
		return fmt.Errorf("invalid request: %q", requestUri.Path)
	}

	return nil
}

func replyError(conn net.Conn) {
	err := reply(conn, []byte(invalidRequest))
	if err != nil {
		log.Log.Reason(err).Error("reply error")
	}
}

func reply(conn net.Conn, response []byte) error {
	_, err := conn.Write(response)
	if err != nil {
		return err
	}
	return nil
}
