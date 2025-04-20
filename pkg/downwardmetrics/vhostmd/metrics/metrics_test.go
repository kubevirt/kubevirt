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

package metrics

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
)

var _ = Describe("metrics", func() {
	DescribeTable("should write", func(value interface{}, result string, metricType api.MetricType) {
		m := MustToMetric(value, "TotalCPUTime", "", api.MetricContextHost)
		Expect(m.Value).To(Equal(result))
		Expect(m.Type).To(Equal(metricType))
		Expect(m.Unit).To(BeEmpty())
	},
		Entry("string with proper type", "mystring", "mystring", api.MetricTypeString),
		Entry("int with proper type", 1, "1", api.MetricTypeInt64),
		Entry("uint with proper type", uint(1), "1", api.MetricTypeUInt64),
		Entry("int64 with proper type", int64(1), "1", api.MetricTypeInt64),
		Entry("uint64 with proper type", uint64(1), "1", api.MetricTypeUInt64),
		Entry("int32 with proper type", int32(1), "1", api.MetricTypeInt32),
		Entry("uint32 with proper type", uint32(1), "1", api.MetricTypeUInt32),
		Entry("float64 with proper type", float64(1292869.190000), "1292869.190000", api.MetricTypeReal64),
		Entry("float32 with proper type", float32(92869.1875), "92869.187500", api.MetricTypeReal32),
	)

	It("should set the unit", func() {
		m := MustToMetric(123, "TotalCPUTime", "s", api.MetricContextHost)
		Expect(m.Unit).To(Equal("s"))
	})
})
