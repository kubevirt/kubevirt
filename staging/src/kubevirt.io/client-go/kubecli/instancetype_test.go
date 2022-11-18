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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package kubecli

import (
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Kubevirt ExpandSpec Client", func() {
	var server *ghttp.Server
	var client KubevirtClient
	expandSpecPath := fmt.Sprintf("/apis/subresources.kubevirt.io/%s/namespaces/%s/expand-vm-spec", v1.SubresourceStorageGroupVersion.Version, k8sv1.NamespaceDefault)

	BeforeEach(func() {
		var err error
		server = ghttp.NewServer()
		client, err = GetKubevirtClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should expand a VirtualMachine", func() {
		vm := NewMinimalVM("testvm")
		server.AppendHandlers(ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", expandSpecPath),
			ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
		))
		expandedVM, err := client.ExpandSpec(k8sv1.NamespaceDefault).ForVirtualMachine(vm)

		Expect(server.ReceivedRequests()).To(HaveLen(1))
		Expect(err).ToNot(HaveOccurred())
		Expect(expandedVM).To(Equal(vm))
	})

	AfterEach(func() {
		server.Close()
	})
})
