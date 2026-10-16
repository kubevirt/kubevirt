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
	"strings"
	"sync"
	"sync/atomic"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/exportproxy/admission"
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
			Help: "Number of admitted export transfers currently being proxied.",
		},
	)

	transfersTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_exportproxy_transfers_total",
			Help: "Total admitted export transfers since startup. " +
				"Excludes unauthorized and forbidden responses.",
		},
	)

	transferredBytesTotal = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_exportproxy_transferred_bytes_total",
			Help: "Total number of bytes transferred by the export proxy since startup.",
		},
	)

	activeTransferCount int64
	draining            atomic.Bool
)

// ActiveTransfer holds a soft-admission slot. Reserve with
// TryRecordTransferStarted, then either Cancel (auth failure / dial error) or
// Confirm + Finish (successful transfer).
type ActiveTransfer struct {
	once sync.Once
}

// Finish releases a reserved or confirmed transfer slot.
func (t *ActiveTransfer) Finish() {
	t.release()
}

// Cancel releases a reserved slot without counting it as an admitted transfer.
func (t *ActiveTransfer) Cancel() {
	t.release()
}

// Confirm marks a reserved transfer as admitted (increments transfers_total).
// Call after the backend accepts the request (non-401/403).
func (t *ActiveTransfer) Confirm() {
	transfersTotal.Inc()
}

func (t *ActiveTransfer) release() {
	t.once.Do(RecordTransferFinished)
}

// ActiveTransferCount returns the number of export transfers currently proxied.
func ActiveTransferCount() int64 {
	return atomic.LoadInt64(&activeTransferCount)
}

// BeginDrain marks the process as shutting down: readiness fails and new
// transfers are rejected so in-flight exports can complete before SIGKILL.
func BeginDrain() {
	draining.Store(true)
}

// IsDraining reports whether the process is draining for shutdown.
func IsDraining() bool {
	return draining.Load()
}

// TryRecordTransferStarted reserves a soft-admission slot when the pod is below
// SoftTransferLimit, soft CPU/memory thresholds, and not draining.
// Returns false without reserving when at capacity.
//
// Call Cancel on dial/auth failure, or Confirm then Finish (via
// NewAdmittedTransferBody) for a successful transfer.
func TryRecordTransferStarted() (*ActiveTransfer, bool) {
	if IsDraining() {
		return nil, false
	}
	if admission.OverSoftUtilizationLimit() {
		return nil, false
	}

	for {
		current := atomic.LoadInt64(&activeTransferCount)
		if current >= admission.SoftTransferLimit {
			return nil, false
		}
		if atomic.CompareAndSwapInt64(&activeTransferCount, current, current+1) {
			activeTransfers.Inc()
			return &ActiveTransfer{}, true
		}
	}
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

// NewAdmittedTransferBody wraps a response body for an admitted transfer: it
// counts transferred bytes and calls finish exactly once when the body is closed.
func NewAdmittedTransferBody(body io.ReadCloser, finish func()) io.ReadCloser {
	if body == nil {
		body = io.NopCloser(strings.NewReader(""))
	}
	return &admittedTransferBody{
		ReadCloser: NewCountingReadCloser(body),
		finish:     finish,
	}
}

// ResetTransferMetricsForTest clears transfer counters for unit tests.
func ResetTransferMetricsForTest() {
	atomic.StoreInt64(&activeTransferCount, 0)
	readinessShedding.Store(false)
	draining.Store(false)
	activeTransfers.Set(0)
	admission.ResetUtilizationReaderForTest()
}

// SetActiveTransferCountForTest sets the active transfer counter for unit tests.
func SetActiveTransferCountForTest(count int64) {
	atomic.StoreInt64(&activeTransferCount, count)
	activeTransfers.Set(float64(count))
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

type admittedTransferBody struct {
	io.ReadCloser
	finish func()
	once   sync.Once
}

func (b *admittedTransferBody) Close() error {
	err := b.ReadCloser.Close()
	b.once.Do(b.finish)
	return err
}
