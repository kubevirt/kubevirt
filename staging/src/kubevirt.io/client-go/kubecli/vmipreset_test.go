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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Kubevirt VirtualMachineInstancePreset Client", func() {
	var server *ghttp.Server
	basePath := "/apis/kubevirt.io/v1/namespaces/default/virtualmachineinstancepresets"
	presetPath := path.Join(basePath, "testpreset")
	proxyPath := "/proxy/path"

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	DescribeTable("should fetch a VirtualMachineInstancePreset", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, presetPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, preset),
		))
		fetchedVMIPreset, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Get("testpreset", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIPreset).To(Equal(preset))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should detect non existent VMIPresets", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, presetPath)),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testpreset")),
		))
		_, err = client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Get("testpreset", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue(), "Expected an IsNotFound error to have occurred")
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should fetch a VirtualMachineInstancePreset list", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVirtualMachineInstancePresetList(*preset)),
		))
		fetchedVMIPresetList, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})
		apiVersion, kind := v1.VirtualMachineInstancePresetGroupVersionKind.ToAPIVersionAndKind()

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedVMIPresetList.Items).To(HaveLen(1))
		Expect(fetchedVMIPresetList.Items[0].APIVersion).To(Equal(apiVersion))
		Expect(fetchedVMIPresetList.Items[0].Kind).To(Equal(kind))
		Expect(fetchedVMIPresetList.Items[0]).To(Equal(*preset))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should create a VirtualMachineInstancePreset", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", path.Join(proxyPath, basePath)),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, preset),
		))
		createdVMIPreset, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Create(preset)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMIPreset).To(Equal(preset))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should update a VirtualMachineInstancePreset", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", path.Join(proxyPath, presetPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, preset),
		))
		updatedVMIPreset, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Update(preset)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIPreset).To(Equal(preset))
	},
		Entry("with regular server URL", ""),
		Entry("with proxied server URL", proxyPath),
	)

	DescribeTable("should delete a VirtualMachineInstancePreset", func(proxyPath string) {
		client, err := GetKubevirtClientFromFlags(server.URL()+proxyPath, "")
		Expect(err).ToNot(HaveOccurred())

		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", path.Join(proxyPath, presetPath)),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err = client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Delete("testpreset", &k8smetav1.DeleteOptions{})

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
