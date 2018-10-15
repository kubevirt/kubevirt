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

package rest

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VirtualMachineInstance Subresources", func() {
	var server *ghttp.Server

	log.Log.SetIOWriter(GinkgoWriter)

	configCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	app := SubresourceAPIApp{}
	BeforeEach(func() {
		server = ghttp.NewServer()
		app.VirtCli, _ = kubecli.GetKubevirtClientFromFlags(server.URL(), "")
	})

	Context("Subresource api", func() {
		It("should find matching pod for running VirtualMachineInstance", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())
			templateService := services.NewTemplateService("whatever", "whatever", "whatever", "whatever", configCache, pvcCache)

			pod, err := templateService.RenderLaunchManifest(vmi)
			Expect(err).ToNot(HaveOccurred())
			pod.ObjectMeta.Name = "madeup-name"

			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = k8sv1.PodRunning

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}
			podList.Items = append(podList.Items, *pod)

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			podName, httpStatusCode, err := app.remoteExecInfo(vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(podName).To(Equal("madeup-name"))
			Expect(httpStatusCode).To(Equal(http.StatusOK))
			close(done)
		}, 5)

		It("should fail if VirtualMachineInstance is not in running state", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Succeeded
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			_, httpStatusCode, err := app.remoteExecInfo(vmi)

			Expect(err).To(HaveOccurred())
			Expect(httpStatusCode).To(Equal(http.StatusBadRequest))
			close(done)
		}, 5)

		It("should fail no matching pod is found", func(done Done) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running
			vmi.ObjectMeta.SetUID(uuid.NewUUID())

			podList := k8sv1.PodList{}
			podList.Items = []k8sv1.Pod{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
			)

			_, httpStatusCode, err := app.remoteExecInfo(vmi)

			Expect(err).To(HaveOccurred())
			Expect(httpStatusCode).To(Equal(http.StatusBadRequest))
			close(done)
		}, 5)
	})

	AfterEach(func() {
		server.Close()
	})
})
