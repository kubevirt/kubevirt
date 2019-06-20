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
	v1 "k8s.io/api/autoscaling/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Kubevirt VirtualMachineInstanceReplicaSet Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstancereplicasets"
	rsPath := basePath + "/testrs"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineInstanceReplicaSet", func() {
		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", rsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, rs),
		))
		fetchedVMIReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Get("testrs", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIReplicaSet).To(Equal(rs))
	})

	It("should detect non existent VMIReplicaSets", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", rsPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testrs")),
		))
		_, err := client.ReplicaSet(k8sv1.NamespaceDefault).Get("testrs", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a VirtualMachineInstanceReplicaSet list", func() {
		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVirtualMachineInstanceReplicaSetList(*rs)),
		))
		fetchedVMIReplicaSetList, err := client.ReplicaSet(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedVMIReplicaSetList.Items).To(HaveLen(1))
		Expect(fetchedVMIReplicaSetList.Items[0]).To(Equal(*rs))
	})

	It("should create a VirtualMachineInstanceReplicaSet", func() {
		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, rs),
		))
		createdVMIReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Create(rs)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMIReplicaSet).To(Equal(rs))
	})

	It("should update a VirtualMachineInstanceReplicaSet", func() {
		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", rsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, rs),
		))
		updatedVMIReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Update(rs)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIReplicaSet).To(Equal(rs))
	})

	It("should update a VirtualMachineInstanceReplicaSet scale subresource", func() {
		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		scale := &v1.Scale{Spec: v1.ScaleSpec{Replicas: 3}}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", rsPath+"/scale"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, scale),
		))
		scaleResponse, err := client.ReplicaSet(k8sv1.NamespaceDefault).UpdateScale(rs.Name, scale)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(scaleResponse).To(Equal(scale))
	})

	It("should get a VirtualMachineInstanceReplicaSet scale subresource", func() {
		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		scale := &v1.Scale{Spec: v1.ScaleSpec{Replicas: 3}}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", rsPath+"/scale"),
			ghttp.RespondWithJSONEncoded(http.StatusOK, scale),
		))
		scaleResponse, err := client.ReplicaSet(k8sv1.NamespaceDefault).GetScale(rs.Name, k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(scaleResponse).To(Equal(scale))
	})

	It("should delete a VirtualMachineInstanceReplicaSet", func() {
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
