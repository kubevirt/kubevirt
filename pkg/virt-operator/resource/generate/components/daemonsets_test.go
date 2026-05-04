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
