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

package vmisync

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	ioprometheusclient "github.com/prometheus/client_model/go"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var _ = Describe("VMI sync metrics", func() {
	BeforeEach(func() {
		Expect(SetupMetrics()).To(Succeed())
		vmiSyncTotal.Reset()
	})

	AfterEach(func() {
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
	})

	It("should increment the counter when VMISynced is called", func() {
		VMISynced("test-ns", "test-vmi")

		Expect(getCounterValue("test-ns", "test-vmi")).To(Equal(1.0))
	})

	It("should accumulate the counter across multiple VMISynced calls", func() {
		VMISynced("test-ns", "test-vmi")
		VMISynced("test-ns", "test-vmi")
		VMISynced("test-ns", "test-vmi")

		Expect(getCounterValue("test-ns", "test-vmi")).To(Equal(3.0))
	})

	It("should remove the metric series when ResetVMISync is called", func() {
		VMISynced("test-ns", "test-vmi")
		Expect(activeSeriesCount()).To(Equal(1))

		ResetVMISync("test-ns/test-vmi")
		Expect(activeSeriesCount()).To(Equal(0))
	})

	It("should only remove the specified VMI's series on ResetVMISync", func() {
		VMISynced("test-ns", "vmi-1")
		VMISynced("test-ns", "vmi-2")

		ResetVMISync("test-ns/vmi-1")
		Expect(activeSeriesCount()).To(Equal(1))
	})

	It("should accumulate syncs from multiple controllers into a single series per VMI", func() {
		VMISynced("test-ns", "test-vmi") // VirtualMachineController
		VMISynced("test-ns", "test-vmi") // MigrationSourceController
		VMISynced("test-ns", "test-vmi") // MigrationTargetController

		Expect(getCounterValue("test-ns", "test-vmi")).To(Equal(3.0))
		Expect(activeSeriesCount()).To(Equal(1))
	})
})

func getCounterValue(namespace, name string) float64 {
	m := &ioprometheusclient.Metric{}
	counter, err := vmiSyncTotal.GetMetricWithLabelValues(namespace, name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, counter.Write(m)).To(Succeed())
	return m.GetCounter().GetValue()
}

func activeSeriesCount() int {
	ch := make(chan prometheus.Metric, 100)
	vmiSyncTotal.Collect(ch)
	close(ch)
	count := 0
	for range ch {
		count++
	}
	return count
}
