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
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Kubevirt VirtualMachineInstancePreset Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstancepresets"
	presetPath := basePath + "/testpreset"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineInstancePreset", func() {
		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", presetPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, preset),
		))
		fetchedVMIPreset, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Get("testpreset", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMIPreset).To(Equal(preset))
	})

	It("should detect non existent VMIPresets", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", presetPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testpreset")),
		))
		_, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Get("testpreset", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue(), "Expected an IsNotFound error to have occurred")
	})

	It("should fetch a VirtualMachineInstancePreset list", func() {
		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVirtualMachineInstancePresetList(*preset)),
		))
		fetchedVMIPresetList, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedVMIPresetList.Items).To(HaveLen(1))
		Expect(fetchedVMIPresetList.Items[0]).To(Equal(*preset))
	})

	It("should create a VirtualMachineInstancePreset", func() {
		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, preset),
		))
		createdVMIPreset, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Create(preset)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMIPreset).To(Equal(preset))
	})

	It("should update a VirtualMachineInstancePreset", func() {
		preset := NewMinimalVirtualMachineInstancePreset("testpreset")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", presetPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, preset),
		))
		updatedVMIPreset, err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Update(preset)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMIPreset).To(Equal(preset))
	})

	It("should delete a VirtualMachineInstancePreset", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", presetPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachineInstancePreset(k8sv1.NamespaceDefault).Delete("testpreset", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})
