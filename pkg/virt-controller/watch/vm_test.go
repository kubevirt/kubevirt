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
	"strings"

	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	clientv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kubev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VM watcher", func() {
	var server *ghttp.Server
	//var vmService services.VMService

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	var app VirtControllerApp = VirtControllerApp{}
	app.launcherImage = "kubevirt/virt-launcher"
	app.migratorImage = "kubevirt/virt-handler"

	BeforeEach(func() {

		server = ghttp.NewServer()
		config := rest.Config{}
		config.Host = server.URL()
		app.clientSet, _ = kubernetes.NewForConfig(&config)
		app.restClient, _ = kubecli.GetRESTClientFromFlags(server.URL(), "")
		app.vmCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		app.vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

		app.initCommon()
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
			temlateService, err := services.NewTemplateService("whatever", "whatever")
			Expect(err).ToNot(HaveOccurred())
			pod, err := temlateService.RenderLaunchManifest(vm)
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
			app.vmCache.Add(vm)
			app.vmQueue.Add(key)
			app.vmController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			close(done)
		}, 10)

		It("should should schedule a POD with Registry Disk.", func(done Done) {

			// Create a VM to be scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = ""
			vm.ObjectMeta.SetUID(uuid.NewUUID())
			vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, v1.Disk{
				Type:   "ContainerRegistryDisk:v1alpha",
				Device: "disk",
				Source: v1.DiskSource{
					Name: "someimage:v1.2.3.4",
				},
				Target: v1.DiskTarget{
					Device: "vda",
				},
			})

			// Create a Pod for the VM
			templateService, err := services.NewTemplateService("whatever", "whatever")
			Expect(err).ToNot(HaveOccurred())

			// We want to ensure the vm object we initially post
			// doesn't have ports set, so we make a copy in order
			// to render the pod object early for the test.
			vmCopy := v1.VM{}
			model.Copy(&vmCopy, vm)
			registrydisk.ApplyPorts(&vmCopy)
			pod, err := templateService.RenderLaunchManifest(&vmCopy)
			Expect(err).ToNot(HaveOccurred())

			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = clientv1.PodSucceeded

			for idx, _ := range pod.Status.ContainerStatuses {
				if strings.Contains(pod.Status.ContainerStatuses[idx].Name, "disk") == false {
					pod.Status.ContainerStatuses[idx].Ready = true
				}
			}

			podListInitial := clientv1.PodList{}
			podListInitial.Items = []clientv1.Pod{}

			podListPostCreate := clientv1.PodList{}
			podListPostCreate.Items = []clientv1.Pod{*pod}

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
			app.vmCache.Add(vm)
			app.vmQueue.Add(key)
			app.vmController.Execute()

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
			temlateService, err := services.NewTemplateService("whatever", "whatever")
			Expect(err).ToNot(HaveOccurred())
			var pod *kubev1.Pod
			pod, err = temlateService.RenderLaunchManifest(vm)
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
			app.vmCache.Add(vm)
			app.vmQueue.Add(key)
			app.vmController.Execute()

			Expect(len(server.ReceivedRequests())).To(Equal(2))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		server.Close()
	})
})
