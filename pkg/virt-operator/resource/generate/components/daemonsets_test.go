package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
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

	hostPathFor := func(ds *appsv1.DaemonSet, name string) string {
		for i := range ds.Spec.Template.Spec.Volumes {
			vol := ds.Spec.Template.Spec.Volumes[i]
			if vol.Name == name {
				Expect(vol.HostPath).NotTo(BeNil(), "volume %q should be a hostPath volume", name)
				return vol.HostPath.Path
			}
		}
		Fail("volume " + name + " not found")
		return ""
	}

	mountPathFor := func(ds *appsv1.DaemonSet, name string) string {
		container := ds.Spec.Template.Spec.Containers[0]
		for i := range container.VolumeMounts {
			if container.VolumeMounts[i].Name == name {
				return container.VolumeMounts[i].MountPath
			}
		}
		Fail("volume mount " + name + " not found")
		return ""
	}

	It("should use the default kubelet root for host paths when unset", func() {
		ds := NewHandlerDaemonSet(config, "", "", "")

		Expect(hostPathFor(ds, "kubelet")).To(Equal("/var/lib/kubelet"))
		Expect(hostPathFor(ds, "kubelet-pods")).To(Equal("/var/lib/kubelet/pods"))
	})

	It("should thread a custom kubelet root into the host paths", func() {
		config.KubeletRootDir = "/var/lib/k0s/kubelet"
		ds := NewHandlerDaemonSet(config, "", "", "")

		Expect(hostPathFor(ds, "kubelet")).To(Equal("/var/lib/k0s/kubelet"))
		Expect(hostPathFor(ds, "kubelet-pods")).To(Equal("/var/lib/k0s/kubelet/pods"))
	})

	It("should keep the in-container kubelet mount target stable regardless of the host root", func() {
		config.KubeletRootDir = "/var/lib/k0s/kubelet"
		ds := NewHandlerDaemonSet(config, "", "", "")

		// The in-container mount targets must stay constant so that in-container
		// lookups (e.g. cpu_manager_state under /var/lib/kubelet) keep working.
		Expect(mountPathFor(ds, "kubelet")).To(Equal("/var/lib/kubelet"))
		Expect(mountPathFor(ds, "kubelet-pods")).To(Equal("/pods"))
	})
})
