/*
 * This file is part of the kubevirt project
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

package watch

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/util/uuid"
	clientv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kubev1 "k8s.io/client-go/pkg/api/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VM watcher", func() {
	var server *ghttp.Server
	var vmService services.VMService
	var templateService services.TemplateService
	var restClient *rest.RESTClient
	var vmCache cache.Store
	var vmQueue workqueue.RateLimitingInterface
	var vmController *VMController

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	Register()
	RegisterTestObjects()

	BeforeEach(func() {
		CC.Clear()
		server = GetTestServer(CC)
		launcherImage = "kubevirt/virt-launcher"
		migratorImage = "kubevirt/virt-handler"
		templateService = GetTemplateService(CC).TemplateService
		restClient = GetRestClient(CC)
		vmService = GetVMService(CC).VMService
		vmCache = GetCache(CC, "vm")
		vmQueue = GetQueue(CC, "vm")
		vmController = GetVMController(CC)
	})

	Context("Creating a VM ", func() {
		It("should should schedule a POD.", func(done Done) {

			// Create a VM to be scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = ""
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			// Create the expected VM after the update
			obj, err := conversion.NewCloner().DeepCopy(vm)
			Expect(err).ToNot(HaveOccurred())

			// Create a Pod for the VM
			pod, err := templateService.RenderLaunchManifest(vm)
			Expect(err).ToNot(HaveOccurred())
			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = clientv1.PodSucceeded

			podListInitial := clientv1.PodList{}
			podListInitial.Items = []clientv1.Pod{}

			podListPostCreate := clientv1.PodList{}
			podListPostCreate.Items = []clientv1.Pod{*pod}

			expectedVM := obj.(*v1.VM)
			expectedVM.Status.Phase = v1.Scheduling
			expectedVM.Status.MigrationNodeName = pod.Spec.NodeName

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podListInitial),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),

				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			// Tell the controller that there is a new VM
			key, _ := cache.MetaNamespaceKeyFunc(vm)
			vmCache.Add(vm)
			vmQueue.Add(key)
			vmController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			close(done)
		}, 10)
	})

	Context("Running Pod for unscheduled VM given", func() {
		It("should update the VM with the node of the running Pod", func(done Done) {

			// Create a VM which is being scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = v1.Scheduling
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			// Create a target Pod for the VM
			var pod *kubev1.Pod
			pod, err := templateService.RenderLaunchManifest(vm)
			Expect(err).ToNot(HaveOccurred())
			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = kubev1.PodRunning
			pods := clientv1.PodList{
				Items: []kubev1.Pod{*pod},
			}

			// Create the expected VM after the update
			obj, err := conversion.NewCloner().DeepCopy(vm)
			Expect(err).ToNot(HaveOccurred())

			expectedVM := obj.(*v1.VM)
			expectedVM.Status.Phase = v1.Scheduled
			expectedVM.Status.NodeName = pod.Spec.NodeName
			expectedVM.ObjectMeta.Labels = map[string]string{v1.NodeNameLabel: pod.Spec.NodeName}

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pods),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.VerifyJSONRepresenting(expectedVM),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedVM),
				),
			)

			// Tell the controller that there is a new running Pod
			key, _ := cache.MetaNamespaceKeyFunc(vm)
			vmCache.Add(vm)
			vmQueue.Add(key)
			vmController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(2))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		server.Close()
	})
})
