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
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("Kubevirt VirtualMachine Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachines"
	vmiPath := basePath + "/testvm"
	subBasePath := fmt.Sprintf("/apis/subresources.kubevirt.io/%s/namespaces/default/virtualmachines", virtv1.SubresourceStorageGroupVersion.Version)
	subVMIPath := subBasePath + "/testvm"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineInstance", func() {
		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))
		fetchedVMI, err := client.VirtualMachine(k8sv1.NamespaceDefault).Get("testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMI).To(Equal(vm))
	})

	It("should detect non existent VirtualMachines", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvm")),
		))
		_, err := client.VirtualMachine(k8sv1.NamespaceDefault).Get("testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a VirtualMachine list", func() {
		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVMList(*vm)),
		))
		fetchedVMIList, err := client.VirtualMachine(k8sv1.NamespaceDefault).List(&k8smetav1.ListOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIList.Items).To(HaveLen(1))
		Expect(fetchedVMIList.Items[0]).To(Equal(*vm))
	})

	It("should create a VirtualMachine", func() {
		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, vm),
		))
		createdVMI, err := client.VirtualMachine(k8sv1.NamespaceDefault).Create(vm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMI).To(Equal(vm))
	})

	It("should update a VirtualMachine", func() {
		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))
		updatedVMI, err := client.VirtualMachine(k8sv1.NamespaceDefault).Update(vm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI).To(Equal(vm))
	})

	It("should patch a VirtualMachine", func() {
		vm := NewMinimalVM("testvm")
		running := true
		vm.Spec.Running = &running

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PATCH", vmiPath),
			ghttp.VerifyBody([]byte("{\"spec\":{\"running\":true}}")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))

		patchedVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Patch(vm.Name, types.MergePatchType,
			[]byte("{\"spec\":{\"running\":true}}"))

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(vm.Spec.Running).To(Equal(patchedVM.Spec.Running))

	})

	It("should fail on patch a VirtualMachine", func() {
		vm := NewMinimalVM("testvm")

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PATCH", vmiPath),
			ghttp.VerifyBody([]byte("{\"spec\":{\"running\":true}}")),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, vm),
		))

		patchedVM, err := client.VirtualMachine(k8sv1.NamespaceDefault).Patch(vm.Name, types.MergePatchType,
			[]byte("{\"spec\":{\"running\":true}}"))

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(vm.Spec.Running).To(Equal(patchedVM.Spec.Running))

	})

	It("should delete a VirtualMachineInstance", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachine(k8sv1.NamespaceDefault).Delete("testvm", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should restart a VirtualMachine", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", subVMIPath+"/restart"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachine(k8sv1.NamespaceDefault).Restart("testvm")

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should migrate a VirtualMachine", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", subVMIPath+"/migrate"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachine(k8sv1.NamespaceDefault).Migrate("testvm")

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should rename a VM", func() {
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("PUT", subVMIPath+"/rename"),
				ghttp.RespondWith(http.StatusAccepted, nil),
			),
		)

		err := client.VirtualMachine(k8sv1.NamespaceDefault).Rename("testvm", &virtv1.RenameOptions{NewName: "testvmnew"})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})
