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

var _ = Describe("Kubevirt VirtualMachineSnapshot Client", func() {

	var server *ghttp.Server
	var client KubevirtClient
	basePath := "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachinesnapshots"
	vmsPath := basePath + "/testvms"

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fetch a VirtualMachineSnapshot", func() {
		vms := NewMinimalVirtualMachineSnapshot("testvms")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vms),
		))
		fetchedVMSnapshot, err := client.VirtualMachineSnapshot(k8sv1.NamespaceDefault).Get("testvms", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(fetchedVMSnapshot).To(Equal(vms))
	})

	It("should detect non existent VMSnapshots", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", vmsPath),
			ghttp.RespondWithJSONEncoded(http.StatusNotFound, errors.NewNotFound(schema.GroupResource{}, "testvms")),
		))
		_, err := client.VirtualMachineSnapshot(k8sv1.NamespaceDefault).Get("testvms", k8smetav1.GetOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).To(HaveOccurred())
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should fetch a VMSnapshot list", func() {
		ss := NewMinimalVirtualMachineSnapshot("testvms")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("GET", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, NewVirtualMachineSnapshotList(*ss)),
		))
		fetchedVMSnapshotList, err := client.VirtualMachineSnapshot(k8sv1.NamespaceDefault).List(k8smetav1.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(fetchedVMSnapshotList.Items).To(HaveLen(1))
		Expect(fetchedVMSnapshotList.Items[0]).To(Equal(*ss))
	})

	It("should create a VMSnapshot", func() {
		ss := NewMinimalVirtualMachineSnapshot("testvms")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("POST", basePath),
			ghttp.RespondWithJSONEncoded(http.StatusCreated, ss),
		))
		createdVMSnapshot, err := client.VirtualMachineSnapshot(k8sv1.NamespaceDefault).Create(ss)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(createdVMSnapshot).To(Equal(ss))
	})

	It("should update a VMSnapshot", func() {
		ss := NewMinimalVirtualMachineSnapshot("testvms")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", vmsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, ss),
		))
		updatedVMSnapshot, err := client.VirtualMachineSnapshot(k8sv1.NamespaceDefault).Update(ss)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVMSnapshot).To(Equal(ss))
	})

	It("should delete a VMSnapshot", func() {
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("DELETE", vmsPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, nil),
		))
		err := client.VirtualMachineSnapshot(k8sv1.NamespaceDefault).Delete("testvms", &k8smetav1.DeleteOptions{})

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})
})
