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

package services_test

import (
	"encoding/json"
	"flag"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	corev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	. "kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VM", func() {

	var vmService VMService
	var server *ghttp.Server

	BeforeEach(func() {

		flag.Parse()
		server = ghttp.NewServer()
		virtClient, _ := kubecli.GetKubevirtClientFromFlags(server.URL(), "")
		templateService, _ := NewTemplateService("kubevirt/virt-launcher", "/var/run/libvirt", "imagesecret")
		vmService = NewVMService(virtClient, templateService)

	})

	Context("calling StartVM Pod for a pod that does not exists", func() {
		It("should create the pod", func() {
			vm := v1.NewMinimalVM("test-vm")
			vm.ObjectMeta.UID = "testUID"
			pod := corev1.Pod{}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),
			)
			_, err := vmService.StartVMPod(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(server.ReceivedRequests())).To(Equal(1))
		})
	})

})

func VerifyAffinity(affinity *corev1.Affinity) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		pod := &corev1.Pod{}
		Expect(json.NewDecoder(req.Body).Decode(pod)).To(Succeed())
		Expect(pod.Spec.Affinity).To(Equal(affinity))
	}
}
