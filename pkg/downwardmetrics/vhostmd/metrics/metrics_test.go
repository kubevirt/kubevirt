package metrics

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"
)

var _ = Describe("metrics", func() {
	table.DescribeTable("should write", func(value interface{}, result string, metricType api.MetricType) {
		m := MustToMetric(value, "TotalCPUTime", "", api.MetricContextHost)
		Expect(m.Value).To(Equal(result))
		Expect(m.Type).To(Equal(metricType))
		Expect(m.Unit).To(BeEmpty())
	},
		table.Entry("string with proper type", "mystring", "mystring", api.MetricTypeString),
		table.Entry("int with proper type", 1, "1", api.MetricTypeInt64),
		table.Entry("uint with proper type", uint(1), "1", api.MetricTypeUInt64),
		table.Entry("int64 with proper type", int64(1), "1", api.MetricTypeInt64),
		table.Entry("uint64 with proper type", uint64(1), "1", api.MetricTypeUInt64),
		table.Entry("int32 with proper type", int32(1), "1", api.MetricTypeInt32),
		table.Entry("uint32 with proper type", uint32(1), "1", api.MetricTypeUInt32),
		table.Entry("float64 with proper type", float64(1292869.190000), "1292869.190000", api.MetricTypeReal64),
		table.Entry("float32 with proper type", float32(92869.1875), "92869.187500", api.MetricTypeReal32),
	)

	It("should set the unit", func() {
		m := MustToMetric(123, "TotalCPUTime", "s", api.MetricContextHost)
		Expect(m.Unit).To(Equal("s"))
	})
})
