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
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Stale VMI reconciliation metrics", func() {
	BeforeEach(func() {
		GetStaleVMIReconciliationsTotal().Reset()
	})

	AfterEach(func() {
		GetStaleVMIReconciliationsTotal().Reset()
	})

	It("should expose both terminal outcomes at zero immediately after setup", func() {
		DeferCleanup(func() {
			Expect(operatormetrics.CleanRegistry()).To(Succeed())
		})
		Expect(SetupMetrics("test-node", 1, nil, nil)).To(Succeed())

		// Gather without retrieving label values, which would create the series under test.
		registry := prometheus.NewRegistry()
		Expect(registry.Register(GetStaleVMIReconciliationsTotal())).To(Succeed())
		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(HaveLen(1))
		Expect(families[0].GetName()).To(Equal("kubevirt_virt_handler_stale_vmi_reconciliations_total"))
		Expect(families[0].Metric).To(HaveLen(2))

		results := map[string]float64{}
		for _, metric := range families[0].Metric {
			Expect(metric.Label).To(HaveLen(1))
			Expect(metric.Label[0].GetName()).To(Equal("result"))
			Expect(metric.Counter).ToNot(BeNil())
			results[metric.Label[0].GetValue()] = metric.Counter.GetValue()
		}
		Expect(results).To(Equal(map[string]float64{"cleaned": 0, "error": 0}))
	})

	It("should accumulate counts independently with only a result label", func() {
		results := []struct {
			result string
			label  string
			count  int
		}{
			{StaleVMIReconciliationCleaned, "cleaned", 1},
			{StaleVMIReconciliationError, "error", 3},
		}

		for _, result := range results {
			for i := 0; i < result.count; i++ {
				IncStaleVMIReconciliation(result.result)
			}
		}

		for _, result := range results {
			dto := &io_prometheus_client.Metric{}
			counter, err := GetStaleVMIReconciliationsTotal().GetMetricWithLabelValues(result.result)
			Expect(err).ToNot(HaveOccurred())
			Expect(counter).ToNot(BeNil())
			Expect(counter.Write(dto)).To(Succeed())
			Expect(*dto.Counter.Value).To(Equal(float64(result.count)))
			Expect(dto.Label).To(HaveLen(1))
			Expect(dto.Label[0].GetName()).To(Equal("result"))
			Expect(dto.Label[0].GetValue()).To(Equal(result.label))
		}
	})
})
