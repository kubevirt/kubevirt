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
	"sync"
	"sync/atomic"
	"time"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
)

var (
	transferMetrics = []operatormetrics.Metric{
		activeTransfers,
		transfersTotal,
		transferredBytesTotal,
		transferThroughputBytesPerSecond,
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

	transferThroughputBytesPerSecond = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_exportproxy_transfer_throughput_bytes_per_second",
			Help: "Current aggregate export transfer throughput in bytes per second.",
		},
	)

	activeTransferCount int64

	throughputTracker struct {
		mu          sync.Mutex
		windowStart time.Time
		windowBytes int64
	}
)

type activeTransfer struct{}

// Finish marks the end of an active export transfer.
func (activeTransfer) Finish() {
	RecordTransferFinished()
}

// ActiveTransferCount returns the number of export transfers currently proxied.
func ActiveTransferCount() int64 {
	return atomic.LoadInt64(&activeTransferCount)
}

// TryRecordTransferStarted increments active and total transfer counters when the
// pod is below SoftTransferLimit. Returns false without incrementing when at capacity.
func TryRecordTransferStarted() (activeTransfer, bool) {
	for {
		current := atomic.LoadInt64(&activeTransferCount)
		if current >= admission.SoftTransferLimit {
			return activeTransfer{}, false
		}
		if atomic.CompareAndSwapInt64(&activeTransferCount, current, current+1) {
			activeTransfers.Inc()
			transfersTotal.Inc()
			return activeTransfer{}, true
		}
	}
}

// RecordTransferFinished decrements the active transfer counter and resets
// throughput tracking when no transfers remain active.
func RecordTransferFinished() {
	if atomic.AddInt64(&activeTransferCount, -1) == 0 {
		resetThroughputIfIdle()
	}
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
		recordTransferredBytes(int64(n))
	}
	return n, err
}

func recordTransferredBytes(n int64) {
	transferredBytesTotal.Add(float64(n))

	throughputTracker.mu.Lock()
	defer throughputTracker.mu.Unlock()

	now := time.Now()
	if throughputTracker.windowStart.IsZero() {
		throughputTracker.windowStart = now
	}
	throughputTracker.windowBytes += n

	elapsed := now.Sub(throughputTracker.windowStart).Seconds()
	if elapsed >= 1 {
		transferThroughputBytesPerSecond.Set(float64(throughputTracker.windowBytes) / elapsed)
		throughputTracker.windowStart = now
		throughputTracker.windowBytes = 0
	}
}

func resetThroughputIfIdle() {
	throughputTracker.mu.Lock()
	defer throughputTracker.mu.Unlock()

	if atomic.LoadInt64(&activeTransferCount) != 0 {
		return
	}

	throughputTracker.windowStart = time.Time{}
	throughputTracker.windowBytes = 0
	transferThroughputBytesPerSecond.Set(0)
}

func resetThroughputWindow() {
	throughputTracker.mu.Lock()
	defer throughputTracker.mu.Unlock()

	throughputTracker.windowStart = time.Time{}
	throughputTracker.windowBytes = 0
}
