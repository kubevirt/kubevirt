package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	"k8s.io/apimachinery/pkg/api/resource"
)

var _ = Describe("Deployments", func() {
	It("should create Prometheus service that is headless", func() {
		By("Creating Prometheus service")
		service := NewPrometheusService("mynamespace")

		By("Verifying service is ClusterIP type")
		Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))

		By("Verifying service is headless")
		Expect(service.Spec.ClusterIP).To(Equal(corev1.ClusterIPNone))
	})

	Describe("NewSynchronizationControllerDeployment", func() {
		const (
			namespace = "kubevirt"
		)

		DescribeTable("Network attachment configuration",
			func(setupConfig func(*util.KubeVirtDeploymentConfig), matchExpectedHasAnnotation, matchExpectedAnnotation gomegatypes.GomegaMatcher) {
				config := &util.KubeVirtDeploymentConfig{
					Namespace: namespace,
				}

				By("Setting up configuration")
				setupConfig(config)

				By("Creating synchronization controller deployment")
				deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
				Expect(deployment).ToNot(BeNil())

				By("Verifying network attachment annotation")
				annotation, hasAnnotation := deployment.Spec.Template.ObjectMeta.Annotations[networkv1.NetworkAttachmentAnnot]

				Expect(hasAnnotation).To(matchExpectedHasAnnotation)
				Expect(annotation).To(matchExpectedAnnotation)
			},
			Entry("no cross-cluster network configured",
				func(config *util.KubeVirtDeploymentConfig) {
					// No additional configuration needed
				},
				BeFalseBecause("no cross cluster network is configured"),
				BeEmpty(),
			),
			Entry("cross-cluster network configured",
				func(config *util.KubeVirtDeploymentConfig) {
					config.AdditionalProperties = map[string]string{
						util.AdditionalPropertiesCrossClusterMigrationNetwork: "test-crosscluster-network",
					}
				},
				BeTrueBecause("just cross cluster network is configured"),
				Equal("test-crosscluster-network@"+virtv1.CrossClusterMigrationInterfaceName),
			),
			Entry("both migration and cross-cluster networks configured",
				func(config *util.KubeVirtDeploymentConfig) {
					config.AdditionalProperties = map[string]string{
						util.AdditionalPropertiesMigrationNetwork:             "migration-network",
						util.AdditionalPropertiesCrossClusterMigrationNetwork: "crosscluster-network",
					}
				},
				BeTrueBecause("both migration and cross cluster networks are configured"),
				Equal("migration-network@"+virtv1.MigrationInterfaceName+","+"crosscluster-network@"+virtv1.CrossClusterMigrationInterfaceName),
			),
		)

		It("should defer node placement to reconciliation", func() {
			By("Creating config with SynchronizationPlacement that would previously be applied at generation")
			config := &util.KubeVirtDeploymentConfig{
				Namespace: namespace,
				SynchronizationPlacement: &virtv1.ComponentConfig{
					NodePlacement: &virtv1.NodePlacement{
						NodeSelector: map[string]string{
							"kubevirt.io/crosscluster-access": "true",
						},
						Tolerations: []corev1.Toleration{
							{
								Key:      "dedicated",
								Operator: corev1.TolerationOpEqual,
								Value:    "kubevirt",
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
						Affinity: &corev1.Affinity{
							NodeAffinity: &corev1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
									NodeSelectorTerms: []corev1.NodeSelectorTerm{
										{
											MatchExpressions: []corev1.NodeSelectorRequirement{
												{
													Key:      "topology.kubernetes.io/zone",
													Operator: corev1.NodeSelectorOpIn,
													Values:   []string{"us-east-1a"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}

			By("Creating synchronization controller deployment")
			deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
			Expect(deployment).ToNot(BeNil())

			By("Verifying custom placement is not applied at generation")
			Expect(deployment.Spec.Template.Spec.NodeSelector).To(BeNil())
			Expect(deployment.Spec.Template.Spec.Tolerations).To(ContainElement(corev1.Toleration{
				Key:      "CriticalAddonsOnly",
				Operator: corev1.TolerationOpExists,
			}))
			Expect(deployment.Spec.Template.Spec.Tolerations).NotTo(ContainElement(corev1.Toleration{
				Key:      "dedicated",
				Operator: corev1.TolerationOpEqual,
				Value:    "kubevirt",
				Effect:   corev1.TaintEffectNoSchedule,
			}))

			By("Verifying default control-plane placement is not applied at generation")
			Expect(deployment.Spec.Template.Spec.Affinity).ToNot(BeNil())
			Expect(deployment.Spec.Template.Spec.Affinity.PodAntiAffinity).ToNot(BeNil())
			Expect(deployment.Spec.Template.Spec.Affinity.NodeAffinity).To(BeNil())
			for _, tol := range deployment.Spec.Template.Spec.Tolerations {
				Expect(tol.Key).NotTo(Equal("node-role.kubernetes.io/control-plane"))
			}
		})
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
				d := NewApiServerDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtControllerName, func() corev1.PodSpec {
				d := NewControllerDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtOperatorName, func() corev1.PodSpec {
				d := NewOperatorDeployment("", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", corev1.PullIfNotPresent)
				return d.Spec.Template.Spec
			}),
			Entry(VirtExportProxyName, func() corev1.PodSpec {
				d := NewExportProxyDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtSynchronizationControllerName, func() corev1.PodSpec {
				d := NewSynchronizationControllerDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
		)

		It("should have emptyDir /tmp volume on virt-api", func() {
			d := NewApiServerDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
			podSpec := d.Spec.Template.Spec

			Expect(podSpec.Volumes).To(ContainElement(corev1.Volume{
				Name: tmpDirName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: new(resource.MustParse(tmpDirSizeLimit)),
					},
				},
			}),
				"virt-api should have a size-limited tmp-dir emptyDir volume")

			for _, container := range podSpec.Containers {
				Expect(container.VolumeMounts).To(ContainElement(corev1.VolumeMount{
					Name:      tmpDirName,
					MountPath: tmpDirMountPath,
				}), "container %s should mount tmp-dir at /tmp", container.Name)
			}
		})

		DescribeTable("should NOT have tmp-dir volume on",
			func(createDeployment func() corev1.PodSpec) {
				podSpec := createDeployment()
				Expect(podSpec.Volumes).ToNot(
					ContainElement(HaveField("Name", tmpDirName)),
					"deployment should not have tmp-dir volume")
			},
			Entry(VirtControllerName, func() corev1.PodSpec {
				d := NewControllerDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtOperatorName, func() corev1.PodSpec {
				d := NewOperatorDeployment("", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", "", corev1.PullIfNotPresent)
				return d.Spec.Template.Spec
			}),
			Entry(VirtExportProxyName, func() corev1.PodSpec {
				d := NewExportProxyDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
			Entry(VirtSynchronizationControllerName, func() corev1.PodSpec {
				d := NewSynchronizationControllerDeployment(&util.KubeVirtDeploymentConfig{}, "", "", "")
				return d.Spec.Template.Spec
			}),
		)
	})
})
