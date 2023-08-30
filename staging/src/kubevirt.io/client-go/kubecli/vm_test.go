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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package kubecli

import (
	"context"
	"fmt"
	"net/http"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Kubevirt VirtualMachine Client", func() {

	var server *ghttp.Server
	basePath := "/apis/kubevirt.io/v1/namespaces/default/virtualmachines"
	vmPath := path.Join(basePath, "testvm")
	subBasePath := fmt.Sprintf("/apis/subresources.kubevirt.io/%s/namespaces/default/virtualmachines", v1.SubresourceStorageGroupVersion.Version)
	subVMPath := path.Join(subBasePath, "testvm")
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	DescribeTable("should fetch a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vmPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))
		fetchedVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Get(context.Background(), "testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVM).To(Equal(vm))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent VirtualMachines", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, vmPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvm")),
		))
		_, err = client.VirtualMachine(k8sv1.NamespaceDefault).Get(context.Background(), "testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a VirtualMachine list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVMList(*vm)),
		))
		fetchedVMList, err := client.VirtualMachine(k8sv1.NamespaceDefault).List(context.Background(), &k8smetav1.ListOptions{})
		apiVersion, kind := v1.VirtualMachineGroupVersionKind.ToAPIVersionAndKind()

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMList.Items).To(HaveLen(1))
		Expect(fetchedVMList.Items[0].APIVersion).To(Equal(apiVersion))
		Expect(fetchedVMList.Items[0].Kind).To(Equal(kind))
		Expect(fetchedVMList.Items[0]).To(Equal(*vm))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a VirtualMachine", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, vm),
		))
		createdVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Create(context.Background(), vm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVM).To(Equal(vm))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a VirtualMachine", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, vmPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))
		updatedVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Update(context.Background(), vm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVM).To(Equal(vm))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should patch a VirtualMachine", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vm := NewMinimalVM("testvm")
		running := true
		vm.Spec.Running = &running

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PATCH", path.Join(proxyPath, vmPath)),
			ghttp.VerifyBody([]byte("{\"spec\":{\"running\":true}}")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))

		patchedVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Patch(context.Background(), vm.Name, types.MergePatchType, []byte("{\"spec\":{\"running\":true}}"), &k8smetav1.PatchOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.Running).To(Equal(patchedVM.Spec.Running))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fail on patch a VirtualMachine", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		vm := NewMinimalVM("testvm")

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PATCH", path.Join(proxyPath, vmPath)),
			ghttp.VerifyBody([]byte("{\"spec\":{\"running\":true}}")),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, vm),
		))

		patchedVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Patch(context.Background(), vm.Name, types.MergePatchType, []byte("{\"spec\":{\"running\":true}}"), &k8smetav1.PatchOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(vm.Spec.Running).To(Equal(patchedVM.Spec.Running))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a VirtualMachineInstance", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, vmPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachine(k8sv1.NamespaceDefault).Delete(context.Background(), "testvm", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should restart a VirtualMachine", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMPath, "restart")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachine(k8sv1.NamespaceDefault).Restart(context.Background(), "testvm", &v1.RestartOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should migrate a VirtualMachine", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, subVMPath, "migrate")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachine(k8sv1.NamespaceDefault).Migrate(context.Background(), "testvm", &virtv1.MigrateOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	AfterEach(func() {
		server.Close()
	})
})
