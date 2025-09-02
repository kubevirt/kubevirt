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

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("domainstats", func() {
	Context("collector functions", func() {
		var metric = operatormetrics.NewCounter(
			operatormetrics.MetricOpts{
				Name: "test-metric-1",
				Help: "test help 1",
			},
		)

		var vmiReport = &VirtualMachineInstanceReport{
			vmi: &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi-1",
					Namespace: "test-ns-1",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "test-node-1",
				},
			},
		}

		It("newCollectorResultWithLabels should return a CollectorResult with the correct values", func() {
			cr := vmiReport.newCollectorResultWithLabels(metric, 5, map[string]string{"test-label-1": "test-value-1"})

			Expect(cr.Metric.GetOpts().Name).To(Equal("test-metric-1"))
			Expect(cr.Metric.GetOpts().Help).To(Equal("test help 1"))
			Expect(cr.Value).To(Equal(5.0))

			Expect(cr.ConstLabels).To(HaveKeyWithValue("node", "test-node-1"))
			Expect(cr.ConstLabels).To(HaveKeyWithValue("namespace", "test-ns-1"))
			Expect(cr.ConstLabels).To(HaveKeyWithValue("name", "test-vmi-1"))

			Expect(cr.ConstLabels).To(HaveKeyWithValue("test-label-1", "test-value-1"))
		})

		It("newCollectorResult should return a CollectorResult with the correct values", func() {
			cr := vmiReport.newCollectorResult(metric, 5)

			Expect(cr.Metric.GetOpts().Name).To(Equal("test-metric-1"))
			Expect(cr.Metric.GetOpts().Help).To(Equal("test help 1"))
			Expect(cr.Value).To(Equal(5.0))

			Expect(cr.ConstLabels).To(HaveKeyWithValue("node", "test-node-1"))
			Expect(cr.ConstLabels).To(HaveKeyWithValue("namespace", "test-ns-1"))
			Expect(cr.ConstLabels).To(HaveKeyWithValue("name", "test-vmi-1"))
		})
	})
})
