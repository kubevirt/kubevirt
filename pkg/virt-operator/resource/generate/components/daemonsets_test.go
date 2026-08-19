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

	Context("SecurityContext", func() {
		It("should NOT set ReadOnlyRootFilesystem on privileged virt-handler container", func() {
			ds := NewHandlerDaemonSet(config, "", "", "")
			container := ds.Spec.Template.Spec.Containers[0]
			Expect(container.Name).To(Equal(VirtHandlerName))
			Expect(container.SecurityContext).ToNot(BeNil())
			Expect(container.SecurityContext.Privileged).ToNot(BeNil())
			Expect(*container.SecurityContext.Privileged).To(BeTrue())
			Expect(container.SecurityContext.ReadOnlyRootFilesystem).To(BeNil(),
				"privileged containers should not set readOnlyRootFilesystem as it provides no security benefit and breaks runtime writes")
		})

		It("should NOT set ReadOnlyRootFilesystem on privileged virt-launcher init container", func() {
			ds := NewHandlerDaemonSet(config, "", "", "")
			var launcher *corev1.Container
			for i := range ds.Spec.Template.Spec.InitContainers {
				if ds.Spec.Template.Spec.InitContainers[i].Name == VirtLauncherName {
					launcher = &ds.Spec.Template.Spec.InitContainers[i]
					break
				}
			}
			Expect(launcher).ToNot(BeNil(), "virt-launcher init container should exist")
			Expect(launcher.SecurityContext).ToNot(BeNil())
			Expect(launcher.SecurityContext.Privileged).ToNot(BeNil())
			Expect(*launcher.SecurityContext.Privileged).To(BeTrue())
			Expect(launcher.SecurityContext.ReadOnlyRootFilesystem).To(BeNil(),
				"privileged containers should not set readOnlyRootFilesystem as it provides no security benefit and breaks runtime writes")
		})
	})

	DescribeTable("should propagate imagePullPolicy to",
		func(additionalProperties map[string]string, containerName string, isInitContainer bool, expectedPolicy corev1.PullPolicy) {
			config.AdditionalProperties = additionalProperties
			ds := NewHandlerDaemonSet(config, "", "", "")

			containers := ds.Spec.Template.Spec.Containers
			if isInitContainer {
				containers = ds.Spec.Template.Spec.InitContainers
			}

			var target *corev1.Container
			for i := range containers {
				if containers[i].Name == containerName {
					target = &containers[i]
					break
				}
			}
			Expect(target).NotTo(BeNil(), "container %s should exist", containerName)
			Expect(target.ImagePullPolicy).To(Equal(expectedPolicy))
		},
		Entry("the virt-launcher init container when configured",
			map[string]string{
				operatorutil.AdditionalPropertiesNamePullPolicy: string(corev1.PullAlways),
			},
			"virt-launcher", true, corev1.PullAlways,
		),
		Entry("the virt-launcher init container by default",
			map[string]string(nil),
			"virt-launcher", true, corev1.PullIfNotPresent,
		),
		Entry("the virt-launcher-image-holder container when configured",
			map[string]string{
				operatorutil.AdditionalPropertiesNamePullPolicy: string(corev1.PullAlways),
				operatorutil.AdditionalPropertiesPullSecrets:    `[{"name":"test-secret"}]`,
			},
			"virt-launcher-image-holder", false, corev1.PullAlways,
		),
		Entry("the virt-launcher-image-holder container by default",
			map[string]string{
				operatorutil.AdditionalPropertiesPullSecrets: `[{"name":"test-secret"}]`,
			},
			"virt-launcher-image-holder", false, corev1.PullIfNotPresent,
		),
	)

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
