package v1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/pkg/api/v1"
	"testing"
)

var _ = Describe("PodSelectors", func() {
	Context("Pod affinity rules", func() {
		var (
			pod             = &v1.Pod{}
			selector        = &v1.NodeSelectorTerm{}
			podWithSelector = &v1.Pod{}
		)

		BeforeEach(func() {
			pod = &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
			}
			selector = &v1.NodeSelectorTerm{
				MatchExpressions: []v1.NodeSelectorRequirement{
					v1.NodeSelectorRequirement{
						Key:      "kubernetes.io/hostname",
						Operator: v1.NodeSelectorOpNotIn,
						Values:   []string{pod.Spec.NodeName},
					},
				},
			}
			podWithSelector = &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "testnode",
					Affinity: &v1.Affinity{
						NodeAffinity: &v1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
								NodeSelectorTerms: []v1.NodeSelectorTerm{
									v1.NodeSelectorTerm{
										MatchExpressions: []v1.NodeSelectorRequirement{
											v1.NodeSelectorRequirement{
												Key:      "kubernetes.io/hostname",
												Operator: v1.NodeSelectorOpIn,
												Values:   []string{"test2"},
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})

		AfterEach(func() {
		})

		It("should work", func() {
			vm := NewMinimalVM("testvm")
			vm.Status.NodeName = "test-node"
			newSelector := AntiAffinityFromVMNode(vm)
			Expect(newSelector).ToNot(BeNil())
			Expect(len(newSelector.MatchExpressions)).To(Equal(1))
			Expect(len(newSelector.MatchExpressions[0].Values)).To(Equal(1))
			Expect(newSelector.MatchExpressions[0].Values[0]).To(Equal("test-node"))
		})
		It("Should create missing structs", func() {
			newPod, err := ApplyAntiAffinityToPod(pod, selector)
			Expect(err).ToNot(HaveOccurred())
			Expect(newPod.Spec.Affinity).ToNot(BeNil())
			Expect(newPod.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(newPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution).ToNot(BeNil())
			terms := newPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			Expect(len(terms)).To(Equal(1))
		})
		It("Should append to existing node selectors", func() {
			newPod, err := ApplyAntiAffinityToPod(podWithSelector, selector)
			Expect(err).ToNot(HaveOccurred())
			terms := newPod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			Expect(len(terms)).To(Equal(2))
		})
	})
})

func TestSelectors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PodSelectors")
}
