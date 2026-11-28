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

package controlplane_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	vap "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components/validatingadmissionpolicies"
	kvinstall "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

const usbResource = "devices.kubevirt.io/usb"

// The host-device policy only needs a real API server, the KubeVirt parameter
// object, and the policy binding. virt-operator and a device plugin are not
// involved. The policy and binding are the objects
// GenerateCurrentInstallStrategy produces, which is what virt-operator
// reconciles. Cluster coverage that virt-operator installs them stays in
// tests/validatingadmissionpolicy.
var _ = Describe("permitted host device admission", Ordered, func() {
	var (
		f                *framework.Framework
		ns               string
		controllerClient kubernetes.Interface
	)

	BeforeAll(func() {
		f = framework.New()
		// Start registers DeferCleanup, so this envtest stops with the container.
		f.Start(context.Background())
	})

	BeforeAll(func(ctx SpecContext) {
		ns = f.CreateNamespace(ctx)
		controllerUser := components.ServiceAccountUsername(ns, components.ControllerServiceAccountName)

		kv := &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: ns},
			Spec: v1.KubeVirtSpec{
				ImageRegistry: "fake-registry",
				ImageTag:      "v9.9.9",
				Configuration: v1.KubeVirtConfiguration{
					PermittedHostDevices: &v1.PermittedHostDevices{
						USB: []v1.USBHostDevice{{
							ResourceName: usbResource,
						}},
					},
				},
			},
		}
		_, err := f.VirtClient().GeneratedKubeVirtClient().KubevirtV1().KubeVirts(ns).Create(ctx, kv, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		_, err = f.K8sClient().RbacV1().Roles(ns).Create(ctx, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "host-device-pod-creator"},
			Rules: []rbacv1.PolicyRule{{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"create", "get", "delete"},
			}},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		_, err = f.K8sClient().RbacV1().RoleBindings(ns).Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "virt-controller-pods"},
			Subjects: []rbacv1.Subject{{
				Kind:     rbacv1.UserKind,
				Name:     controllerUser,
				APIGroup: rbacv1.GroupName,
			}},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     "host-device-pod-creator",
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		installHostDeviceAdmission(ctx, f, kv)

		By("waiting until the policy denies a caller other than virt-controller")
		Eventually(ctx, func(g Gomega) {
			requireHostDeviceDenied(ctx, g, f.K8sClient(), ns)
		}).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(Succeed())

		controllerClient = kubeClientAs(f.VirtClient().Config(), controllerUser)
	})

	It("should admit a pod that does not request a managed host device", func(ctx SpecContext) {
		pod, err := createPod(ctx, f.K8sClient(), ns, "no-host-device-", "")
		Expect(err).NotTo(HaveOccurred())
		Expect(deletePod(ctx, f.K8sClient(), ns, pod.Name)).To(Succeed())
	})

	It("should reject a pod that requests a managed host device", func(ctx SpecContext) {
		requireHostDeviceDenied(ctx, Default, f.K8sClient(), ns)
	})

	It("should admit a pod that requests a managed host device when the caller is virt-controller", func(ctx SpecContext) {
		pod, err := createPod(ctx, controllerClient, ns, "controller-host-device-", usbResource)
		Expect(err).NotTo(HaveOccurred())
		Expect(deletePod(ctx, controllerClient, ns, pod.Name)).To(Succeed())
	})
})

func installHostDeviceAdmission(ctx context.Context, f *framework.Framework, kv *v1.KubeVirt) {
	GinkgoHelper()

	config := operatorutil.GetTargetConfigFromKV(kv)
	strategy, err := kvinstall.GenerateCurrentInstallStrategy(config, "", kv.Namespace)
	Expect(err).NotTo(HaveOccurred())

	policy := hostDevicePolicy(strategy)
	_, err = f.VirtClient().AdmissionregistrationV1().ValidatingAdmissionPolicies().Create(
		ctx, policy.DeepCopy(), metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func(ctx SpecContext) {
		deleteErr := f.VirtClient().AdmissionregistrationV1().ValidatingAdmissionPolicies().Delete(
			ctx, vap.HostDevicesRestrictionPolicyName, metav1.DeleteOptions{})
		if deleteErr != nil && !errors.IsNotFound(deleteErr) {
			Expect(deleteErr).NotTo(HaveOccurred())
		}
	})

	binding := hostDeviceBinding(strategy)
	_, err = f.VirtClient().AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Create(
		ctx, binding.DeepCopy(), metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func(ctx SpecContext) {
		deleteErr := f.VirtClient().AdmissionregistrationV1().ValidatingAdmissionPolicyBindings().Delete(
			ctx, vap.HostDevicesRestrictionBindingName, metav1.DeleteOptions{})
		if deleteErr != nil && !errors.IsNotFound(deleteErr) {
			Expect(deleteErr).NotTo(HaveOccurred())
		}
	})
}

func hostDevicePolicy(strategy *kvinstall.Strategy) *admissionregistrationv1.ValidatingAdmissionPolicy {
	GinkgoHelper()
	for _, policy := range strategy.ValidatingAdmissionPolicies() {
		if policy.Name == vap.HostDevicesRestrictionPolicyName {
			return policy
		}
	}
	Fail("install strategy did not contain " + vap.HostDevicesRestrictionPolicyName)
	return nil
}

func hostDeviceBinding(strategy *kvinstall.Strategy) *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	GinkgoHelper()
	for _, binding := range strategy.ValidatingAdmissionPolicyBindings() {
		if binding.Name == vap.HostDevicesRestrictionBindingName {
			return binding
		}
	}
	Fail("install strategy did not contain " + vap.HostDevicesRestrictionBindingName)
	return nil
}

func kubeClientAs(cfg *rest.Config, username string) kubernetes.Interface {
	GinkgoHelper()
	copied := rest.CopyConfig(cfg)
	copied.Impersonate = rest.ImpersonationConfig{UserName: username}
	client, err := kubernetes.NewForConfig(copied)
	Expect(err).NotTo(HaveOccurred())
	return client
}

func createPod(ctx context.Context, client kubernetes.Interface, ns, generateName, device string) (*corev1.Pod, error) {
	GinkgoHelper()
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{GenerateName: generateName},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:  "compute",
				Image: "registry.k8s.io/pause:3.10",
			}},
		},
	}
	if device != "" {
		qty := resource.MustParse("1")
		pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{corev1.ResourceName(device): qty}
		pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{corev1.ResourceName(device): qty}
	}
	return client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
}

func deletePod(ctx context.Context, client kubernetes.Interface, ns, name string) error {
	GinkgoHelper()
	return client.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

func requireHostDeviceDenied(ctx context.Context, g Gomega, client kubernetes.Interface, ns string) {
	GinkgoHelper()
	pod, createErr := createPod(ctx, client, ns, "host-device-", usbResource)
	if createErr == nil {
		g.Expect(deletePod(ctx, client, ns, pod.Name)).To(Succeed())
	}
	g.Expect(createErr).To(MatchError(ContainSubstring(vap.HostDevicesRestrictionErr)))
	g.Expect(errors.IsForbidden(createErr)).To(BeTrue())
}
