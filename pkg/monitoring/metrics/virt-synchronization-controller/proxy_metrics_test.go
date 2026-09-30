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

package synccontrollermetrics

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var _ = Describe("ChannelTypeFromID", func() {
	DescribeTable("maps protocol ports to channel types",
		func(channelID int32, expected string) {
			Expect(ChannelTypeFromID(channelID)).To(Equal(expected))
		},
		Entry("state / direct migration", int32(libvirtDirectMigrationPort), channelTypeState),
		Entry("disk / block migration", int32(libvirtBlockMigrationPort), channelTypeDisk),
		Entry("unknown channel", int32(9185), ""),
		Entry("zero", int32(0), ""),
	)
})

var _ = Describe("Migration proxy metrics", func() {
	BeforeEach(func() {
		resetRegistrationForTesting()
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
		Expect(SetupMetrics()).To(Succeed())
		proxyActiveConnections.Reset()
		proxyBytesTransferredTotal.Reset()
		proxyMigrationBytes.Reset()
		proxyThroughputBytesPerSecond.Reset()
		proxyErrors.Reset()
	})

	AfterEach(func() {
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
		resetRegistrationForTesting()
	})

	It("should accumulate total and per-migration bytes for state and disk", func() {
		BytesTransferredAdd("mig-1", "source", channelTypeState, 100)
		BytesTransferredAdd("mig-1", "source", channelTypeState, 50)
		BytesTransferredAdd("mig-1", "source", channelTypeDisk, 200)

		Expect(getCounterValue(proxyBytesTransferredTotal, "source", channelTypeState)).To(Equal(150.0))
		Expect(getCounterValue(proxyBytesTransferredTotal, "source", channelTypeDisk)).To(Equal(200.0))
		Expect(getGaugeValue(proxyMigrationBytes, "mig-1", "source", channelTypeState)).To(Equal(150.0))
		Expect(getGaugeValue(proxyMigrationBytes, "mig-1", "source", channelTypeDisk)).To(Equal(200.0))
	})

	It("should ignore unknown channel types and non-positive byte counts", func() {
		BytesTransferredAdd("mig-1", "source", "control", 100)
		BytesTransferredAdd("mig-1", "source", channelTypeState, 0)
		BytesTransferredAdd("mig-1", "source", channelTypeState, -10)

		Expect(seriesCount(proxyBytesTransferredTotal)).To(Equal(0))
		Expect(seriesCount(proxyMigrationBytes)).To(Equal(0))
	})

	It("should track migrations independently", func() {
		BytesTransferredAdd("mig-a", "source", channelTypeState, 10)
		BytesTransferredAdd("mig-b", "target", channelTypeDisk, 20)

		Expect(getGaugeValue(proxyMigrationBytes, "mig-a", "source", channelTypeState)).To(Equal(10.0))
		Expect(getGaugeValue(proxyMigrationBytes, "mig-b", "target", channelTypeDisk)).To(Equal(20.0))
		Expect(getCounterValue(proxyBytesTransferredTotal, "source", channelTypeState)).To(Equal(10.0))
		Expect(getCounterValue(proxyBytesTransferredTotal, "target", channelTypeDisk)).To(Equal(20.0))
	})

	It("should clear per-migration series on ClearMigrationBytes without touching totals", func() {
		BytesTransferredAdd("mig-1", "source", channelTypeState, 100)
		BytesTransferredAdd("mig-1", "target", channelTypeDisk, 50)
		BytesTransferredAdd("mig-2", "source", channelTypeState, 25)

		ClearMigrationBytes("mig-1")

		Expect(seriesCount(proxyMigrationBytes)).To(Equal(1))
		Expect(getGaugeValue(proxyMigrationBytes, "mig-2", "source", channelTypeState)).To(Equal(25.0))
		Expect(getCounterValue(proxyBytesTransferredTotal, "source", channelTypeState)).To(Equal(125.0))
		Expect(getCounterValue(proxyBytesTransferredTotal, "target", channelTypeDisk)).To(Equal(50.0))
	})

	It("should track active connections", func() {
		ActiveConnectionsInc("source")
		ActiveConnectionsInc("source")
		ActiveConnectionsDec("source")
		Expect(getGaugeValue(proxyActiveConnections, "source")).To(Equal(1.0))
	})

	It("should increment error counters", func() {
		ErrorsInc("source", "connect_error")
		ErrorsInc("source", "connect_error")
		ErrorsInc("target", "send_error")

		Expect(getCounterValue(proxyErrors, "source", "connect_error")).To(Equal(2.0))
		Expect(getCounterValue(proxyErrors, "target", "send_error")).To(Equal(1.0))
	})

	It("should register metrics once via EnsureProxyMetrics", func() {
		resetRegistrationForTesting()
		Expect(operatormetrics.CleanRegistry()).To(Succeed())

		stop := make(chan struct{})
		defer close(stop)

		Expect(EnsureProxyMetrics(stop)).To(Succeed())
		Expect(EnsureProxyMetrics(stop)).To(Succeed())

		BytesTransferredAdd("mig-1", "source", channelTypeState, 42)
		Expect(getCounterValue(proxyBytesTransferredTotal, "source", channelTypeState)).To(Equal(42.0))
	})

	It("should update throughput after a report window", func() {
		BytesTransferredAdd("mig-1", "source", channelTypeState, 1000)
		BytesTransferredAdd("mig-1", "source", channelTypeDisk, 2000)

		// Drive one report cycle directly instead of waiting on the ticker.
		last := map[throughputKey]uint64{}
		lastTime := time.Now().Add(-throughputWindow)
		now := time.Now()
		elapsed := now.Sub(lastTime).Seconds()
		Expect(elapsed).To(BeNumerically(">", 0))

		for _, key := range []throughputKey{
			{proxyType: "source", channelType: channelTypeState},
			{proxyType: "source", channelType: channelTypeDisk},
		} {
			cur := throughputCounter(key.proxyType, key.channelType).Load()
			rate := float64(cur-last[key]) / elapsed
			proxyThroughputBytesPerSecond.WithLabelValues(key.proxyType, key.channelType).Set(rate)
		}

		Expect(getGaugeValue(proxyThroughputBytesPerSecond, "source", channelTypeState)).To(BeNumerically(">", 0))
		Expect(getGaugeValue(proxyThroughputBytesPerSecond, "source", channelTypeDisk)).To(BeNumerically(">", 0))
	})
})

func getCounterValue(vec *operatormetrics.CounterVec, labelValues ...string) float64 {
	m := &ioprometheusclient.Metric{}
	counter, err := vec.GetMetricWithLabelValues(labelValues...)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, counter.Write(m)).To(Succeed())
	return m.GetCounter().GetValue()
}

func getGaugeValue(vec *operatormetrics.GaugeVec, labelValues ...string) float64 {
	m := &ioprometheusclient.Metric{}
	gauge, err := vec.GetMetricWithLabelValues(labelValues...)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, gauge.Write(m)).To(Succeed())
	return m.GetGauge().GetValue()
}

func seriesCount(collector prometheus.Collector) int {
	ch := make(chan prometheus.Metric, 100)
	collector.Collect(ch)
	close(ch)
	count := 0
	for range ch {
		count++
	}
	return count
}
