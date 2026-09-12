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

package virthandler

import (
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Guest OS panic metrics", func() {
	BeforeEach(func() {
		guestOSPanicTotal.Reset()
	})

	It("should increment the counter for a VMI", func() {
		IncGuestOSPanic("test-ns", "test-vmi", "hyper-v", 0x7e)

		dto := &io_prometheus_client.Metric{}
		counter, err := guestOSPanicTotal.GetMetricWithLabelValues("test-ns", "test-vmi", "hyper-v", "0x7e")
		Expect(err).ToNot(HaveOccurred())
		Expect(counter).ToNot(BeNil())
		Expect(counter.Write(dto)).To(Succeed())
		Expect(*dto.Counter.Value).To(Equal(1.0))
	})

	It("should accumulate counts across multiple panics", func() {
		IncGuestOSPanic("test-ns", "test-vmi", "unknown", 0)
		IncGuestOSPanic("test-ns", "test-vmi", "unknown", 0)
		IncGuestOSPanic("test-ns", "test-vmi", "unknown", 0)

		dto := &io_prometheus_client.Metric{}
		counter, err := guestOSPanicTotal.GetMetricWithLabelValues("test-ns", "test-vmi", "unknown", "unknown")
		Expect(err).ToNot(HaveOccurred())
		Expect(counter).ToNot(BeNil())
		Expect(counter.Write(dto)).To(Succeed())
		Expect(*dto.Counter.Value).To(Equal(3.0))
	})

	It("should track different VMIs and panic types independently", func() {
		IncGuestOSPanic("ns-a", "vmi-1", "hyper-v", 0x7e)
		IncGuestOSPanic("ns-a", "vmi-1", "hyper-v", 0x7e)
		IncGuestOSPanic("ns-b", "vmi-2", "unknown", 0)

		dto1 := &io_prometheus_client.Metric{}
		counter1, err := guestOSPanicTotal.GetMetricWithLabelValues("ns-a", "vmi-1", "hyper-v", "0x7e")
		Expect(err).ToNot(HaveOccurred())
		Expect(counter1).ToNot(BeNil())
		Expect(counter1.Write(dto1)).To(Succeed())
		Expect(*dto1.Counter.Value).To(Equal(2.0))

		dto2 := &io_prometheus_client.Metric{}
		counter2, err := guestOSPanicTotal.GetMetricWithLabelValues("ns-b", "vmi-2", "unknown", "unknown")
		Expect(err).ToNot(HaveOccurred())
		Expect(counter2).ToNot(BeNil())
		Expect(counter2.Write(dto2)).To(Succeed())
		Expect(*dto2.Counter.Value).To(Equal(1.0))
	})
})

var _ = Describe("Guest OS termination metrics", func() {
	const (
		namespace = "test-namespace"
		name      = "test-vmi"
	)

	BeforeEach(func() {
		guestOSTerminationTotal.Reset()
	})

	DescribeTable("should report supported guest OS termination reasons",
		func(reason api.TerminationReason) {
			IncGuestOSTermination(namespace, name, reason)

			Expect(getGuestOSTerminationCounterValue(namespace, name, reason)).To(Equal(float64(1)))
		},
		Entry(string(api.TerminationReasonGuestShutdown), api.TerminationReasonGuestShutdown),
		Entry(string(api.TerminationReasonPlatformRequestedShutdown), api.TerminationReasonPlatformRequestedShutdown),
		Entry(string(api.TerminationReasonHostShutdown), api.TerminationReasonHostShutdown),
		Entry(string(api.TerminationReasonHostStoppedFailed), api.TerminationReasonHostStoppedFailed),
		Entry(string(api.TerminationReasonGuestCrashed), api.TerminationReasonGuestCrashed),
	)

	It("should partition guest OS termination counters by VMI", func() {
		reason := api.TerminationReasonGuestShutdown

		IncGuestOSTermination(namespace, "vmi-a", reason)
		IncGuestOSTermination(namespace, "vmi-b", reason)
		IncGuestOSTermination(namespace, "vmi-b", reason)

		Expect(getGuestOSTerminationCounterValue(namespace, "vmi-a", reason)).To(Equal(float64(1)))
		Expect(getGuestOSTerminationCounterValue(namespace, "vmi-b", reason)).To(Equal(float64(2)))
	})

	It("should ignore unsupported guest OS termination reasons", func() {
		IncGuestOSTermination(namespace, name, "")
		IncGuestOSTermination(namespace, name, "UnexpectedReason")

		Expect(collectedGuestOSTerminationMetrics()).To(BeEmpty())
	})

	It("should delete all guest OS termination metrics for a VMI", func() {
		for _, reason := range api.SupportedTerminationReasons() {
			IncGuestOSTermination(namespace, "deleted-vmi", reason)
			IncGuestOSTermination(namespace, "kept-vmi", reason)
		}

		DeleteGuestOSTerminationMetrics(namespace, "deleted-vmi")

		for _, reason := range api.SupportedTerminationReasons() {
			Expect(hasGuestOSTerminationMetric(namespace, "deleted-vmi", reason)).To(BeFalse())
			Expect(hasGuestOSTerminationMetric(namespace, "kept-vmi", reason)).To(BeTrue())
		}
	})
})

func getGuestOSTerminationCounterValue(namespace, name string, reason api.TerminationReason) float64 {
	metric, err := guestOSTerminationTotal.GetMetricWithLabelValues(namespace, name, string(reason))
	Expect(err).ToNot(HaveOccurred())

	dto := &io_prometheus_client.Metric{}
	Expect(metric.Write(dto)).To(Succeed())

	return dto.Counter.GetValue()
}

func collectedGuestOSTerminationMetrics() []prometheus.Metric {
	metrics := make(chan prometheus.Metric)
	go func() {
		guestOSTerminationTotal.Collect(metrics)
		close(metrics)
	}()

	var result []prometheus.Metric
	for metric := range metrics {
		result = append(result, metric)
	}
	return result
}

func hasGuestOSTerminationMetric(namespace, name string, reason api.TerminationReason) bool {
	for _, metric := range collectedGuestOSTerminationMetricDTOs() {
		if guestMetricLabelValue(metric, "namespace") == namespace &&
			guestMetricLabelValue(metric, "name") == name &&
			guestMetricLabelValue(metric, "reason") == string(reason) {
			return true
		}
	}
	return false
}

func collectedGuestOSTerminationMetricDTOs() []*io_prometheus_client.Metric {
	metrics := collectedGuestOSTerminationMetrics()
	result := make([]*io_prometheus_client.Metric, 0, len(metrics))
	for _, metric := range metrics {
		dto := &io_prometheus_client.Metric{}
		Expect(metric.Write(dto)).To(Succeed())
		result = append(result, dto)
	}
	return result
}

func guestMetricLabelValue(metric *io_prometheus_client.Metric, name string) string {
	for _, label := range metric.GetLabel() {
		if label.GetName() == name {
			return label.GetValue()
		}
	}
	return ""
}
