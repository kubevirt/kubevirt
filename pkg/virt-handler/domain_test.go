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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/uuid"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	. "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
)

var _ = Describe("Domain", func() {

	var vmStore cache.Store
	var vmQueue workqueue.RateLimitingInterface
	var domainStore cache.Store
	var domainQueue workqueue.RateLimitingInterface
	var dispatch kubecli.ControllerDispatch
	var restClient rest.RESTClient

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		vmStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		dispatch = NewDomainDispatch(vmQueue, vmStore, restClient, record.NewFakeRecorder(100))

		domainStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		domainQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	})

	Context("A new domain appears on the host", func() {
		It("should inform vm controller if no correspnding VM is in the cache", func() {
			dom := api.NewMinimalDomain("testvm")

			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = ""
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			vmStore.Add(vm)
			key, _ := cache.MetaNamespaceKeyFunc(dom)
			domainStore.Add(dom)
			domainQueue.Add(key)
			kubecli.Dequeue(domainStore, domainQueue, dispatch)

			Expect(vmQueue.Len()).To(Equal(1))
		})
		It("should inform vm controller if a VM with a different UUID is in the cache", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.GetObjectMeta().SetUID(types.UID("uuid1"))
			vmStore.Add(vm)

			domain := api.NewMinimalDomain("testvm")
			domain.GetObjectMeta().SetUID(types.UID("uuid2"))
			domainStore.Add(domain)
			key, _ := cache.MetaNamespaceKeyFunc(domain)
			domainQueue.Add(key)
			kubecli.Dequeue(domainStore, domainQueue, dispatch)
			Expect(vmQueue.Len()).To(Equal(1))
		})
		It("should not inform vm controller if a correspnding VM is in the cache", func() {
			vmStore.Add(v1.NewMinimalVM("testvm"))
			domain := api.NewMinimalDomain("testvm")
			domainStore.Add(domain)
			key, _ := cache.MetaNamespaceKeyFunc(domain)
			domainQueue.Add(key)
			dispatch.Execute(domainStore, domainQueue, key)
			Expect(vmQueue.Len()).To(Equal(0))
		})

		It("should error out if the key is unparsable", func() {
			key := "a/b/c/d"
			domainQueue.Add(key)
			kubecli.Dequeue(domainStore, domainQueue, dispatch)
			Expect(domainQueue.NumRequeues(key)).To(Equal(1))
			Expect(vmQueue.Len()).To(Equal(0))
		})

		It("should not requeue if domain reference is not in the cache", func() {
			vmStore.Add(v1.NewMinimalVM("testvm"))
			domain := api.NewMinimalDomain("testvm")
			key, _ := cache.MetaNamespaceKeyFunc(domain)
			domainQueue.Add(key)
			kubecli.Dequeue(domainStore, domainQueue, dispatch)
			Expect(domainQueue.NumRequeues(key)).To(Equal(0))
		})

	})

	AfterEach(func() {
	})
})
