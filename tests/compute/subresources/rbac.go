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
 * Copyright The KubeVirt Authors
 *
 */

package compute

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = compute.SIGDescribe("[rfe_id:1195][crit:medium][vendor:cnv-qe@redhat.com][level:component] Rbac authorization", func() {
	var saClient kubecli.KubevirtClient

	When("correct permissions are provided", func() {
		BeforeEach(func() {
			saClient = getClientForSA(kubevirt.Client(), testsuite.SubresourceServiceAccountName)
		})

		It("[test_id:3170]should allow access to vm subresource endpoint", func() {
			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
			vm, err := kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = saClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:3172]should allow access to version subresource endpoint", func() {
			_, err := saClient.ServerVersion().Get()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow access to guestfs subresource endpoint", func() {
			_, err := saClient.GuestfsVersion().Get()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow access to expand-vm-spec subresource", func() {
			_, err := saClient.ExpandSpec(testsuite.GetTestNamespace(nil)).ForVirtualMachine(libvmi.NewVirtualMachine(libvmifact.NewGuestless()))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("correct permissions are not provided", func() {
		BeforeEach(func() {
			saClient = getClientForSA(kubevirt.Client(), testsuite.SubresourceUnprivilegedServiceAccountName)
		})

		It("[test_id:3171]should block access to vm subresource endpoint", func() {
			vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
			vm, err := kubevirt.Client().VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			err = saClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).To(HaveOccurred())
			Expect(errors.ReasonForError(err)).To(Equal(metav1.StatusReasonForbidden))
		})

		It("[test_id:3173]should allow access to version subresource endpoint", func() {
			_, err := saClient.ServerVersion().Get()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should allow access to guestfs subresource endpoint", func() {
			_, err := saClient.GuestfsVersion().Get()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should block access to expand-vm-spec subresource", func() {
			_, err := saClient.ExpandSpec(testsuite.GetTestNamespace(nil)).ForVirtualMachine(libvmi.NewVirtualMachine(libvmifact.NewGuestless()))
			Expect(err).To(HaveOccurred())
			Expect(errors.ReasonForError(err)).To(Equal(metav1.StatusReasonForbidden))
		})
	})
})

func getClientForSA(virtCli kubecli.KubevirtClient, saName string) kubecli.KubevirtClient {
	secret, err := virtCli.CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Get(context.Background(), saName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	token, ok := secret.Data["token"]
	Expect(ok).To(BeTrue())

	saClient, err := kubecli.GetKubevirtClientFromRESTConfig(&rest.Config{
		Host: virtCli.Config().Host,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
		BearerToken: string(token),
	})
	Expect(err).ToNot(HaveOccurred())

	return saClient
}
