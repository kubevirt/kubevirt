package v1

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/api/v1"
)

var _ = Describe("PodSelectors", func() {
	Context("Pod affinity rules", func() {
		var (
			pod      = &v1.Pod{}
			affinity = &v1.Affinity{}
		)

		BeforeEach(func() {
			pod = &v1.Pod{
				Spec: v1.PodSpec{
					NodeName: "testnode",
				},
			}
			affinity = &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
						NodeSelectorTerms: []v1.NodeSelectorTerm{
							{
								MatchExpressions: []v1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: v1.NodeSelectorOpNotIn,
										Values:   []string{pod.Spec.NodeName},
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
			affinity := AntiAffinityFromVMNode(vm)
			newSelector := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]
			Expect(newSelector).ToNot(BeNil())
			Expect(len(newSelector.MatchExpressions)).To(Equal(1))
			Expect(len(newSelector.MatchExpressions[0].Values)).To(Equal(1))
			Expect(newSelector.MatchExpressions[0].Values[0]).To(Equal("test-node"))

		})
	})
})

func TestSelectors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PodSelectors")
}
