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

package watchdog

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Watchdog", func() {

	Context("When watching files in a directory", func() {

		var tmpVirtShareDir string
		var tmpWatchdogDir string
		var informer cache.SharedIndexInformer
		var stopInformer chan struct{}
		var queue workqueue.RateLimitingInterface

		startedInformer := false

		TestForKeyEvent := func(expectedKey string, shouldExist bool) bool {
			// wait for key to either enter or exit the store.
			Eventually(func() bool {
				_, exists, _ := informer.GetStore().GetByKey(expectedKey)

				if shouldExist == exists {
					return true
				}
				return false
			}).Should(BeTrue())

			// ensure queue item for key exists
			len := queue.Len()
			for i := len; i > 0; i-- {
				key, _ := queue.Get()
				defer queue.Done(key)
				if key == expectedKey {
					return true
				}
			}
			return false
		}

		startWatchdogInformer := func() {
			var err error
			stopInformer = make(chan struct{})
			startedInformer = true
			tmpVirtShareDir, err = ioutil.TempDir("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			tmpWatchdogDir = WatchdogFileDirectory(tmpVirtShareDir)
			err = os.Mkdir(tmpWatchdogDir, 0755)
			Expect(err).ToNot(HaveOccurred())

			queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			informer = cache.NewSharedIndexInformer(
				NewWatchdogListWatchFromClient(tmpVirtShareDir, 2),
				&api.Domain{},
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

			informer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(queue))
			go informer.Run(stopInformer)
			Expect(cache.WaitForCacheSync(stopInformer, informer.HasSynced)).To(BeTrue())
		}

		It("should detect expired watchdog files", func() {
			startWatchdogInformer()

			keyExpired := "default/expiredvm"
			fileName := tmpWatchdogDir + "/default_expiredvm"
			Expect(os.Create(fileName)).ToNot(BeNil())

			files, err := detectExpiredFiles(1, tmpWatchdogDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(0))

			time.Sleep(time.Second * 3)

			Expect(TestForKeyEvent(keyExpired, true)).To(Equal(true))

			files, err = detectExpiredFiles(1, tmpWatchdogDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(1))

			Expect(os.Create(fileName)).ToNot(BeNil())
			files, err = detectExpiredFiles(1, tmpWatchdogDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(0))
		})

		It("should successfully remove watchdog file", func() {

			vm := v1.NewMinimalVM("tvm")
			namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
			domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

			startWatchdogInformer()

			keyExpired := fmt.Sprintf("%s/%s", namespace, domain)
			fileName := WatchdogFileFromNamespaceName(tmpVirtShareDir, namespace, domain)
			Expect(os.Create(fileName)).ToNot(BeNil())

			files, err := detectExpiredFiles(1, tmpWatchdogDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(0))

			expired, err := WatchdogFileIsExpired(1, tmpVirtShareDir, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(expired).To(Equal(false))

			time.Sleep(time.Second * 3)

			Expect(TestForKeyEvent(keyExpired, true)).To(Equal(true))

			files, err = detectExpiredFiles(1, tmpWatchdogDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(1))

			expired, err = WatchdogFileIsExpired(1, tmpVirtShareDir, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(expired).To(Equal(true))

			exists, err := WatchdogFileExists(tmpVirtShareDir, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(Equal(true))

			err = WatchdogFileRemove(tmpVirtShareDir, vm)
			Expect(err).ToNot(HaveOccurred())

			files, err = detectExpiredFiles(1, tmpWatchdogDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(files)).To(Equal(0))

			exists, err = WatchdogFileExists(tmpVirtShareDir, vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(Equal(false))
		})

		It("should not expire updated files", func() {
			startWatchdogInformer()

			fileName := tmpVirtShareDir + "/default_expiredvm"
			Expect(os.Create(fileName)).ToNot(BeNil())

			for i := 0; i < 4; i++ {
				WatchdogFileUpdate(fileName)
				time.Sleep(time.Second * 1)
				files, err := detectExpiredFiles(2, tmpWatchdogDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(files)).To(Equal(0))
				Expect(queue.Len()).To(Equal(0))
			}
		})

		It("should provide file in watchdog subdirectory", func() {
			dir := WatchdogFileDirectory(tmpVirtShareDir)
			Expect(dir).To(Equal(tmpVirtShareDir + "/watchdog-files"))

			dir = WatchdogFileFromNamespaceName(tmpVirtShareDir, "tnamespace", "tvm")
			Expect(dir).To(Equal(tmpVirtShareDir + "/watchdog-files/tnamespace_tvm"))
		})

		AfterEach(func() {
			if startedInformer {
				close(stopInformer)
			}
			os.RemoveAll(tmpVirtShareDir)
			startedInformer = false
		})

	})
})
