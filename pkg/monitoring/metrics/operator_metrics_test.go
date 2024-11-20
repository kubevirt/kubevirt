package metrics_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/metrics"
)

var _ = Describe("Operator Metrics", func() {
	Context("kubevirt_hco_system_health_status", func() {
		It("should set system error reason correctly", func() {
			metrics.SetHCOSystemError("Reason1")
			v, err := metrics.GetHCOMetricSystemHealthStatus("Reason1")
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(metrics.SystemHealthStatusError))

			metrics.SetHCOSystemError("Reason2")
			v, err = metrics.GetHCOMetricSystemHealthStatus("Reason2")
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(metrics.SystemHealthStatusError))

			v, err = metrics.GetHCOMetricSystemHealthStatus("Reason1")
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal(metrics.SystemHealthStatusUnknown))
		})
	})
})
