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

package inotifyinformer

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Inotify", func() {

	Context("When watching files in a directory", func() {

		var tmpDir string
		var informer cache.SharedIndexInformer
		var stopInformer chan struct{}
		var queue workqueue.RateLimitingInterface

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

		BeforeEach(func() {
			var err error
			stopInformer = make(chan struct{})
			tmpDir, err = ioutil.TempDir("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			// create two files
			Expect(os.Create(tmpDir + "/" + "default_testvmi")).ToNot(BeNil())
			Expect(os.Create(tmpDir + "/" + "default1_testvmi1")).ToNot(BeNil())

			queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			informer = cache.NewSharedIndexInformer(
				NewFileListWatchFromClient(tmpDir),
				&api.Domain{},
				0,
				cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

			informer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(queue))
			go informer.Run(stopInformer)
			Expect(cache.WaitForCacheSync(stopInformer, informer.HasSynced)).To(BeTrue())

		})

		It("should update the cache with all files in the directory", func() {
			Expect(informer.GetStore().ListKeys()).To(HaveLen(2))
			_, exists, _ := informer.GetStore().GetByKey("default/testvmi")
			Expect(exists).To(BeTrue())
			_, exists, _ = informer.GetStore().GetByKey("default1/testvmi1")
			Expect(exists).To(BeTrue())
		})

		It("should detect multiple creations and deletions", func() {
			num := 5
			key := "default2/test.vmi2"
			fileName := tmpDir + "/" + "default2_test.vmi2"

			for i := 0; i < num; i++ {
				Expect(os.Create(fileName)).ToNot(BeNil())
				Expect(TestForKeyEvent(key, true)).To(BeTrue())

				Expect(os.Remove(fileName)).To(Succeed())
				Expect(TestForKeyEvent(key, false)).To(BeTrue())
			}

		})

		Context("and something goes wrong", func() {
			It("should notify and abort when listing files", func() {
				lw := NewFileListWatchFromClient(tmpDir)
				// Deleting the watch directory should have some impact
				Expect(os.RemoveAll(tmpDir)).To(Succeed())
				_, err := lw.List(v1.ListOptions{})
				Expect(err).To(HaveOccurred())
			})
			It("should ignore invalid file content", func() {
				lw := NewFileListWatchFromClient(tmpDir)
				_, err := lw.List(v1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				i, err := lw.Watch(v1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				defer i.Stop()

				// Adding files in wrong formats should have an impact
				// TODO should we just ignore them?
				Expect(os.Create(tmpDir + "/" + "test")).ToNot(BeNil())

				// No event should be received
				Consistently(i.ResultChan()).ShouldNot(Receive())
			})
		})

		AfterEach(func() {
			close(stopInformer)
			os.RemoveAll(tmpDir)
		})

	})
})
