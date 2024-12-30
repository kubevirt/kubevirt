package metrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/metrics"
)

var _ = Describe("Operator Metrics", func() {
	Context("kubevirt_hco_system_health_status", func() {
		It("should set the correct system health status", func() {
			metrics.SetHCOMetricSystemHealthStatus(metrics.SystemHealthStatusError)
			v, err := metrics.GetHCOMetricSystemHealthStatus()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(metrics.SystemHealthStatusError))

			metrics.SetHCOMetricSystemHealthStatus(metrics.SystemHealthStatusWarning)
			v, err = metrics.GetHCOMetricSystemHealthStatus()
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(metrics.SystemHealthStatusWarning))
		})
	})
})
