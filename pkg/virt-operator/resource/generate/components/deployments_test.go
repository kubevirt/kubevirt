package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Deployments", func() {
	It("should create Prometheus service that is headless", func() {
		service := NewPrometheusService("mynamespace")
		Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		Expect(service.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))
	})
})
