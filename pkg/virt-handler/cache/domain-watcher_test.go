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
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
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
			)

			Eventually(d.result).Should(BeClosed())

			Expect(func() { d.Stop() }).ShouldNot(Panic())
			Expect(func() { d.Stop() }).ShouldNot(Panic())
		})
	})

	Context("listAllKnownDomains", func() {
		var ghostCacheDir string

		BeforeEach(func() {
			ghostCacheDir = GinkgoT().TempDir()
			InitializeGhostRecordCache(NewIterableCheckpointManager(ghostCacheDir))
		})

		It("should return domain with Unknown status when socket exists but connection fails", func() {
			socketDir := GinkgoT().TempDir()
			socketPath := filepath.Join(socketDir, "cmd.sock")

			err := os.WriteFile(socketPath, []byte{}, 0600)
			Expect(err).ToNot(HaveOccurred())

			err = GhostRecordGlobalStore.Add("test-ns", "test-vmi", socketPath, "uid-1234")
			Expect(err).ToNot(HaveOccurred())

			domains, err := listAllKnownDomains()
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(HaveLen(1))
			Expect(domains[0].ObjectMeta.Namespace).To(Equal("test-ns"))
			Expect(domains[0].ObjectMeta.Name).To(Equal("test-vmi"))
			Expect(domains[0].ObjectMeta.UID).To(BeEquivalentTo("uid-1234"))
			Expect(domains[0].Status.Status).To(Equal(api.Unknown))
			Expect(domains[0].ObjectMeta.DeletionTimestamp).To(BeNil())
		})

		It("should return domain with DeletionTimestamp when socket file does not exist", func() {
			socketPath := "/nonexistent/path/cmd.sock"

			err := GhostRecordGlobalStore.Add("test-ns", "test-vmi", socketPath, "uid-1234")
			Expect(err).ToNot(HaveOccurred())

			domains, err := listAllKnownDomains()
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(HaveLen(1))
			Expect(domains[0].ObjectMeta.Namespace).To(Equal("test-ns"))
			Expect(domains[0].ObjectMeta.Name).To(Equal("test-vmi"))
			Expect(domains[0].ObjectMeta.DeletionTimestamp).ToNot(BeNil())
		})

		It("should handle mix of reachable, unreachable, and missing sockets", func() {
			socketDir := GinkgoT().TempDir()

			unreachablePath := filepath.Join(socketDir, "unreachable.sock")
			err := os.WriteFile(unreachablePath, []byte{}, 0600)
			Expect(err).ToNot(HaveOccurred())
			err = GhostRecordGlobalStore.Add("ns1", "unreachable-vmi", unreachablePath, "uid-1")
			Expect(err).ToNot(HaveOccurred())

			missingPath := filepath.Join(socketDir, "missing.sock")
			err = GhostRecordGlobalStore.Add("ns2", "missing-vmi", missingPath, "uid-2")
			Expect(err).ToNot(HaveOccurred())

			domains, err := listAllKnownDomains()
			Expect(err).ToNot(HaveOccurred())
			Expect(domains).To(HaveLen(2))

			var unknownDomain, deletedDomain *api.Domain
			for _, d := range domains {
				if d.Status.Status == api.Unknown {
					unknownDomain = d
				}
				if d.ObjectMeta.DeletionTimestamp != nil {
					deletedDomain = d
				}
			}

			Expect(unknownDomain).ToNot(BeNil())
			Expect(unknownDomain.ObjectMeta.Name).To(Equal("unreachable-vmi"))

			Expect(deletedDomain).ToNot(BeNil())
			Expect(deletedDomain.ObjectMeta.Name).To(Equal("missing-vmi"))
		})
	})
})
