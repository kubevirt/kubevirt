package watch

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Labels", func() {
	newNode := func(labels map[string]string) *k8sv1.Node {
		return &k8sv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
		}
	}

	newShadowNode := func(labels map[string]string) *v1.ShadowNode {
		return &v1.ShadowNode{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
		}
	}

	It("should keep kubernetes labels", func() {
		kubernetesLabel := "kubernetes.io/os"
		node := newNode(map[string]string{
			kubernetesLabel: "linux",
		})
		shadowNode := newShadowNode(nil)

		nodeLabels, shadowNodeLabels := calculateNodeLabels(node, shadowNode)
		Expect(nodeLabels).To(HaveKey(kubernetesLabel), "node labels")
		Expect(shadowNodeLabels).To(HaveKey(kubernetesLabel), "shadow node labels")
	})

	It("should add node labeller  labels", func() {
		cpuFeatureLabel := v1.CPUFeatureLabel + "something"
		node := newNode(nil)
		shadowNode := newShadowNode(map[string]string{
			cpuFeatureLabel: "true",
		})

		nodeLabels, shadowNodeLabels := calculateNodeLabels(node, shadowNode)
		Expect(nodeLabels).NotTo(HaveKey(cpuFeatureLabel), "node labels")
		Expect(shadowNodeLabels).To(HaveKey(cpuFeatureLabel), "shadow node labels")
	})

	It("should update node labeller  labels", func() {
		cpuFeatureLabel := v1.CPUFeatureLabel + "something"
		node := newNode(map[string]string{
			cpuFeatureLabel: "false",
		})
		shadowNode := newShadowNode(map[string]string{
			cpuFeatureLabel: "true",
		})

		nodeLabels, shadowNodeLabels := calculateNodeLabels(node, shadowNode)
		Expect(nodeLabels).To(HaveKeyWithValue(cpuFeatureLabel, "false"), "node labels")
		Expect(shadowNodeLabels).To(HaveKeyWithValue(cpuFeatureLabel, "true"), "shadow node labels")
	})

})
