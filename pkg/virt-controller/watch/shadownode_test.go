package watch

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("ShadowNode controller", func() {
	Context("Labels", func() {
		It("should keep kubernetes labels", func() {
			kubernetesLabel := "kubernetes.io/os"
			nodeLabels := map[string]string{
				kubernetesLabel: "linux",
			}

			newNodeLabels := calculateNodeLabels(nodeLabels, nil)
			Expect(newNodeLabels).To(HaveKey(kubernetesLabel), "shadow node labels")
		})

		It("should add node kubevirt-related labels", func() {
			cpuFeatureLabel := v1.CPUFeatureLabel + "something"
			ShadowNodeLabels := map[string]string{
				cpuFeatureLabel: "true",
			}

			newNodeLabels := calculateNodeLabels(nil, ShadowNodeLabels)
			Expect(newNodeLabels).To(HaveKey(cpuFeatureLabel), "shadow node labels")
		})

		It("should update kubevirt-related labels", func() {
			cpuFeatureLabel := v1.CPUFeatureLabel + "something"
			nodeLabels := map[string]string{
				cpuFeatureLabel: "false",
			}
			ShadowNodeLabels := map[string]string{
				cpuFeatureLabel: "true",
			}

			newNodeLabels := calculateNodeLabels(nodeLabels, ShadowNodeLabels)
			Expect(newNodeLabels).To(HaveKeyWithValue(cpuFeatureLabel, "true"), "shadow node labels")
		})

		It("should not add non kubevirt Label from shadowNode", func() {
			nonKubevirtLabel := "IrrelevantLabel"
			ShadowNodeLabels := map[string]string{
				nonKubevirtLabel: "true",
			}

			newNodeLabels := calculateNodeLabels(nil, ShadowNodeLabels)
			Expect(newNodeLabels).ToNot(HaveKeyWithValue(nonKubevirtLabel, "true"), "shadow node labels")
		})
	})
	Context("Annotations", func() {
		It("should keep kubernetes annotations", func() {
			kubernetesAnnotation := "k8s.ovn.org/node-id"
			nodeAnnotations := map[string]string{
				kubernetesAnnotation: "8",
			}

			newNodeAnnotations := calculateNodeAnnotations(nodeAnnotations, nil)
			Expect(newNodeAnnotations).To(HaveKey(kubernetesAnnotation), "shadow node Annotations")
		})

		It("should add kubevirt-related annotations", func() {
			kubevirtRelatedAnnotation := v1.AppLabel + "/something"
			ShadowNodeAnnotations := map[string]string{
				kubevirtRelatedAnnotation: "true",
			}

			newNodeAnnotations := calculateNodeAnnotations(nil, ShadowNodeAnnotations)
			Expect(newNodeAnnotations).To(HaveKey(kubevirtRelatedAnnotation), "shadow node Annotations")
		})

		It("should update kubevirt-related annotations", func() {
			kubevirtRelatedAnnotation := v1.AppLabel + "/something"
			nodeAnnotations := map[string]string{
				kubevirtRelatedAnnotation: "false",
			}
			ShadowNodeAnnotations := map[string]string{
				kubevirtRelatedAnnotation: "true",
			}

			newNodeAnnotations := calculateNodeAnnotations(nodeAnnotations, ShadowNodeAnnotations)
			Expect(newNodeAnnotations).To(HaveKeyWithValue(kubevirtRelatedAnnotation, "true"), "shadow node Annotations")
		})

		It("should not add non kubevirt from shadowNode", func() {
			nonKubevirtAnnotation := "IrrelevantAnnotation"
			ShadowNodeAnnotations := map[string]string{
				nonKubevirtAnnotation: "true",
			}

			newNodeAnnotations := calculateNodeAnnotations(nil, ShadowNodeAnnotations)
			Expect(newNodeAnnotations).ToNot(HaveKeyWithValue(nonKubevirtAnnotation, "true"), "shadow node Annotations")
		})

		It("should remove an existing heartbeat node annotation", func() {
			heartbeatAnnotation := v1.VirtHandlerHeartbeat
			nodeAnnotations := map[string]string{
				heartbeatAnnotation: "false",
			}

			newNodeAnnotations := calculateNodeAnnotations(nodeAnnotations, nil)
			Expect(newNodeAnnotations).ToNot(HaveKey(heartbeatAnnotation), "shadow node Annotations")
		})

		It("should not sync shadowNode heartbeat annotation to node", func() {
			heartbeatAnnotation := v1.VirtHandlerHeartbeat
			ShadowNodeAnnotations := map[string]string{
				heartbeatAnnotation: "someValue",
			}

			newNodeAnnotations := calculateNodeAnnotations(nil, ShadowNodeAnnotations)
			Expect(newNodeAnnotations).ToNot(HaveKey(heartbeatAnnotation), "shadow node Annotations")
		})
	})
})
