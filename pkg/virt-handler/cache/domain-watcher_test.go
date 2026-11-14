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
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	authv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	cmdauth "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/auth"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	cmdserver "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cmd-server"
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
				nil,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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
				ghostRecordStore:    ghostRecordStore,
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

	Context("handleResync", func() {
		It("should close each launcher connection before moving on", func() {
			ghostRecordStore := InitializeGhostRecordCache(
				NewIterableCheckpointManager(GinkgoT().TempDir(), GinkgoT().TempDir()),
			)
			ctrl := gomock.NewController(GinkgoT())

			var launchers []*connCountingListener
			for i := range 2 {
				name := fmt.Sprintf("domain-%d", i)
				domainManager := virtwrap.NewMockDomainManager(ctrl)
				domainManager.EXPECT().ListAllDomains().Return([]*api.Domain{api.NewMinimalDomain(name)}, nil)
				domainManager.EXPECT().GetGuestOSInfo().Return(nil)
				domainManager.EXPECT().InterfacesStatus().Return(nil)

				socketPath := filepath.Join(GinkgoT().TempDir(), "launcher.sock")
				launchers = append(launchers, runConnCountingCmdServer(socketPath, domainManager))
				Expect(ghostRecordStore.Add("test-ns", name, socketPath, types.UID(fmt.Sprintf("uid-%d", i)))).To(Succeed())
			}
			accepted := func() (n int32) {
				for _, l := range launchers {
					n += l.accepted.Load()
				}
				return n
			}
			open := func() (n int32) {
				for _, l := range launchers {
					n += l.open.Load()
				}
				return n
			}

			// Nobody reads the unbuffered result channel until we do, so
			// handleResync waits on every send like it does when the
			// informer is slow.
			d := &domainWatcher{result: make(chan watch.Event), ghostRecordStore: ghostRecordStore}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				d.handleResync(ctx)
			}()

			for i := range 2 {
				Eventually(accepted).Should(BeEquivalentTo(i + 1))
				Eventually(open).Should(BeZero(), "launcher connections still open while sending event %d", i+1)
				Eventually(d.result).Should(Receive())
			}
			Eventually(done).Should(BeClosed())
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

	Context("AuthenticateSocket", func() {
		const launcherUID = "launcher-pod-uid-123"
		const rogueUID = "rogue-pod-uid-456"

		var (
			ctx        context.Context
			cancel     context.CancelFunc
			socketPath string
			activePods map[types.UID]string
		)

		BeforeEach(func() {
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
			socketPath = launcherSocketPath(launcherUID)
			activePods = map[types.UID]string{types.UID(launcherUID): "node-1"}
		})

		AfterEach(func() {
			cancel()
		})

		It("should authenticate a launcher that serves a valid, pod-bound token", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "valid-token"})
			client := clientReturning(authenticatedReview(launcherUID, cmdauth.Audience))

			Expect(AuthenticateSocket(ctx, client, socketPath, activePods)).To(Succeed())
		})

		It("should be a no-op when the k8s client is nil, so the feature gate stays off", func() {
			Expect(AuthenticateSocket(ctx, nil, socketPath, nil)).To(Succeed())
		})

		It("should reject a launcher whose token TokenReview does not authenticate", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "bogus-token"})
			review := authenticatedReview(launcherUID, cmdauth.Audience)
			review.Status.Authenticated = false

			err := AuthenticateSocket(ctx, clientReturning(review), socketPath, activePods)
			Expect(err).To(MatchError(ContainSubstring("not authenticated")))
		})

		It("should reject a token granted a different audience", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "wrong-audience-token"})
			client := clientReturning(authenticatedReview(launcherUID, "https://kubernetes.default.svc"))

			err := AuthenticateSocket(ctx, client, socketPath, activePods)
			Expect(err).To(MatchError(ContainSubstring("audience")))
		})

		It("should reject a token bound to a different pod than the socket path", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "other-pod-token"})
			client := clientReturning(authenticatedReview("some-other-pod-uid", cmdauth.Audience))

			err := AuthenticateSocket(ctx, client, socketPath, activePods)
			Expect(err).To(MatchError(ContainSubstring("pod UID mismatch")))
		})

		It("should reject a token that carries no pod UID", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "not-pod-bound"})
			review := authenticatedReview(launcherUID, cmdauth.Audience)
			review.Status.User.Extra = nil

			err := AuthenticateSocket(ctx, clientReturning(review), socketPath, activePods)
			Expect(err).To(MatchError(ContainSubstring("pod identity")))
		})

		It("should reject a pod that is not in the VMI's ActivePods", func() {
			rogueSocket := launcherSocketPath(rogueUID)
			serveCmdAuth(ctx, rogueSocket, &fakeAuthServer{token: "rogue-but-valid-token"})
			client := clientReturning(authenticatedReview(rogueUID, cmdauth.Audience))

			err := AuthenticateSocket(ctx, client, rogueSocket, activePods)
			Expect(err).To(MatchError(ContainSubstring("not in VMI ActivePods")))
		})

		// A nil map means the caller could not resolve the owning VMI. Skipping
		// the check there would let a pod with a valid token of its own pass.
		It("should fail closed when ActivePods is nil", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "valid-token"})
			client := clientReturning(authenticatedReview(launcherUID, cmdauth.Audience))

			err := AuthenticateSocket(ctx, client, socketPath, nil)
			Expect(err).To(MatchError(ContainSubstring("no ActivePods")))
		})

		It("should fail closed when ActivePods is empty", func() {
			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "valid-token"})
			client := clientReturning(authenticatedReview(launcherUID, cmdauth.Audience))

			err := AuthenticateSocket(ctx, client, socketPath, map[types.UID]string{})
			Expect(err).To(MatchError(ContainSubstring("no ActivePods")))
		})

		// There is no fallback for launchers predating CmdAuth: an
		// authentication check a peer can decline is not a check.
		It("should reject a peer that does not serve CmdAuth, even if it is in ActivePods", func() {
			serveNothing(ctx, socketPath)
			client := clientReturning(authenticatedReview(launcherUID, cmdauth.Audience))

			err := AuthenticateSocket(ctx, client, socketPath, activePods)
			Expect(err).To(MatchError(ContainSubstring("Authenticate RPC")))
		})

		It("should authenticate both launchers during a live migration", func() {
			const targetUID = "target-pod-uid"
			migrationPods := map[types.UID]string{
				types.UID(launcherUID): "node-1",
				types.UID(targetUID):   "node-2",
			}

			serveCmdAuth(ctx, socketPath, &fakeAuthServer{token: "source-token"})
			Expect(AuthenticateSocket(ctx, clientReturning(authenticatedReview(launcherUID, cmdauth.Audience)),
				socketPath, migrationPods)).To(Succeed())

			targetSocket := launcherSocketPath(targetUID)
			serveCmdAuth(ctx, targetSocket, &fakeAuthServer{token: "target-token"})
			Expect(AuthenticateSocket(ctx, clientReturning(authenticatedReview(targetUID, cmdauth.Audience)),
				targetSocket, migrationPods)).To(Succeed())
		})
	})

})

// connCountingListener counts the connections it accepted and how many of
// them are still open.
type connCountingListener struct {
	net.Listener
	accepted atomic.Int32
	open     atomic.Int32
}

func (l *connCountingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.accepted.Add(1)
	l.open.Add(1)
	return &countedConn{Conn: conn, open: &l.open}, nil
}

type countedConn struct {
	net.Conn
	open      *atomic.Int32
	closeOnce sync.Once
}

func (c *countedConn) Close() error {
	c.closeOnce.Do(func() { c.open.Add(-1) })
	return c.Conn.Close()
}

func runConnCountingCmdServer(socketPath string, domainManager virtwrap.DomainManager) *connCountingListener {
	listener, err := net.Listen("unix", socketPath)
	Expect(err).ToNot(HaveOccurred())
	counting := &connCountingListener{Listener: listener}

	server := grpc.NewServer()
	cmdserver.RegisterCmdServer(cmdserver.NewLauncher(domainManager, nil))(server)
	go func() { _ = server.Serve(counting) }()
	DeferCleanup(server.Stop)
	return counting
}

// fakeAuthServer stands in for the CmdAuth service on virt-launcher.
type fakeAuthServer struct {
	token string
	err   error
}

func (f *fakeAuthServer) Authenticate(_ context.Context, _ *cmdauth.AuthRequest) (*cmdauth.AuthResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &cmdauth.AuthResponse{Token: f.token}, nil
}

// launcherSocketPath builds a path shaped like a real launcher socket, so
// that podUIDFromSocketPath can recover the pod UID from it. The temp dir is
// kept short because unix socket paths are limited to ~107 bytes.
func launcherSocketPath(podUID string) string {
	base, err := os.MkdirTemp("", "ls")
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(func() { os.RemoveAll(base) })

	dir := filepath.Join(base, podUID, "volumes", "sockets")
	Expect(os.MkdirAll(dir, 0755)).To(Succeed())
	return filepath.Join(dir, "launcher-sock")
}

// serveCmdAuth starts a gRPC server on socketPath with the CmdAuth service
// registered. serveNothing starts one without it, which is how a launcher
// predating this feature behaves.
func serveCmdAuth(ctx context.Context, socketPath string, impl cmdauth.CmdAuthServer) {
	server := grpc.NewServer()
	cmdauth.RegisterCmdAuthServer(server, impl)
	startGRPC(ctx, server, socketPath)
}

func serveNothing(ctx context.Context, socketPath string) {
	startGRPC(ctx, grpc.NewServer(), socketPath)
}

func startGRPC(ctx context.Context, server *grpc.Server, socketPath string) {
	listener, err := net.Listen("unix", socketPath)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		_ = server.Serve(listener)
	}()
	go func() {
		<-ctx.Done()
		server.Stop()
	}()
	DeferCleanup(server.Stop)
}

// authenticatedReview is the TokenReview response the API server returns for
// a valid, pod-bound projected ServiceAccount token.
func authenticatedReview(podUID string, audiences ...string) *authv1.TokenReview {
	return &authv1.TokenReview{
		Status: authv1.TokenReviewStatus{
			Authenticated: true,
			Audiences:     audiences,
			User: authv1.UserInfo{
				Username: "system:serviceaccount:default:default",
				Extra: map[string]authv1.ExtraValue{
					tokenReviewPodUIDKey: {podUID},
				},
			},
		},
	}
}

func clientReturning(review *authv1.TokenReview) kubernetes.Interface {
	client := fake.NewSimpleClientset()
	client.Fake.PrependReactor("create", "tokenreviews",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, review, nil
		})
	return client
}
