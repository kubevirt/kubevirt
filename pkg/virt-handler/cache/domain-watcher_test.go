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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

	Context("handleStaleSocketConnections", func() {
		It("should remove a socket from unresponsiveSockets when it is no longer in the socket list", func() {
			ghostCacheDir := GinkgoT().TempDir()
			InitializeGhostRecordCache(NewIterableCheckpointManager(ghostCacheDir))

			const stalePath = "/nonexistent/socket.sock"
			d := &domainWatcher{
				eventChan:       make(chan watch.Event, 10),
				watchdogTimeout: 30,
				unresponsiveSockets: map[string]int64{
					stalePath: time.Now().UTC().Unix(),
				},
			}

			err := d.handleStaleSocketConnections()
			Expect(err).ToNot(HaveOccurred())

			d.watchDogLock.Lock()
			defer d.watchDogLock.Unlock()
			Expect(d.unresponsiveSockets).ToNot(HaveKey(stalePath))
		})

		It("should emit a Modified event when a socket exceeds the watchdog timeout", func() {
			ghostCacheDir := GinkgoT().TempDir()
			ghostRecordStore := InitializeGhostRecordCache(NewIterableCheckpointManager(ghostCacheDir))

			const socketPath = "/nonexistent/socket.sock"
			const uid = "test-uid-1234"
			err := ghostRecordStore.Add("test-ns", "test-vmi", socketPath, uid)
			Expect(err).ToNot(HaveOccurred())

			d := &domainWatcher{
				eventChan:       make(chan watch.Event, 10),
				watchdogTimeout: 1,
				unresponsiveSockets: map[string]int64{
					// Mark the socket as unresponsive well before the timeout.
					socketPath: time.Now().UTC().Add(-10 * time.Second).Unix(),
				},
			}

			err = d.handleStaleSocketConnections()
			Expect(err).ToNot(HaveOccurred())

			Expect(d.eventChan).To(Receive(HaveField("Type", watch.Modified)))
		})

		It("should not hold watchDogLock while sending to eventChan", func() {
			ghostCacheDir := GinkgoT().TempDir()
			ghostRecordStore := InitializeGhostRecordCache(NewIterableCheckpointManager(ghostCacheDir))

			const socketPath = "/nonexistent/socket2.sock"
			const uid = "test-uid-5678"
			err := ghostRecordStore.Add("test-ns", "test-vmi2", socketPath, uid)
			Expect(err).ToNot(HaveOccurred())

			// Zero-buffer channel
			d := &domainWatcher{
				eventChan:       make(chan watch.Event, 0),
				watchdogTimeout: 1,
				unresponsiveSockets: map[string]int64{
					socketPath: time.Now().UTC().Add(-10 * time.Second).Unix(),
				},
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = d.handleStaleSocketConnections()
			}()

			// The goroutine blocks on eventChan send
			Eventually(func() bool {
				if d.watchDogLock.TryLock() {
					d.watchDogLock.Unlock()
					return true
				}
				return false
			}).WithTimeout(2 * time.Second).WithPolling(20 * time.Millisecond).Should(BeTrue())

			// Unblock the goroutine and wait for it to finish.
			<-d.eventChan
			Eventually(done).WithTimeout(2 * time.Second).Should(BeClosed())
		})
	})
})
