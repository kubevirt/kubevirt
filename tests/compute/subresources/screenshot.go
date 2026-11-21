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

package subresources

import (
	"bytes"
	"context"
	"image/png"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(compute.SIG("Screenshot subresource RBAC", decorators.WgS390x, decorators.WgArm64, func() {
	It("allows a screenshot-only serviceaccount to capture a screenshot and denies interactive VNC", func() {
		vmi := libvmops.RunVMIAndExpectLaunch(
			libvmifact.NewGuestless(libvmi.WithAutoattachGraphicsDevice(true)),
			flags.StartupTimeoutSecondsMedium(),
		)
		saClient := screenshotOnlyClient(vmi.Namespace)

		By("denying interactive VNC")
		_, err := subresourceGet(saClient, vmi.Namespace, vmi.Name, "vnc")
		Expect(err).To(HaveOccurred())
		Expect(errors.IsForbidden(err)).To(BeTrue())

		By("denying the legacy vnc/screenshot subresource, which kube-apiserver authorizes as vnc")
		_, err = subresourceGet(saClient, vmi.Namespace, vmi.Name, "vnc", "screenshot")
		Expect(err).To(HaveOccurred())
		Expect(errors.IsForbidden(err)).To(BeTrue())

		By("allowing the screenshot subresource")
		Eventually(func() error {
			raw, err := subresourceGet(saClient, vmi.Namespace, vmi.Name, "screenshot")
			if err != nil {
				return err
			}
			_, err = png.Decode(bytes.NewReader(raw))
			return err
		}, 60*time.Second).Should(Succeed())
	})
}))

func subresourceGet(client kubecli.KubevirtClient, namespace, name string, subresource ...string) ([]byte, error) {
	return client.GeneratedKubeVirtClient().KubevirtV1().RESTClient().Get().
		AbsPath("/apis/subresources.kubevirt.io/" + v1.ApiStorageVersion).
		Namespace(namespace).
		Resource("virtualmachineinstances").
		Name(name).
		SubResource(subresource...).
		Do(context.Background()).
		Raw()
}

func screenshotOnlyClient(namespace string) kubecli.KubevirtClient {
	const saName = "screenshot-only-sa"
	virtCli := kubevirt.Client()

	sa := &k8sv1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: saName}}
	_, err := virtCli.CoreV1().ServiceAccounts(namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(func() {
		deleteScreenshotOnlyObject(virtCli.CoreV1().ServiceAccounts(namespace).Delete(context.Background(), saName, metav1.DeleteOptions{}))
	})

	secret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: saName,
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": saName,
			},
		},
		Type: k8sv1.SecretTypeServiceAccountToken,
	}
	_, err = virtCli.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(func() {
		deleteScreenshotOnlyObject(virtCli.CoreV1().Secrets(namespace).Delete(context.Background(), saName, metav1.DeleteOptions{}))
	})

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: saName},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{v1.SubresourceGroupName},
			Resources: []string{"virtualmachineinstances/screenshot"},
			Verbs:     []string{"get"},
		}},
	}
	_, err = virtCli.RbacV1().Roles(namespace).Create(context.Background(), role, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(func() {
		deleteScreenshotOnlyObject(virtCli.RbacV1().Roles(namespace).Delete(context.Background(), saName, metav1.DeleteOptions{}))
	})

	binding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: saName},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     saName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      saName,
			Namespace: namespace,
		}},
	}
	_, err = virtCli.RbacV1().RoleBindings(namespace).Create(context.Background(), binding, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(func() {
		deleteScreenshotOnlyObject(virtCli.RbacV1().RoleBindings(namespace).Delete(context.Background(), saName, metav1.DeleteOptions{}))
	})

	var token string
	Eventually(func(g Gomega) {
		created, err := virtCli.CoreV1().Secrets(namespace).Get(context.Background(), saName, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		rawToken, ok := created.Data["token"]
		g.Expect(ok).To(BeTrue())
		token = string(rawToken)
		g.Expect(token).ToNot(BeEmpty())
	}, 30*time.Second, time.Second).Should(Succeed())

	saClient, err := kubecli.GetKubevirtClientFromRESTConfig(&rest.Config{
		Host: virtCli.Config().Host,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
		BearerToken: token,
	})
	Expect(err).ToNot(HaveOccurred())
	return saClient
}

func deleteScreenshotOnlyObject(err error) {
	if errors.IsNotFound(err) {
		return
	}
	Expect(err).ToNot(HaveOccurred())
}
