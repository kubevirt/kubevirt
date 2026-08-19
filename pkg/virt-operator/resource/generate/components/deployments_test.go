package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"

	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Deployments", func() {
	It("should create Prometheus service that is headless", func() {
		service := NewPrometheusService("mynamespace")
		Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
		Expect(service.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))
	})

	Context("SecurityContext", func() {
		DescribeTable("should set ReadOnlyRootFilesystem to true on",
			func(createDeployment func() corev1.PodSpec) {
				podSpec := createDeployment()
				for _, c := range podSpec.Containers {
					Expect(c.SecurityContext).ToNot(BeNil(),
						"container %s should have SecurityContext", c.Name)
					Expect(c.SecurityContext.ReadOnlyRootFilesystem).ToNot(BeNil(),
						"container %s should have ReadOnlyRootFilesystem set", c.Name)
					Expect(*c.SecurityContext.ReadOnlyRootFilesystem).To(BeTrue(),
						"container %s should have ReadOnlyRootFilesystem=true", c.Name)
				}
			},
			Entry(VirtAPIName, func() corev1.PodSpec {
				d := NewApiServerDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtControllerName, func() corev1.PodSpec {
				d := NewControllerDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtOperatorName, func() corev1.PodSpec {
				d := NewOperatorDeployment("", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", corev1.PullIfNotPresent)
				return d.Spec.Template.Spec
			}),
			Entry(VirtExportProxyName, func() corev1.PodSpec {
				d := NewExportProxyDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtSynchronizationControllerName, func() corev1.PodSpec {
				d := NewSynchronizationControllerDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
		)

		It("should have emptyDir /tmp volume on virt-api", func() {
			d := NewApiServerDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
			podSpec := d.Spec.Template.Spec

			hasTmpVolume := false
			for _, vol := range podSpec.Volumes {
				if vol.Name == TmpDirName {
					Expect(vol.VolumeSource.EmptyDir).ToNot(BeNil())
					hasTmpVolume = true
					break
				}
			}
			Expect(hasTmpVolume).To(BeTrue(),
				"virt-api should have tmp-dir emptyDir volume for os.MkdirTemp usage")

			for _, c := range podSpec.Containers {
				hasTmpMount := false
				for _, m := range c.VolumeMounts {
					if m.Name == TmpDirName && m.MountPath == TmpDirMountPath {
						hasTmpMount = true
						break
					}
				}
				Expect(hasTmpMount).To(BeTrue(),
					"container %s should mount tmp-dir at /tmp", c.Name)
			}
		})

		DescribeTable("should NOT have tmp-dir volume on",
			func(createDeployment func() corev1.PodSpec) {
				podSpec := createDeployment()
				for _, vol := range podSpec.Volumes {
					Expect(vol.Name).ToNot(Equal(TmpDirName),
						"deployment should not have tmp-dir volume (minimalist approach)")
				}
			},
			Entry(VirtControllerName, func() corev1.PodSpec {
				d := NewControllerDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtOperatorName, func() corev1.PodSpec {
				d := NewOperatorDeployment("", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", corev1.PullIfNotPresent)
				return d.Spec.Template.Spec
			}),
			Entry(VirtExportProxyName, func() corev1.PodSpec {
				d := NewExportProxyDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtSynchronizationControllerName, func() corev1.PodSpec {
				d := NewSynchronizationControllerDeployment(&operatorutil.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
		)
	})
})
