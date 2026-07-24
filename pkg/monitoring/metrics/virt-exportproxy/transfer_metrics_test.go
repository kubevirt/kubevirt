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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var _ = BeforeSuite(func() {
	Expect(operatormetrics.CleanRegistry()).To(Succeed())
	Expect(SetupMetrics()).To(Succeed())
})

var _ = AfterSuite(func() {
	Expect(operatormetrics.CleanRegistry()).To(Succeed())
})

var _ = Describe("Export proxy transfer metrics", func() {
	BeforeEach(func() {
		resetTransferMetricsState()
	})

	It("tracks active and total transfers", func() {
		totalBefore := getCounterValue(transfersTotal)
		transfer, ok := TryRecordTransferStarted()
		Expect(ok).To(BeTrue())
		Expect(ActiveTransferCount()).To(Equal(int64(1)))
		Expect(getGaugeValue(activeTransfers)).To(Equal(1.0))
		Expect(getCounterValue(transfersTotal)).To(Equal(totalBefore + 1.0))

		transfer.Finish()
		Expect(getGaugeValue(activeTransfers)).To(Equal(0.0))
		Expect(getCounterValue(transfersTotal)).To(Equal(totalBefore + 1.0))
	})

	It("accumulates transferred bytes", func() {
		body := NewCountingReadCloser(io.NopCloser(strings.NewReader("hello world")))
		data, err := io.ReadAll(body)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal([]byte("hello world")))
		Expect(getCounterValue(transferredBytesTotal)).To(Equal(float64(len("hello world"))))
	})

	It("resets throughput when the last transfer finishes", func() {
		transfer, ok := TryRecordTransferStarted()
		Expect(ok).To(BeTrue())
		recordTransferredBytes(1024)
		Expect(getGaugeValue(transferThroughputBytesPerSecond)).To(BeNumerically(">=", 0))

		transfer.Finish()
		Expect(getGaugeValue(transferThroughputBytesPerSecond)).To(Equal(0.0))
	})

	It("keeps throughput while other transfers remain active", func() {
		first, ok := TryRecordTransferStarted()
		Expect(ok).To(BeTrue())
		second, ok := TryRecordTransferStarted()
		Expect(ok).To(BeTrue())
		recordTransferredBytes(2048)

		first.Finish()
		Expect(atomic.LoadInt64(&activeTransferCount)).To(Equal(int64(1)))
		Expect(getGaugeValue(activeTransfers)).To(Equal(1.0))

		second.Finish()
		Expect(getGaugeValue(activeTransfers)).To(Equal(0.0))
		Expect(getGaugeValue(transferThroughputBytesPerSecond)).To(Equal(0.0))
	})

	It("does not reset throughput if a new transfer starts before reset completes", func() {
		transferThroughputBytesPerSecond.Set(42)
		throughputTracker.mu.Lock()
		throughputTracker.windowBytes = 8192
		throughputTracker.mu.Unlock()
		atomic.StoreInt64(&activeTransferCount, 1)

		resetThroughputIfIdle()

		Expect(getGaugeValue(transferThroughputBytesPerSecond)).To(Equal(42.0))
		throughputTracker.mu.Lock()
		Expect(throughputTracker.windowBytes).To(Equal(int64(8192)))
		throughputTracker.mu.Unlock()
	})
})

func getGaugeValue(metric *operatormetrics.Gauge) float64 {
	return getCollectorValue(metric.GetCollector()).GetGauge().GetValue()
}

func getCounterValue(metric *operatormetrics.Counter) float64 {
	return getCollectorValue(metric.GetCollector()).GetCounter().GetValue()
}

func getCollectorValue(collector prometheus.Collector) *dto.Metric {
	ch := make(chan prometheus.Metric, 1)
	collector.Collect(ch)
	close(ch)

	m, ok := <-ch
	ExpectWithOffset(1, ok).To(BeTrue())

	dm := &dto.Metric{}
	ExpectWithOffset(1, m.Write(dm)).To(Succeed())
	return dm
}
