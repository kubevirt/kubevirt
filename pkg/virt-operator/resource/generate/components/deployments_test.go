package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Deployments", func() {
	It("should create Prometheus service that is headless", func() {
		service := NewPrometheusService("mynamespace")
		Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		Expect(service.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))
	})

	It("should probe virt-exportproxy readiness on /readyz and liveness on /healthz", func() {
		config := &operatorutil.KubeVirtDeploymentConfig{
			Namespace: "kubevirt",
		}
		deployment := NewExportProxyDeployment(config, "", "", "")
		container := deployment.Spec.Template.Spec.Containers[0]

		Expect(container.ReadinessProbe).NotTo(BeNil())
		Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/readyz"))
		Expect(container.LivenessProbe).NotTo(BeNil())
		Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/healthz"))
	})
})
