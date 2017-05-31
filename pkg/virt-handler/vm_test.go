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

package virthandler_test

import (
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/testutil"
	. "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"

	"k8s.io/client-go/tools/record"
)

var _ = Describe("VM", func() {
	var server *ghttp.Server
	var vmStore cache.Store
	var vmQueue workqueue.RateLimitingInterface
	var domainManager *virtwrap.MockDomainManager

	var ctrl *gomock.Controller
	var dispatch kubecli.ControllerDispatch

	var recorder record.EventRecorder

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		var err error
		server = testutil.NewKubeServer([]testutil.Resource{})
		host := ""

		coreClient, err := kubecli.GetFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())

		restClient, err := kubecli.GetRESTClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())

		vmStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

		ctrl = gomock.NewController(GinkgoT())
		domainManager = virtwrap.NewMockDomainManager(ctrl)

		recorder = record.NewFakeRecorder(100)
		dispatch = NewVMHandlerDispatch(domainManager, recorder, restClient, coreClient, host)

	})

	Context("VM controller gets informed about a Domain change through the Domain controller", func() {
		It("should kill the Domain if no cluster wide equivalent exists", func(done Done) {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			domainManager.EXPECT().KillVM(v1.NewVMReferenceFromName("testvm")).Do(func(vm *v1.VM) {
				close(done)
			})

			dispatch.Execute(vmStore, vmQueue, "default/testvm")
		}, 1)
		It("should leave the Domain alone if the VM is migrating to its host", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.MigrationNodeName = "master"
			testutil.AddServerResource(server, vm)
			vmStore.Add(vm)
			dispatch.Execute(vmStore, vmQueue, "default/testvm")

		})
		It("should re-enqueue if the Key is unparseable", func() {
			Expect(vmQueue.Len()).Should(Equal(0))
			vmQueue.Add("a/b/c/d/e")
			kubecli.Dequeue(vmStore, vmQueue, dispatch)
			Expect(vmQueue.NumRequeues("a/b/c/d/e")).To(Equal(1))
		})

		table.DescribeTable("should leave the VM alone if it is in the final phase", func(phase v1.VMPhase) {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = phase
			vmStore.Add(vm)

			vmQueue.Add("default/testvm")
			kubecli.Dequeue(vmStore, vmQueue, dispatch)
			// expect no mock interactions
			Expect(vmQueue.NumRequeues("default/testvm")).To(Equal(0))
		},
			table.Entry("succeeded", v1.Succeeded),
			table.Entry("failed", v1.Failed),
		)
	})

	AfterEach(func() {
		server.Close()
		ctrl.Finish()
	})
})

func TestVMs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PVC")
}
