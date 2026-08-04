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

	It("should mount the host kubelet root at the same path in the container", func() {
		customRoot := "/var/lib/rancher/k3s/agent/kubelet"
		config.AdditionalProperties = map[string]string{operatorutil.AdditionalPropertiesKubeletRootDir: customRoot}

		ds := NewHandlerDaemonSet(config, "", "", "")
		container := ds.Spec.Template.Spec.Containers[0]

		var kubeletMount *corev1.VolumeMount
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == "kubelet" {
				kubeletMount = &container.VolumeMounts[i]
				break
			}
		}
		Expect(kubeletMount).NotTo(BeNil())
		Expect(kubeletMount.MountPath).To(Equal(customRoot))

		var kubeletVol *corev1.Volume
		for i := range ds.Spec.Template.Spec.Volumes {
			if ds.Spec.Template.Spec.Volumes[i].Name == "kubelet" {
				kubeletVol = &ds.Spec.Template.Spec.Volumes[i]
				break
			}
		}
		Expect(kubeletVol).NotTo(BeNil())
		Expect(kubeletVol.HostPath).NotTo(BeNil())
		Expect(kubeletVol.HostPath.Path).To(Equal(customRoot))
	})

	It("should add a device-plugins volume at the default path when kubelet root is not the default", func() {
		// Kubernetes device-plugin sockets are always at /var/lib/kubelet/device-plugins/
		// regardless of --root-dir (kubernetes/kubernetes#120626). When kubeletRootDir differs
		// from the default, virt-handler's device plugins need a separate mount to reach the
		// kubelet socket for device registration.
		customRoot := "/var/lib/rancher/k3s/agent/kubelet"
		config.AdditionalProperties = map[string]string{operatorutil.AdditionalPropertiesKubeletRootDir: customRoot}

		ds := NewHandlerDaemonSet(config, "", "", "")
		container := ds.Spec.Template.Spec.Containers[0]

		var dpMount *corev1.VolumeMount
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == "kubelet-device-plugins" {
				dpMount = &container.VolumeMounts[i]
				break
			}
		}
		Expect(dpMount).NotTo(BeNil(), "kubelet-device-plugins mount must exist for non-default kubelet root")
		Expect(dpMount.MountPath).To(Equal("/var/lib/kubelet/device-plugins"))

		var dpVol *corev1.Volume
		for i := range ds.Spec.Template.Spec.Volumes {
			if ds.Spec.Template.Spec.Volumes[i].Name == "kubelet-device-plugins" {
				dpVol = &ds.Spec.Template.Spec.Volumes[i]
				break
			}
		}
		Expect(dpVol).NotTo(BeNil())
		Expect(dpVol.HostPath).NotTo(BeNil())
		Expect(dpVol.HostPath.Path).To(Equal("/var/lib/kubelet/device-plugins"))
	})
})
