package v1beta1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

var _ = Describe("HyperconvergedResources", func() {
	Describe("hcoConfig2CnaoPlacement", func() {
		seconds1, seconds2 := int64(1), int64(2)
		tolr1 := corev1.Toleration{
			Key: "key1", Operator: "operator1", Value: "value1", Effect: "effect1", TolerationSeconds: &seconds1,
		}
		tolr2 := corev1.Toleration{
			Key: "key2", Operator: "operator2", Value: "value2", Effect: "effect2", TolerationSeconds: &seconds2,
		}

		It("Should return nil if HCO's input is empty", func() {
			Expect(hcoConfig2CnaoPlacement(&sdkapi.NodePlacement{})).To(BeNil())
		})

		It("Should return only NodeSelector", func() {
			hcoConf := &sdkapi.NodePlacement{
				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			}
			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).ToNot(BeNil())
			Expect(cnaoPlacement.Tolerations).To(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAntiAffinity).To(BeNil())

			Expect(cnaoPlacement.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(cnaoPlacement.NodeSelector["key2"]).Should(Equal("value2"))
		})

		It("Should return only Tolerations", func() {
			hcoConf := &sdkapi.NodePlacement{
				Tolerations: []corev1.Toleration{tolr1, tolr2},
			}
			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).To(BeNil())
			Expect(cnaoPlacement.Tolerations).ToNot(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAntiAffinity).To(BeNil())

			Expect(cnaoPlacement.Tolerations[0]).Should(Equal(tolr1))
			Expect(cnaoPlacement.Tolerations[1]).Should(Equal(tolr2))
		})

		It("Should return only Affinity", func() {
			affinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
									{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
								},
								MatchFields: []corev1.NodeSelectorRequirement{
									{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
									{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
								},
							},
						},
					},
				},
			}
			hcoConf := &sdkapi.NodePlacement{
				Affinity: affinity,
			}
			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).To(BeNil())
			Expect(cnaoPlacement.Tolerations).To(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(cnaoPlacement.Affinity.PodAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAntiAffinity).To(BeNil())

			Expect(cnaoPlacement.Affinity.NodeAffinity).Should(Equal(affinity.NodeAffinity))
		})

		It("Should return the whole object", func() {
			hcoConf := &sdkapi.NodePlacement{

				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
										{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
									},
									MatchFields: []corev1.NodeSelectorRequirement{
										{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
										{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
									},
								},
							},
						},
					},
				},
				Tolerations: []corev1.Toleration{tolr1, tolr2},
			}

			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).ToNot(BeNil())
			Expect(cnaoPlacement.Tolerations).ToNot(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(cnaoPlacement.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(cnaoPlacement.Tolerations[0]).Should(Equal(tolr1))
			Expect(cnaoPlacement.Tolerations[1]).Should(Equal(tolr2))

			Expect(cnaoPlacement.Affinity.NodeAffinity).Should(Equal(hcoConf.Affinity.NodeAffinity))
		})
	})

	Describe("hcoConfig2CdiPlacement", func() {
		seconds1, seconds2 := int64(1), int64(2)
		tolr1 := corev1.Toleration{
			Key: "key1", Operator: "operator1", Value: "value1", Effect: "effect1", TolerationSeconds: &seconds1,
		}
		tolr2 := corev1.Toleration{
			Key: "key2", Operator: "operator2", Value: "value2", Effect: "effect2", TolerationSeconds: &seconds2,
		}

		It("Should be empty if HCO's input is empty", func() {
			np := cdiv1beta1.NodePlacement{}
			hcoConfig2CdiPlacement(&sdkapi.NodePlacement{}, &np)

			Expect(np.Affinity).To(BeNil())
			Expect(np.Tolerations).To(BeEmpty())
			Expect(np.NodeSelector).To(BeEmpty())
		})

		It("Should return only NodeSelector", func() {
			hcoConf := &sdkapi.NodePlacement{
				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			}
			np := cdiv1beta1.NodePlacement{}
			hcoConfig2CdiPlacement(hcoConf, &np)

			Expect(np.NodeSelector).ToNot(BeNil())
			Expect(np.Tolerations).To(BeNil())
			Expect(np.Affinity).To(BeNil())

			Expect(np.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(np.NodeSelector["key2"]).Should(Equal("value2"))
		})

		It("Should return only Tolerations", func() {
			hcoConf := &sdkapi.NodePlacement{
				Tolerations: []corev1.Toleration{tolr1, tolr2},
			}
			np := cdiv1beta1.NodePlacement{}
			hcoConfig2CdiPlacement(hcoConf, &np)

			Expect(np.NodeSelector).To(BeNil())
			Expect(np.Tolerations).ToNot(BeNil())
			Expect(np.Affinity).To(BeNil())

			Expect(np.Tolerations[0]).Should(Equal(tolr1))
			Expect(np.Tolerations[1]).Should(Equal(tolr2))
		})

		It("Should return only Affinity", func() {
			affinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
									{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
								},
								MatchFields: []corev1.NodeSelectorRequirement{
									{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
									{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
								},
							},
						},
					},
				},
			}
			hcoConf := &sdkapi.NodePlacement{
				Affinity: affinity,
			}

			np := cdiv1beta1.NodePlacement{}
			hcoConfig2CdiPlacement(hcoConf, &np)

			Expect(np.NodeSelector).To(BeNil())
			Expect(np.Tolerations).To(BeNil())
			Expect(np.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(np.Affinity.PodAffinity).To(BeNil())
			Expect(np.Affinity.PodAntiAffinity).To(BeNil())

			Expect(np.Affinity.NodeAffinity).Should(Equal(affinity.NodeAffinity))
		})

		It("Should return the whole object", func() {
			hcoConf := &sdkapi.NodePlacement{

				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
										{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
									},
									MatchFields: []corev1.NodeSelectorRequirement{
										{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
										{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
									},
								},
							},
						},
					},
				},
				Tolerations: []corev1.Toleration{tolr1, tolr2},
			}

			np := cdiv1beta1.NodePlacement{}
			hcoConfig2CdiPlacement(hcoConf, &np)

			Expect(np.NodeSelector).ToNot(BeNil())
			Expect(np.Tolerations).ToNot(BeNil())
			Expect(np.Affinity.NodeAffinity).ToNot(BeNil())

			Expect(np.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(np.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(np.Tolerations[0]).Should(Equal(tolr1))
			Expect(np.Tolerations[1]).Should(Equal(tolr2))

			Expect(np.Affinity.NodeAffinity).Should(Equal(hcoConf.Affinity.NodeAffinity))
		})
	})
})
