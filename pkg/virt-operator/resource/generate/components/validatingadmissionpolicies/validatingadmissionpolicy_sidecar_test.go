/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package validatingadmissionpolicies_test

import (
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
)

func podToUnstructured(pod *corev1.Pod) map[string]interface{} {
	data, err := json.Marshal(pod)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	var obj map[string]interface{}
	ExpectWithOffset(1, json.Unmarshal(data, &obj)).ToNot(HaveOccurred())
	return obj
}

func admitSidecarSubPath(pod *corev1.Pod) error {
	return evaluatePolicy(vap.NewSidecarSubPathValidatingAdmissionPolicy(), podToUnstructured(pod))
}

func admitPluginSocketPath(plugin *pluginv1alpha1.Plugin) error {
	return evaluatePolicy(vap.NewPluginSocketPathValidatingAdmissionPolicy(), pluginToUnstructured(plugin))
}

func launcherPod(containers ...corev1.Container) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"kubevirt.io": "virt-launcher"},
		},
		Spec: corev1.PodSpec{
			Containers: containers,
		},
	}
}

func pluginVolumeMount(subPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "kubevirt-plugin-sockets",
		MountPath: "/var/run/kubevirt-plugin/test/",
		SubPath:   subPath,
	}
}

var _ = Describe("Sidecar ValidatingAdmissionPolicies", func() {
	Context("SidecarSubPath policy", func() {
		It("should use objectSelector to match virt-launcher pods", func() {
			policy := vap.NewSidecarSubPathValidatingAdmissionPolicy()
			Expect(policy.Spec.MatchConstraints).ToNot(BeNil())
			Expect(policy.Spec.MatchConstraints.ObjectSelector).ToNot(BeNil())
			Expect(policy.Spec.MatchConstraints.ObjectSelector.MatchLabels).To(HaveKeyWithValue(v1.AppLabel, "virt-launcher"))
			Expect(policy.Spec.MatchConditions).To(BeEmpty())
		})

		It("should allow compute container without subPath", func() {
			pod := launcherPod(corev1.Container{
				Name:         "compute",
				VolumeMounts: []corev1.VolumeMount{pluginVolumeMount("")},
			})
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})

		It("should allow sidecar container with subPath", func() {
			pod := launcherPod(
				corev1.Container{Name: "compute"},
				corev1.Container{
					Name:         "plugin-sidecar-test",
					VolumeMounts: []corev1.VolumeMount{pluginVolumeMount("test")},
				},
			)
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})

		It("should reject sidecar container without subPath", func() {
			pod := launcherPod(
				corev1.Container{Name: "compute"},
				corev1.Container{
					Name:         "plugin-sidecar-test",
					VolumeMounts: []corev1.VolumeMount{pluginVolumeMount("")},
				},
			)
			Expect(admitSidecarSubPath(pod)).NotTo(Succeed())
		})

		It("should allow container not mounting plugin volume", func() {
			pod := launcherPod(
				corev1.Container{Name: "compute"},
				corev1.Container{
					Name:         "unrelated-sidecar",
					VolumeMounts: []corev1.VolumeMount{{Name: "other-volume", MountPath: "/mnt"}},
				},
			)
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})

		It("should reject initContainer without subPath", func() {
			pod := launcherPod(corev1.Container{Name: "compute"})
			pod.Spec.InitContainers = []corev1.Container{{
				Name:         "init-sidecar",
				VolumeMounts: []corev1.VolumeMount{pluginVolumeMount("")},
			}}
			Expect(admitSidecarSubPath(pod)).NotTo(Succeed())
		})

		It("should allow initContainer with subPath", func() {
			pod := launcherPod(corev1.Container{Name: "compute"})
			pod.Spec.InitContainers = []corev1.Container{{
				Name:         "init-sidecar",
				VolumeMounts: []corev1.VolumeMount{pluginVolumeMount("my-plugin")},
			}}
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})

		It("should allow containers without volumeMounts", func() {
			pod := launcherPod(
				corev1.Container{Name: "compute"},
				corev1.Container{Name: "sidecar-no-mounts"},
			)
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})

		It("should allow initContainer without volumeMounts", func() {
			pod := launcherPod(corev1.Container{Name: "compute"})
			pod.Spec.InitContainers = []corev1.Container{{
				Name: "volumedisk0",
			}}
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})

		It("should allow ephemeralContainer without volumeMounts", func() {
			pod := launcherPod(corev1.Container{Name: "compute"})
			pod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{
				EphemeralContainerCommon: corev1.EphemeralContainerCommon{
					Name: "debug-no-mounts",
				},
			}}
			Expect(admitSidecarSubPath(pod)).To(Succeed())
		})
	})

	Context("PluginSocketPath policy", func() {
		It("should allow valid socket path", func() {
			plugin := &pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "my-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar: &pluginv1alpha1.SidecarDomainHook{
							SocketPath: "/var/run/kubevirt-plugin/my-plugin/hook.sock",
						},
					}},
				},
			}
			Expect(admitPluginSocketPath(plugin)).To(Succeed())
		})

		It("should reject socket path without .sock suffix", func() {
			plugin := &pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "my-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar: &pluginv1alpha1.SidecarDomainHook{
							SocketPath: "/var/run/kubevirt-plugin/my-plugin/hook",
						},
					}},
				},
			}
			Expect(admitPluginSocketPath(plugin)).NotTo(Succeed())
		})

		It("should reject socket path over 108 characters", func() {
			longPath := "/var/run/kubevirt-plugin/my-plugin/" + strings.Repeat("a", 80) + ".sock"
			plugin := &pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "my-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar: &pluginv1alpha1.SidecarDomainHook{
							SocketPath: longPath,
						},
					}},
				},
			}
			Expect(admitPluginSocketPath(plugin)).NotTo(Succeed())
		})

		It("should reject socket path outside plugin directory", func() {
			plugin := &pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "my-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar: &pluginv1alpha1.SidecarDomainHook{
							SocketPath: "/tmp/evil/hook.sock",
						},
					}},
				},
			}
			Expect(admitPluginSocketPath(plugin)).NotTo(Succeed())
		})

		It("should allow plugin with only CEL hooks (no sidecar)", func() {
			plugin := &pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "test-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						CEL: &pluginv1alpha1.CELDomainHook{Expression: `Domain{}`},
					}},
				},
			}
			Expect(admitPluginSocketPath(plugin)).To(Succeed())
		})

		It("should allow valid nested socket path", func() {
			plugin := &pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "my-plugin"},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar: &pluginv1alpha1.SidecarDomainHook{
							SocketPath: "/var/run/kubevirt-plugin/my-plugin/sub/dir/hook.sock",
						},
					}},
				},
			}
			Expect(admitPluginSocketPath(plugin)).To(Succeed())
		})
	})
})
