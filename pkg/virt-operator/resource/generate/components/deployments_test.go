package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/util"
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

		DescribeTable("Synchronization placement",
			func(setupPlacement func() *virtv1.ComponentConfig, verifyPlacement func(*corev1.PodSpec)) {
				By("Setting up placement configuration")
				config := &util.KubeVirtDeploymentConfig{
					Namespace:                namespace,
					SynchronizationPlacement: setupPlacement(),
				}

				By("Creating synchronization controller deployment")
				deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
				Expect(deployment).ToNot(BeNil())

				By("Verifying placement is applied")
				verifyPlacement(&deployment.Spec.Template.Spec)
			},
			Entry("with nodeSelector",
				func() *virtv1.ComponentConfig {
					return &virtv1.ComponentConfig{
						NodePlacement: &virtv1.NodePlacement{
							NodeSelector: map[string]string{
								"kubevirt.io/crosscluster-access": "true",
							},
						},
					}
				},
				func(podSpec *corev1.PodSpec) {
					Expect(podSpec.NodeSelector).To(HaveKeyWithValue("kubevirt.io/crosscluster-access", "true"))
				},
			),
			Entry("with tolerations",
				func() *virtv1.ComponentConfig {
					return &virtv1.ComponentConfig{
						NodePlacement: &virtv1.NodePlacement{
							Tolerations: []corev1.Toleration{
								{
									Key:      "dedicated",
									Operator: corev1.TolerationOpEqual,
									Value:    "kubevirt",
									Effect:   corev1.TaintEffectNoSchedule,
								},
							},
						},
					}
				},
				func(podSpec *corev1.PodSpec) {
					Expect(podSpec.Tolerations).To(ContainElement(corev1.Toleration{
						Key:      "dedicated",
						Operator: corev1.TolerationOpEqual,
						Value:    "kubevirt",
						Effect:   corev1.TaintEffectNoSchedule,
					}))
				},
			),
			Entry("with affinity",
				func() *virtv1.ComponentConfig {
					return &virtv1.ComponentConfig{
						NodePlacement: &virtv1.NodePlacement{
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
					}
				},
				func(podSpec *corev1.PodSpec) {
					Expect(podSpec.Affinity).ToNot(BeNil())
					Expect(podSpec.Affinity.NodeAffinity).ToNot(BeNil())
				},
			),
		)

		It("should use default control-plane placement when no custom placement configured", func() {
			By("Creating config without custom placement")
			config := &util.KubeVirtDeploymentConfig{
				Namespace: namespace,
			}

			By("Creating synchronization controller deployment")
			deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
			Expect(deployment).ToNot(BeNil())

			By("Verifying default control-plane placement")
		})

		It("should apply all placement fields when all are configured", func() {
			By("Creating config with complete placement configuration")
			config := &util.KubeVirtDeploymentConfig{
				Namespace: namespace,
				SynchronizationPlacement: &virtv1.ComponentConfig{
					NodePlacement: &virtv1.NodePlacement{
						NodeSelector: map[string]string{
							"kubevirt.io/crosscluster-access": "true",
							"node-type":                       "dedicated",
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
													Values:   []string{"us-west-2a"},
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

			By("Verifying nodeSelector is applied")
			Expect(deployment.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("kubevirt.io/crosscluster-access", "true"))
			Expect(deployment.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("node-type", "dedicated"))

			By("Verifying tolerations are applied")
			Expect(deployment.Spec.Template.Spec.Tolerations).To(ContainElement(corev1.Toleration{
				Key:      "dedicated",
				Operator: corev1.TolerationOpEqual,
				Value:    "kubevirt",
				Effect:   corev1.TaintEffectNoSchedule,
			}))

			By("Verifying affinity is applied")
			Expect(deployment.Spec.Template.Spec.Affinity).ToNot(BeNil())
			Expect(deployment.Spec.Template.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).ToNot(BeNil())
		})

		It("should override defaults completely when custom placement is provided", func() {
			By("Creating config with custom nodeSelector only")
			config := &util.KubeVirtDeploymentConfig{
				Namespace: namespace,
				SynchronizationPlacement: &virtv1.ComponentConfig{
					NodePlacement: &virtv1.NodePlacement{
						NodeSelector: map[string]string{
							"custom-node": "true",
						},
					},
				},
			}

			By("Creating synchronization controller deployment")
			deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
			Expect(deployment).ToNot(BeNil())

			By("Verifying custom nodeSelector is applied")
			Expect(deployment.Spec.Template.Spec.NodeSelector).To(HaveKeyWithValue("custom-node", "true"))

			By("Verifying default control-plane nodeSelector is NOT present")
			Expect(deployment.Spec.Template.Spec.NodeSelector).ToNot(HaveKey("node-role.kubernetes.io/control-plane"))

			By("Verifying default control-plane toleration is NOT present")
			hasControlPlaneToleration := false
			for _, tol := range deployment.Spec.Template.Spec.Tolerations {
				if tol.Key == "node-role.kubernetes.io/control-plane" {
					hasControlPlaneToleration = true
					break
				}
			}
			Expect(hasControlPlaneToleration).To(BeFalse(), "Default control-plane toleration should not be present when custom placement is configured")
		})

		It("should handle nil NodePlacement gracefully", func() {
			By("Creating config with ComponentConfig but nil NodePlacement")
			config := &util.KubeVirtDeploymentConfig{
				Namespace: namespace,
				SynchronizationPlacement: &virtv1.ComponentConfig{
					NodePlacement: nil,
				},
			}

			By("Creating synchronization controller deployment")
			deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
			Expect(deployment).ToNot(BeNil())
		})

		It("should handle empty NodePlacement fields gracefully", func() {
			By("Creating config with NodePlacement but all fields nil")
			config := &util.KubeVirtDeploymentConfig{
				Namespace: namespace,
				SynchronizationPlacement: &virtv1.ComponentConfig{
					NodePlacement: &virtv1.NodePlacement{
						NodeSelector: nil,
						Tolerations:  nil,
						Affinity:     nil,
					},
				},
			}

			By("Creating synchronization controller deployment")
			deployment := NewSynchronizationControllerDeployment(config, "kubevirt", "v1.0.0", "sync-controller")
			Expect(deployment).ToNot(BeNil())

			By("Verifying nodeSelector is not applied (nil field)")
			Expect(deployment.Spec.Template.Spec.NodeSelector).To(BeNil())

			By("Verifying base deployment's defaults remain when NodePlacement fields are nil")
			// CriticalAddonsOnly toleration from base deployment
			Expect(deployment.Spec.Template.Spec.Tolerations).To(ContainElement(corev1.Toleration{
				Key:      "CriticalAddonsOnly",
				Operator: corev1.TolerationOpExists,
			}))
			// PodAntiAffinity from base deployment
			Expect(deployment.Spec.Template.Spec.Affinity).ToNot(BeNil())
			Expect(deployment.Spec.Template.Spec.Affinity.PodAntiAffinity).ToNot(BeNil())

			By("Verifying default control-plane placement is NOT applied")
			Expect(deployment.Spec.Template.Spec.NodeSelector).ToNot(HaveKey("node-role.kubernetes.io/control-plane"))
			hasControlPlaneToleration := false
			for _, tol := range deployment.Spec.Template.Spec.Tolerations {
				if tol.Key == "node-role.kubernetes.io/control-plane" {
					hasControlPlaneToleration = true
					break
				}
			}
			Expect(hasControlPlaneToleration).To(BeFalse(), "Control-plane toleration should not be present when custom NodePlacement is configured, even if fields are nil")
		})
	})
})
