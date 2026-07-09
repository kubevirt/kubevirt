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

package virtexportproxy

import (
	"io"
	"sync/atomic"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	transferMetrics = []operatormetrics.Metric{
		activeTransfers,
		transfersTotal,
		transferredBytesTotal,
	}

	activeTransfers = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_exportproxy_active_transfers",
			Help: "Number of export transfers currently being proxied.",
		},
	)

	transfersTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_exportproxy_transfers_total",
			Help: "Total number of export transfers handled by the proxy since startup, including active, completed, and failed transfers.",
		},
	)

	transferredBytesTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_exportproxy_transferred_bytes_total",
			Help: "Total number of bytes transferred by the export proxy since startup.",
		},
	)

	activeTransferCount int64
)

type activeTransfer struct{}

// Finish marks the end of an active export transfer.
func (activeTransfer) Finish() {
	RecordTransferFinished()
}

// RecordTransferStarted increments active and total transfer counters and returns
// a handle that must be finished when the transfer completes.
func RecordTransferStarted() activeTransfer {
	atomic.AddInt64(&activeTransferCount, 1)
	activeTransfers.Inc()
	transfersTotal.Inc()
	return activeTransfer{}
}

// RecordTransferFinished decrements the active transfer counter.
func RecordTransferFinished() {
	atomic.AddInt64(&activeTransferCount, -1)
	activeTransfers.Dec()
}

// NewCountingReadCloser wraps a response body and records transferred bytes.
func NewCountingReadCloser(body io.ReadCloser) io.ReadCloser {
	if body == nil {
		return nil
	}
	return &countingReadCloser{ReadCloser: body}
}

type countingReadCloser struct {
	io.ReadCloser
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	if n > 0 {
		transferredBytesTotal.Add(float64(n))
	}
	return n, err
}
