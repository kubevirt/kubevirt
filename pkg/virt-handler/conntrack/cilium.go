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

package conntrack

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"

	"kubevirt.io/client-go/log"
)

const (
	DefaultCiliumSocketPath      = "/var/run/cilium/cilium.sock"
	conntrackExportVersionHeader = "Cilium-Conntrack-Export-Version"

	ciliumAPIBaseURL        = "http://localhost"
	conntrackExportEndpoint = "/v1/conntrack/export"
	conntrackImportEndpoint = "/v1/conntrack/import"
)

type ConntrackClient interface {
	ExportConntrack(ctx context.Context, ip4 string) (*ExportResult, error)
	ImportConntrack(ctx context.Context, data []byte, version byte) error
}

type CiliumClient struct {
	socketPath string
	httpClient *http.Client
}

type ExportResult struct {
	Data    []byte
	Version byte
}

func NewCiliumClient() *CiliumClient {
	return NewCiliumClientWithSocket(DefaultCiliumSocketPath)
}

func NewCiliumClientWithSocket(socketPath string) *CiliumClient {
	return &CiliumClient{
		socketPath: socketPath,
		httpClient: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

func (c *CiliumClient) IsAvailable() bool {
	_, err := os.Stat(c.socketPath)
	return err == nil
}

func (c *CiliumClient) ExportConntrack(ctx context.Context, ip4 string) (*ExportResult, error) {
	url := fmt.Sprintf("%s%s?ip4=%s", ciliumAPIBaseURL, conntrackExportEndpoint, ip4)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create export request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute export request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("export failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read export response: %w", err)
	}

	versionStr := resp.Header.Get(conntrackExportVersionHeader)
	version := parseVersion(versionStr)

	return &ExportResult{
		Data:    data,
		Version: version,
	}, nil
}

func parseVersion(versionStr string) byte {
	if versionStr == "" {
		return 1
	}
	v, err := strconv.ParseUint(versionStr, 10, 8)
	if err != nil {
		log.Log.Warningf("Conntrack sync: failed to parse version '%s', defaulting to 1: %v", versionStr, err)
		return 1
	}
	return byte(v)
}

func (c *CiliumClient) ImportConntrack(ctx context.Context, data []byte, version byte) error {
	url := ciliumAPIBaseURL + conntrackImportEndpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create import request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set(conntrackExportVersionHeader, strconv.FormatUint(uint64(version), 10))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute import request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("import failed with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
