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
	io_prometheus_client "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
