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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package kubecli

import (
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
)

var _ = Describe("Kubevirt Client", func() {
	var server *ghttp.Server
	basePath := "/apis/kubevirt.io/v1/namespaces/default/kubevirts"
	kubevirtPath := path.Join(basePath, "testkubevirt")
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	DescribeTable("should fetch a KubeVirt", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		kubevirt := NewMinimalKubeVirt("testkubevirt")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, kubevirtPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, kubevirt),
		))
		fetchedKubeVirt, err := client.KubeVirt(k8sv1.NamespaceDefault).Get("testkubevirt", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedKubeVirt).To(Equal(kubevirt))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent KubeVirts", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, kubevirtPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testkubevirt")),
		))
		_, err = client.KubeVirt(k8sv1.NamespaceDefault).Get("testkubevirt", &k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a KubeVirt list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		kubevirt := NewMinimalKubeVirt("testkubevirt")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewKubeVirtList(*kubevirt)),
		))
		fetchedKubeVirtList, err := client.KubeVirt(k8sv1.NamespaceDefault).List(&k8smetav1.ListOptions{})
		apiVersion, kind := v1.KubeVirtGroupVersionKind.ToAPIVersionAndKind()

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedKubeVirtList.Items).To(HaveLen(1))
		Expect(fetchedKubeVirtList.Items[0].APIVersion).To(Equal(apiVersion))
		Expect(fetchedKubeVirtList.Items[0].Kind).To(Equal(kind))
		Expect(fetchedKubeVirtList.Items[0]).To(Equal(*kubevirt))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a KubeVirt", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		kubevirt := NewMinimalKubeVirt("testkubevirt")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, kubevirt),
		))
		createdKubeVirt, err := client.KubeVirt(k8sv1.NamespaceDefault).Create(kubevirt)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdKubeVirt).To(Equal(kubevirt))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a KubeVirt", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		kubevirt := NewMinimalKubeVirt("testkubevirt")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, kubevirtPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, kubevirt),
		))
		updatedKubeVirt, err := client.KubeVirt(k8sv1.NamespaceDefault).Update(kubevirt)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedKubeVirt).To(Equal(kubevirt))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should patch a KubeVirt", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		kubevirt := NewMinimalKubeVirt("testkubevirt")
		kubevirt.Spec.ImagePullPolicy = "somethingelse"

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PATCH", path.Join(proxyPath, kubevirtPath)),
			ghttp.VerifyBody([]byte("{\"spec\":{\"imagePullPolicy\":something}}")),
			ghttp.RespondWithJSONEncoded(http.StatusOK, kubevirt),
		))

		_, err = client.KubeVirt(k8sv1.NamespaceDefault).Patch(kubevirt.Name, types.MergePatchType,
			[]byte("{\"spec\":{\"imagePullPolicy\":something}}"), &k8smetav1.PatchOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a KubeVirt", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, kubevirtPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.KubeVirt(k8sv1.NamespaceDefault).Delete("testkubevirt", &k8smetav1.DeleteOptions{})

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
