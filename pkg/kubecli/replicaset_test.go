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

var _ = Describe("Kubevirt VirtualMachineReplicaSet Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachinereplicasets"
	rsPath := basePath + "/testrs"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineReplicaSet", func() {
		rs := NewMinimalVMReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", rsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, rs),
		))
		fetchedVMReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Get("testrs", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMReplicaSet).To(Equal(rs))
	})

	It("should detect non existent VMReplicaSets", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", rsPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testrs")),
		))
		_, err := client.ReplicaSet(k8sv1.NamespaceDefault).Get("testrs", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a VirtualMachineReplicaSet list", func() {
		rs := NewMinimalVMReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVMReplicaSetList(*rs)),
		))
		fetchedVMReplicaSetList, err := client.ReplicaSet(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMReplicaSetList.Items).To(HaveLen(1))
		Expect(fetchedVMReplicaSetList.Items[0]).To(Equal(*rs))
	})

	It("should create a VirtualMachineReplicaSet", func() {
		rs := NewMinimalVMReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, rs),
		))
		createdVMReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Create(rs)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMReplicaSet).To(Equal(rs))
	})

	It("should update a VirtualMachineReplicaSet", func() {
		rs := NewMinimalVMReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, rs),
		))
		updatedVMReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Update(rs)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMReplicaSet).To(Equal(rs))
	})

	It("should delete a VirtualMachineReplicaSet", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", rsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.ReplicaSet(k8sv1.NamespaceDefault).Delete("testrs", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})

func NewVMReplicaSetList(rss ...v1.VirtualMachineReplicaSet) *v1.VirtualMachineReplicaSetList {
	return &v1.VirtualMachineReplicaSetList{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineReplicaSetList"}, Items: rss}
}

func NewMinimalVMReplicaSet(name string) *v1.VirtualMachineReplicaSet {
	return &v1.VirtualMachineReplicaSet{TypeMeta: k8smetav1.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineReplicaSet"}, ObjectMeta: k8smetav1.ObjectMeta{Name: name}}
}
