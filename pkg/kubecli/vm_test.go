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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package kubecli

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Kubevirt VirtualMachineInstance Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances"
	vmiPath := basePath + "/testvmi"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineInstance", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
		))
		fetchedVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Get("testvmi", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMI).To(Equal(vmi))
	})

	It("should detect non existent VMIs", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvmi")),
		))
		_, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Get("testvmi", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a VirtualMachineInstance list", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVMIList(*vmi)),
		))
		fetchedVMIList, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIList.Items).To(HaveLen(1))
		Expect(fetchedVMIList.Items[0]).To(Equal(*vmi))
	})

	It("should create a VirtualMachineInstance", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, vmi),
		))
		createdVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Create(vmi)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMI).To(Equal(vmi))
	})

	It("should update a VirtualMachineInstance", func() {
		vmi := v1.NewMinimalVMI("testvmi")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vmi),
		))
		updatedVMI, err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Update(vmi)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMI).To(Equal(vmi))
	})

	It("should delete a VirtualMachineInstance", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", vmiPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachineInstance(k8sv1.NamespaceDefault).Delete("testvmi", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})

func NewVMIList(vmis ...v1.VirtualMachineInstance) *v1.VirtualMachineInstanceList {
	return &v1.VirtualMachineInstanceList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceList"}, Items: vmis}
}
