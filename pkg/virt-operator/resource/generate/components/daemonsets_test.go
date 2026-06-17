package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

var _ = Describe("Handler DaemonSet", func() {
	var config *operatorutil.KubeVirtDeploymentConfig

	BeforeEach(func() {
		config = &operatorutil.KubeVirtDeploymentConfig{}
	})

	It("should use the configured imagePullPolicy for the virt-launcher init container", func() {
		config.AdditionalProperties = map[string]string{
			operatorutil.AdditionalPropertiesNamePullPolicy: string(corev1.PullAlways),
		}
		ds := NewHandlerDaemonSet(config, "", "", "")
		Expect(ds.Spec.Template.Spec.InitContainers).NotTo(BeEmpty())
		Expect(ds.Spec.Template.Spec.InitContainers[0].Name).To(Equal("virt-launcher"))
		Expect(ds.Spec.Template.Spec.InitContainers[0].ImagePullPolicy).To(Equal(corev1.PullAlways))
	})

	It("should default to IfNotPresent for the virt-launcher init container", func() {
		ds := NewHandlerDaemonSet(config, "", "", "")
		Expect(ds.Spec.Template.Spec.InitContainers[0].ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
	})

	It("should use the configured imagePullPolicy for the virt-launcher-image-holder container", func() {
		config.AdditionalProperties = map[string]string{
			operatorutil.AdditionalPropertiesNamePullPolicy: string(corev1.PullAlways),
			operatorutil.AdditionalPropertiesPullSecrets:    `[{"name":"test-secret"}]`,
		}
		ds := NewHandlerDaemonSet(config, "", "", "")
		var imageHolder *corev1.Container
		for i := range ds.Spec.Template.Spec.Containers {
			if ds.Spec.Template.Spec.Containers[i].Name == "virt-launcher-image-holder" {
				imageHolder = &ds.Spec.Template.Spec.Containers[i]
				break
			}
		}
		Expect(imageHolder).NotTo(BeNil(), "virt-launcher-image-holder container should exist when imagePullSecrets are configured")
		Expect(imageHolder.ImagePullPolicy).To(Equal(corev1.PullAlways))
	})

	It("should not use bidirectional mount propagation for the kubelet volume", func() {
		ds := NewHandlerDaemonSet(config, "", "", "")
		container := ds.Spec.Template.Spec.Containers[0]

		var kubeletMount *corev1.VolumeMount
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == "kubelet" {
				kubeletMount = &container.VolumeMounts[i]
				break
			}
		}
		Expect(kubeletMount).NotTo(BeNil(), "kubelet volume mount should exist")
		Expect(kubeletMount.MountPropagation).NotTo(BeNil())
		Expect(*kubeletMount.MountPropagation).To(Equal(corev1.MountPropagationHostToContainer))
	})
})
