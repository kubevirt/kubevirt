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
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
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

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			d := &domainWatcher{
				watchdogTimeout:     10,
				unresponsiveSockets: make(map[string]int64),
				resyncPeriod:        1 * time.Hour,
				runServer: func(_ context.Context, _ chan watch.Event) error {
					return fmt.Errorf("permanent failure")
				},
				result: make(chan watch.Event, 100),
				ctx:    ctx,
				cancel: cancel,
			}
			d.wg.Add(1)

			Expect(d.worker).To(PanicWith(
				ContainSubstring("domain notify server reached max consecutive failures")))
		})
	})

	Context("consecutive failure across watcher restarts", func() {
		It("should accumulate failures across Watch() calls via ListerWatcher", func() {
			origMax := notifyServerMaxConsecutiveFails
			origHealthy := notifyServerHealthyRunTime
			defer func() {
				notifyServerMaxConsecutiveFails = origMax
				notifyServerHealthyRunTime = origHealthy
			}()
			notifyServerMaxConsecutiveFails = 5
			notifyServerHealthyRunTime = 1 * time.Hour

			failCount := 3
			lw := newListWatchFromNotify(
				func(_ context.Context, _ chan watch.Event) error {
					return fmt.Errorf("permanent failure")
				},
				10,
				1*time.Hour,
				nil,
			)

			// Simulate what SharedInformer does: call Watch(), drain the
			// result channel, then call Watch() again on failure.
			for i := 0; i < failCount; i++ {
				w, err := lw.Watch(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				// Drain until channel closes (worker exited)
				for range w.ResultChan() {
				}
			}

			// After failCount Watch() restarts, the next watcher should
			// have the accumulated counter. If each Watch() creates a
			// fresh domainWatcher without sharing the counter, this
			// will be 0 instead of failCount.
			w, err := lw.Watch(metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			dw := w.(*domainWatcher)
			Expect(dw.consecutiveFails).To(Equal(failCount))
			dw.Stop()
		})
	})

	Context("Stop() idempotency", func() {
		It("should not panic when Stop is called twice", func() {
			d := &domainWatcher{
				watchdogTimeout:     1,
				unresponsiveSockets: make(map[string]int64),
				resyncPeriod:        1 * time.Hour,
				runServer: func(context.Context, chan watch.Event) error {
					return fmt.Errorf("injected error")
				},
			}

			Expect(d.startBackground()).To(Succeed())
			Eventually(func() bool {
				d.Lock()
				defer d.Unlock()
				return !d.backgroundWatcherStarted
			}, 5*time.Second).Should(BeTrue())

			Expect(func() { d.Stop() }).ShouldNot(Panic())
			Expect(func() { d.Stop() }).ShouldNot(Panic())
		})
	})
})
