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
 * Copyright The KubeVirt Authors.
 *
 */

package cache

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/watch"
	k8scache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("Domain Watcher", func() {
	Context("listSockets ", func() {
		It("should return socket list from ghost record cache", func() {
			const podUID = "5678"
			const socketPath = "/path/to/domainsock"

			ghostCacheDir := GinkgoT().TempDir()

			ghostRecordStore := InitializeGhostRecordCache(NewIterableCheckpointManager(ghostCacheDir))

			err := ghostRecordStore.Add("test-ns", "test-domain", socketPath, podUID)
			Expect(err).ToNot(HaveOccurred())

			socketFiles, err := listSockets(ghostRecordStore.list())
			Expect(err).ToNot(HaveOccurred())
			Expect(socketFiles).To(HaveLen(1))
			Expect(socketFiles[0]).To(Equal(socketPath))

		})
	})

	Context("consecutive failure panic", func() {
		It("should panic after reaching max consecutive failures", func() {
			origMax := notifyServerMaxConsecutiveFails
			origHealthy := notifyServerHealthyRunTime
			defer func() {
				notifyServerMaxConsecutiveFails = origMax
				notifyServerHealthyRunTime = origHealthy
			}()
			notifyServerMaxConsecutiveFails = 1
			notifyServerHealthyRunTime = 1 * time.Hour

			d := &domainWatcher{
				virtShareDir:        GinkgoT().TempDir(),
				watchdogTimeout:     10,
				unresponsiveSockets: make(map[string]int64),
				resyncPeriod:        1 * time.Hour,
				runServer: func(string, chan struct{}, chan watch.Event, record.EventRecorder, k8scache.Store, ...time.Duration) error {
					return fmt.Errorf("permanent failure")
				},
				eventChan: make(chan watch.Event, 100),
				stopChan:  make(chan struct{}),
			}
			d.wg.Add(1)

			Expect(d.worker).To(PanicWith(
				ContainSubstring("domain notify server reached max consecutive failures")))
		})
	})

	Context("Stop() idempotency", func() {
		It("should not panic when Stop is called twice", func() {
			d := &domainWatcher{
				virtShareDir:        GinkgoT().TempDir(),
				watchdogTimeout:     1,
				unresponsiveSockets: make(map[string]int64),
				resyncPeriod:        1 * time.Hour,
				runServer: func(string, chan struct{}, chan watch.Event, record.EventRecorder, k8scache.Store, ...time.Duration) error {
					return fmt.Errorf("injected error")
				},
			}

			Expect(d.startBackground()).To(Succeed())
			Eventually(func() bool {
				d.lock.Lock()
				defer d.lock.Unlock()
				return !d.backgroundWatcherStarted
			}, 5*time.Second).Should(BeTrue())

			Expect(func() { d.Stop() }).ShouldNot(Panic())
			Expect(func() { d.Stop() }).ShouldNot(Panic())
		})
	})
})
