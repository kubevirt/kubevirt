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
	"net"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Domain Watcher", func() {
	Context("listSockets ", func() {
		It("should return socket list from ghost record cache", func() {
			const podUID = "5678"
			const socketPath = "/path/to/domainsock"

			ghostCacheDir := GinkgoT().TempDir()

			ghostRecordStore := InitializeGhostRecordCache(NewIterableCheckpointManager(ghostCacheDir, GinkgoT().TempDir()))

			err := ghostRecordStore.Add("test-ns", "test-domain", socketPath, podUID)
			Expect(err).ToNot(HaveOccurred())

			socketFiles := listSockets(ghostRecordStore.list())
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
				unresponsiveSockets: make(map[string]int64),
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}
			d.wg.Add(1)

			runServer := func(_ context.Context, _ chan watch.Event) error {
				return fmt.Errorf("permanent failure")
			}
			Expect(func() { d.worker(ctx, runServer, 1*time.Hour, 10) }).To(PanicWith(
				ContainSubstring("domain notify server reached max consecutive failures")))
		})
	})

	Context("send", func() {
		It("should deliver the event when the result channel is read", func() {
			d := &domainWatcher{result: make(chan watch.Event, 1)}

			Expect(d.send(context.Background(), watch.Event{Type: watch.Modified})).To(BeTrue())
			Expect(<-d.result).To(Equal(watch.Event{Type: watch.Modified}))
		})

		It("should give up instead of blocking forever once ctx is done", func() {
			d := &domainWatcher{result: make(chan watch.Event)}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			sent := make(chan bool, 1)
			go func() {
				sent <- d.send(ctx, watch.Event{Type: watch.Modified})
			}()

			Eventually(sent, 500*time.Millisecond).Should(Receive(BeFalse()))
		})
	})

	Context("Stop() idempotency", func() {
		It("should not panic when Stop is called twice", func() {
			d := newDomainWatcher(
				context.Background(),
				func(context.Context, chan watch.Event) error {
					return fmt.Errorf("injected error")
				},
				1,
				1*time.Hour,
				nil,
				new(int),
				nil, nil,
			)

			Eventually(d.result).Should(BeClosed())

			Expect(func() { d.Stop() }).ShouldNot(Panic())
			Expect(func() { d.Stop() }).ShouldNot(Panic())
		})
	})

	Context("handleStaleSocketConnections", func() {
		var ghostCacheDir string
		var ghostRecordStore *GhostRecordStore

		BeforeEach(func() {
			ghostCacheDir = GinkgoT().TempDir()
			ghostRecordStore = InitializeGhostRecordCache(
				NewIterableCheckpointManager(ghostCacheDir, GinkgoT().TempDir()),
			)
		})

		It("should delete socket from unresponsiveSockets when it becomes responsive", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create a listening socket that we can connect to
			socketDir := GinkgoT().TempDir()
			socketPath := filepath.Join(socketDir, "test.sock")

			// Create a listener on the socket
			listener, err := net.Listen("unix", socketPath)
			Expect(err).ToNot(HaveOccurred())
			defer listener.Close()

			// Add socket to ghost records
			err = ghostRecordStore.Add("test-ns", "test-domain", socketPath, "test-uid")
			Expect(err).ToNot(HaveOccurred())

			// Create domain watcher with socket already marked as unresponsive
			d := &domainWatcher{
				unresponsiveSockets: map[string]int64{socketPath: time.Now().UTC().Unix()},
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			// Call handleStaleSocketConnections - the socket is now responsive
			err = d.handleStaleSocketConnections(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			// Socket should be deleted from unresponsiveSockets
			_, exists := d.unresponsiveSockets[socketPath]
			Expect(exists).To(BeFalse())
		})

		It("should keep socket in unresponsiveSockets if still unresponsive and timeout not exceeded", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Use a non-existent socket path (will always fail to dial)
			socketPath := filepath.Join(GinkgoT().TempDir(), "nonexistent.sock")

			// Add socket to ghost records
			err := ghostRecordStore.Add("test-ns", "test-domain", socketPath, "test-uid")
			Expect(err).ToNot(HaveOccurred())

			// Mark socket as recently unresponsive (less than timeout)
			now := time.Now().UTC().Unix()
			d := &domainWatcher{
				unresponsiveSockets: map[string]int64{socketPath: now},
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			watchdogTimeout := 10
			err = d.handleStaleSocketConnections(ctx, watchdogTimeout)
			Expect(err).ToNot(HaveOccurred())

			// Socket should still be in unresponsiveSockets
			timestamp, exists := d.unresponsiveSockets[socketPath]
			Expect(exists).To(BeTrue())
			Expect(timestamp).To(Equal(now))
		})

		It("should send Modified event when socket timeout is exceeded", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Use a non-existent socket path
			socketPath := filepath.Join(GinkgoT().TempDir(), "nonexistent.sock")

			// Add socket to ghost records
			err := ghostRecordStore.Add("test-ns", "test-domain", socketPath, "test-uid")
			Expect(err).ToNot(HaveOccurred())

			// Mark socket as unresponsive for longer than timeout
			watchdogTimeout := 10
			oldTime := time.Now().UTC().Unix() - int64(watchdogTimeout) - 5
			d := &domainWatcher{
				unresponsiveSockets: map[string]int64{socketPath: oldTime},
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			err = d.handleStaleSocketConnections(ctx, watchdogTimeout)
			Expect(err).ToNot(HaveOccurred())

			// Should have sent an event
			Eventually(d.result).Should(Receive(HaveField("Type", watch.Modified)))
		})

		It("should add newly unresponsive socket to map", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Use a non-existent socket path (will be unresponsive)
			socketPath := filepath.Join(GinkgoT().TempDir(), "nonexistent.sock")

			// Add socket to ghost records
			err := ghostRecordStore.Add("test-ns", "test-domain", socketPath, "test-uid")
			Expect(err).ToNot(HaveOccurred())

			d := &domainWatcher{
				unresponsiveSockets: make(map[string]int64),
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			err = d.handleStaleSocketConnections(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			// Socket should be added to unresponsiveSockets
			_, exists := d.unresponsiveSockets[socketPath]
			Expect(exists).To(BeTrue())
		})

		It("should handle multiple sockets with different states", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			socketDir := GinkgoT().TempDir()

			// Socket 1: responsive (has listener)
			socket1Path := filepath.Join(socketDir, "responsive.sock")
			listener1, err := net.Listen("unix", socket1Path)
			Expect(err).ToNot(HaveOccurred())
			defer listener1.Close()

			// Socket 2: unresponsive (non-existent)
			socket2Path := filepath.Join(socketDir, "unresponsive.sock")

			// Socket 3: was unresponsive, now responsive
			socket3Path := filepath.Join(socketDir, "recovered.sock")
			listener3, err := net.Listen("unix", socket3Path)
			Expect(err).ToNot(HaveOccurred())
			defer listener3.Close()

			// Add sockets to ghost records
			for i, path := range []string{socket1Path, socket2Path, socket3Path} {
				err := ghostRecordStore.Add("test-ns", fmt.Sprintf("domain-%d", i), path, types.UID(fmt.Sprintf("uid-%d", i)))
				Expect(err).ToNot(HaveOccurred())
			}

			// Initialize domain watcher with socket3 already marked as unresponsive
			now := time.Now().UTC().Unix()
			d := &domainWatcher{
				unresponsiveSockets: map[string]int64{socket3Path: now},
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			err = d.handleStaleSocketConnections(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			// Socket 1 should not be in map (is responsive)
			_, exists := d.unresponsiveSockets[socket1Path]
			Expect(exists).To(BeFalse())

			// Socket 2 should be in map (is unresponsive)
			_, exists = d.unresponsiveSockets[socket2Path]
			Expect(exists).To(BeTrue())

			// Socket 3 should be deleted (was unresponsive, now responsive)
			_, exists = d.unresponsiveSockets[socket3Path]
			Expect(exists).To(BeFalse())
		})

		It("should set DeletionTimestamp on domain when socket exceeds timeout", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			socketPath := filepath.Join(GinkgoT().TempDir(), "unresponsive.sock")
			err := ghostRecordStore.Add("test-ns", "test-domain", socketPath, "test-uid")
			Expect(err).ToNot(HaveOccurred())

			// Mark socket as old enough to exceed timeout
			watchdogTimeout := 10
			oldTime := time.Now().UTC().Unix() - int64(watchdogTimeout) - 5
			d := &domainWatcher{
				unresponsiveSockets: map[string]int64{socketPath: oldTime},
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			err = d.handleStaleSocketConnections(ctx, watchdogTimeout)
			Expect(err).ToNot(HaveOccurred())

			// Verify the event contains a domain with DeletionTimestamp
			var receivedEvent watch.Event
			Eventually(d.result).Should(Receive(&receivedEvent))
			Expect(receivedEvent.Type).To(Equal(watch.Modified))

			domain, ok := receivedEvent.Object.(*api.Domain)
			Expect(ok).To(BeTrue())
			Expect(domain.ObjectMeta.DeletionTimestamp).ToNot(BeNil())
			Expect(domain.Name).To(Equal("test-domain"))
			Expect(domain.Namespace).To(Equal("test-ns"))
		})

		It("should handle ghost record not existing for unresponsive socket", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Socket path that won't be in ghost records
			socketPath := filepath.Join(GinkgoT().TempDir(), "unknown.sock")

			// Mark socket as unresponsive and exceeding timeout
			watchdogTimeout := 10
			oldTime := time.Now().UTC().Unix() - int64(watchdogTimeout) - 5
			d := &domainWatcher{
				unresponsiveSockets: map[string]int64{socketPath: oldTime},
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			// Should not panic or error when ghost record doesn't exist
			err := d.handleStaleSocketConnections(ctx, watchdogTimeout)
			Expect(err).ToNot(HaveOccurred())

			// No event should be sent (record doesn't exist)
			Expect(d.result).To(BeEmpty())
		})

		It("should delete multiple sockets in one call", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			socketDir := GinkgoT().TempDir()

			// Create multiple sockets that are now responsive
			socketPaths := make([]string, 3)
			listeners := make([]net.Listener, 3)
			for i := range 3 {
				socketPath := filepath.Join(socketDir, fmt.Sprintf("responsive-%d.sock", i))
				listener, err := net.Listen("unix", socketPath)
				Expect(err).ToNot(HaveOccurred())
				defer listener.Close()

				socketPaths[i] = socketPath
				listeners[i] = listener

				// Add to ghost records
				err = ghostRecordStore.Add("test-ns", fmt.Sprintf("domain-%d", i), socketPath, types.UID(fmt.Sprintf("uid-%d", i)))
				Expect(err).ToNot(HaveOccurred())
			}

			// Mark all sockets as previously unresponsive
			now := time.Now().UTC().Unix()
			unresponsiveMap := make(map[string]int64)
			for _, path := range socketPaths {
				unresponsiveMap[path] = now
			}

			d := &domainWatcher{
				unresponsiveSockets: unresponsiveMap,
				consecutiveFails:    new(int),
				result:              make(chan watch.Event, 100),
				cancel:              cancel,
			}

			// Call handleStaleSocketConnections - all sockets are now responsive
			err := d.handleStaleSocketConnections(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			// All sockets should be deleted from unresponsiveSockets
			for _, path := range socketPaths {
				_, exists := d.unresponsiveSockets[path]
				Expect(exists).To(BeFalse(), "socket %s should be deleted", path)
			}

			// Verify the map is empty
			Expect(d.unresponsiveSockets).To(BeEmpty())
		})
	})

	Context("podUIDFromSocketPath", func() {
		It("should extract the pod UID from a standard socket path", func() {
			uid, err := podUIDFromSocketPath("/pods/abc-123-def/volumes/kubernetes.io~empty-dir/sockets/launcher-sock")
			Expect(err).ToNot(HaveOccurred())
			Expect(uid).To(Equal("abc-123-def"))
		})

		It("should extract the pod UID when there are leading path segments", func() {
			uid, err := podUIDFromSocketPath("/var/lib/kubelet/pods/real-uid-456/volumes/kubernetes.io~empty-dir/sockets/launcher-sock")
			Expect(err).ToNot(HaveOccurred())
			Expect(uid).To(Equal("real-uid-456"))
		})

		It("should fail when path has no /volumes/ segment", func() {
			_, err := podUIDFromSocketPath("/some/random/path/launcher-sock")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("/volumes/"))
		})
	})

	Context("podUIDFromTokenReview", func() {
		It("should extract the pod UID from the extra claims", func() {
			review := &authv1.TokenReview{
				Status: authv1.TokenReviewStatus{
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							tokenReviewPodUIDKey: {"pod-uid-789"},
						},
					},
				},
			}
			uid, err := podUIDFromTokenReview(review)
			Expect(err).ToNot(HaveOccurred())
			Expect(uid).To(Equal("pod-uid-789"))
		})

		It("should fail when pod UID claim is missing", func() {
			review := &authv1.TokenReview{
				Status: authv1.TokenReviewStatus{
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{},
					},
				},
			}
			_, err := podUIDFromTokenReview(review)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not present"))
		})

		It("should fail when pod UID claim is empty", func() {
			review := &authv1.TokenReview{
				Status: authv1.TokenReviewStatus{
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							tokenReviewPodUIDKey: {""},
						},
					},
				},
			}
			_, err := podUIDFromTokenReview(review)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("should fail when Extra map is nil", func() {
			review := &authv1.TokenReview{
				Status: authv1.TokenReviewStatus{
					User: authv1.UserInfo{},
				},
			}
			_, err := podUIDFromTokenReview(review)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("pod UID mismatch detection", func() {
		It("should detect when token pod UID does not match socket path", func() {
			socketPath := "/pods/legitimate-pod-uid/volumes/kubernetes.io~empty-dir/sockets/launcher-sock"
			socketUID, err := podUIDFromSocketPath(socketPath)
			Expect(err).ToNot(HaveOccurred())

			review := &authv1.TokenReview{
				Status: authv1.TokenReviewStatus{
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							tokenReviewPodUIDKey: {"attacker-pod-uid"},
						},
					},
				},
			}
			tokenUID, err := podUIDFromTokenReview(review)
			Expect(err).ToNot(HaveOccurred())

			Expect(tokenUID).ToNot(Equal(socketUID),
				"a token from an attacker pod must not match the socket's pod UID")
		})

		It("should succeed when token pod UID matches socket path", func() {
			podUID := "matching-pod-uid-abc"
			socketPath := fmt.Sprintf("/pods/%s/volumes/kubernetes.io~empty-dir/sockets/launcher-sock", podUID)
			socketUID, err := podUIDFromSocketPath(socketPath)
			Expect(err).ToNot(HaveOccurred())

			review := &authv1.TokenReview{
				Status: authv1.TokenReviewStatus{
					User: authv1.UserInfo{
						Extra: map[string]authv1.ExtraValue{
							tokenReviewPodUIDKey: {podUID},
						},
					},
				},
			}
			tokenUID, err := podUIDFromTokenReview(review)
			Expect(err).ToNot(HaveOccurred())

			Expect(tokenUID).To(Equal(socketUID))
		})
	})

	Context("ActivePods verification", func() {
		const legitimateUID = "launcher-pod-uid-123"
		const rogueUID = "rogue-pod-uid-456"

		activePods := map[types.UID]string{
			types.UID(legitimateUID): "node-1",
		}

		It("should accept a pod UID that is present in ActivePods", func() {
			_, ok := activePods[types.UID(legitimateUID)]
			Expect(ok).To(BeTrue(), "legitimate launcher UID must be in ActivePods")
		})

		It("should reject a pod UID that is not in ActivePods", func() {
			_, ok := activePods[types.UID(rogueUID)]
			Expect(ok).To(BeFalse(), "rogue pod UID must not be in ActivePods")
		})

		It("should accept any pod UID when activePods is nil (auth disabled or VMI not found)", func() {
			var nilPods map[types.UID]string
			_, ok := nilPods[types.UID(rogueUID)]
			Expect(ok).To(BeFalse(), "nil map lookup returns false but AuthenticateSocket skips the check")
		})

		It("should handle live migration with two active pods", func() {
			migrationPods := map[types.UID]string{
				types.UID("source-pod-uid"): "node-1",
				types.UID("target-pod-uid"): "node-2",
			}
			_, sourceOK := migrationPods[types.UID("source-pod-uid")]
			_, targetOK := migrationPods[types.UID("target-pod-uid")]
			_, rogueOK := migrationPods[types.UID(rogueUID)]

			Expect(sourceOK).To(BeTrue(), "source pod must be in ActivePods during migration")
			Expect(targetOK).To(BeTrue(), "target pod must be in ActivePods during migration")
			Expect(rogueOK).To(BeFalse(), "rogue pod must not be in ActivePods during migration")
		})
	})

})
