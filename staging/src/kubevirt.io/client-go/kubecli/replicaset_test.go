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
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	v1 "k8s.io/api/autoscaling/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	virtv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Kubevirt VirtualMachineInstanceReplicaSet Client", func() {
	var server *ghttp.Server
	basePath := "/apis/kubevirt.io/v1/namespaces/default/virtualmachineinstancereplicasets"
	rsPath := path.Join(basePath, "testrs")
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	DescribeTable("should fetch a VirtualMachineInstanceReplicaSet", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, rsPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, rs),
		))
		fetchedVMIReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Get("testrs", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIReplicaSet).To(Equal(rs))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent VMIReplicaSets", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, rsPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testrs")),
		))
		_, err = client.ReplicaSet(k8sv1.NamespaceDefault).Get("testrs", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a VirtualMachineInstanceReplicaSet list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVirtualMachineInstanceReplicaSetList(*rs)),
		))
		fetchedVMIReplicaSetList, err := client.ReplicaSet(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})
		apiVersion, kind := virtv1.VirtualMachineInstanceReplicaSetGroupVersionKind.ToAPIVersionAndKind()

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedVMIReplicaSetList.Items).To(HaveLen(1))
		Expect(fetchedVMIReplicaSetList.Items[0].APIVersion).To(Equal(apiVersion))
		Expect(fetchedVMIReplicaSetList.Items[0].Kind).To(Equal(kind))
		Expect(fetchedVMIReplicaSetList.Items[0]).To(Equal(*rs))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a VirtualMachineInstanceReplicaSet", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, rs),
		))
		createdVMIReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Create(rs)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMIReplicaSet).To(Equal(rs))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a VirtualMachineInstanceReplicaSet", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, rsPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, rs),
		))
		updatedVMIReplicaSet, err := client.ReplicaSet(k8sv1.NamespaceDefault).Update(rs)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIReplicaSet).To(Equal(rs))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a VirtualMachineInstanceReplicaSet scale subresource", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		scale := &v1.Scale{Spec: v1.ScaleSpec{Replicas: 3}}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, rsPath, "scale")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, scale),
		))
		scaleResponse, err := client.ReplicaSet(k8sv1.NamespaceDefault).UpdateScale(rs.Name, scale)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(scaleResponse).To(Equal(scale))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should get a VirtualMachineInstanceReplicaSet scale subresource", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		rs := NewMinimalVirtualMachineInstanceReplicaSet("testrs")
		scale := &v1.Scale{Spec: v1.ScaleSpec{Replicas: 3}}
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, rsPath, "scale")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, scale),
		))
		scaleResponse, err := client.ReplicaSet(k8sv1.NamespaceDefault).GetScale(rs.Name, k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(scaleResponse).To(Equal(scale))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a VirtualMachineInstanceReplicaSet", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, rsPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.ReplicaSet(k8sv1.NamespaceDefault).Delete("testrs", &k8smetav1.DeleteOptions{})

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
