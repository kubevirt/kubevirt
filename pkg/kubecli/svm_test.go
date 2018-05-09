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

var _ = Describe("Kubevirt StatefulVirtualMachine Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha1/namespaces/default/statefulvirtualmachines"
	vmPath := basePath + "/testvm"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VM", func() {
		svm := NewMinimalSVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, svm),
		))
		fetchedVM, err := client.StatefulVirtualMachine(k8sv1.NamespaceDefault).Get("testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVM).To(Equal(svm))
	})

	It("should detect non existent StatefulVirtualMachines", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvm")),
		))
		_, err := client.StatefulVirtualMachine(k8sv1.NamespaceDefault).Get("testvm", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a StatefulVirtualMachine list", func() {
		svm := NewMinimalSVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewSVMList(*svm)),
		))
		fetchedVMList, err := client.StatefulVirtualMachine(k8sv1.NamespaceDefault).List(&k8smetav1.ListOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMList.Items).To(HaveLen(1))
		Expect(fetchedVMList.Items[0]).To(Equal(*svm))
	})

	It("should create a StatefulVirtualMachine", func() {
		svm := NewMinimalSVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, svm),
		))
		createdVM, err := client.StatefulVirtualMachine(k8sv1.NamespaceDefault).Create(svm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVM).To(Equal(svm))
	})

	It("should update a StatefulVirtualMachine", func() {
		svm := NewMinimalSVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", vmPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, svm),
		))
		updatedVM, err := client.StatefulVirtualMachine(k8sv1.NamespaceDefault).Update(svm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVM).To(Equal(svm))
	})

	It("should delete a VM", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", vmPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.StatefulVirtualMachine(k8sv1.NamespaceDefault).Delete("testvm", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})

func NewMinimalSVM(name string) *v1.StatefulVirtualMachine {
	return &v1.StatefulVirtualMachine{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "StatefulVirtualMachine"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}

func NewSVMList(svms ...v1.StatefulVirtualMachine) *v1.StatefulVirtualMachineList {
	return &v1.StatefulVirtualMachineList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "StatefulVirtualMachineList"}, Items: svms}
}
