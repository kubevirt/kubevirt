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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Domain Watcher", func() {
	Context("listSockets ", func() {
		It("should return socket list from ghost record cache", func() {
			const podUID = "5678"
			const socketPath = "/path/to/domainsock"

			ghostCacheDir := GinkgoT().TempDir()

			Expect(InitializeGhostRecordCache(ghostCacheDir)).To(Succeed())

			err := AddGhostRecord("test-ns", "test-domain", socketPath, podUID)
			Expect(err).ToNot(HaveOccurred())

			socketFiles, err := listSockets(getGhostRecords())
			Expect(err).ToNot(HaveOccurred())
			Expect(socketFiles).To(HaveLen(1))
			Expect(socketFiles[0]).To(Equal(socketPath))

		})
	})

	Context("listAllKnownDomains", func() {
		BeforeEach(func() {
			Expect(InitializeGhostRecordCache(GinkgoT().TempDir())).To(Succeed())
		})

		It("should return domain with Unknown status when socket exists but connection fails", func() {
			socketDir := GinkgoT().TempDir()
			socketPath := filepath.Join(socketDir, "cmd.sock")

			err := os.WriteFile(socketPath, []byte{}, 0600)
			Expect(err).ToNot(HaveOccurred())

			err = AddGhostRecord("test-ns", "test-vmi", socketPath, "uid-1234")
			Expect(err).ToNot(HaveOccurred())

			d := &domainWatcher{}
			domains, err := d.listAllKnownDomains()
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

			err := AddGhostRecord("test-ns", "test-vmi", socketPath, "uid-1234")
			Expect(err).ToNot(HaveOccurred())

			d := &domainWatcher{}
			domains, err := d.listAllKnownDomains()
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
			err = AddGhostRecord("ns1", "unreachable-vmi", unreachablePath, "uid-1")
			Expect(err).ToNot(HaveOccurred())

			missingPath := filepath.Join(socketDir, "missing.sock")
			err = AddGhostRecord("ns2", "missing-vmi", missingPath, "uid-2")
			Expect(err).ToNot(HaveOccurred())

			d := &domainWatcher{}
			domains, err := d.listAllKnownDomains()
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
