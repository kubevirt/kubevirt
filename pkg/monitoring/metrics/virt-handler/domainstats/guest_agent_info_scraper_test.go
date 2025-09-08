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
 */

package domainstats

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("GuestAgentInfoScraper", func() {
	var (
		scraper    *GuestAgentInfoScraper
		socketFile string
		guestInfo  *k6tv1.VirtualMachineInstanceGuestAgentInfo
	)

	BeforeEach(func() {
		scraper = NewGuestAgentInfoScraper()
		socketFile = "/tmp/test-socket"

		guestInfo = &k6tv1.VirtualMachineInstanceGuestAgentInfo{
			Hostname: "test-hostname",
			Load: &k6tv1.VirtualMachineInstanceGuestOSLoad{
				Load1mSet:  true,
				Load1m:     1.5,
				Load5mSet:  true,
				Load5m:     2.0,
				Load15mSet: true,
				Load15m:    2.5,
			},
		}
	})

	Context("Caching mechanism", func() {
		It("should initialize with empty cache", func() {
			By("verifying cache is empty")
			Expect(scraper.cache).To(BeEmpty())
		})

		It("should store guest info in cache", func() {
			By("adding guest info to cache")
			scraper.addCacheEntry(socketFile, time.Now(), guestInfo)

			By("verifying cache is populated")
			cached, _, exists := scraper.getCacheEntry(socketFile)

			Expect(exists).To(BeTrue())
			Expect(cached.GuestAgentInfo).To(Equal(guestInfo))
		})

		It("should skip empty guest info", func() {
			By("adding empty guest info to cache")
			scraper.addCacheEntry(socketFile, time.Now(), nil)

			By("verifying cache remains empty")
			_, _, exists := scraper.getCacheEntry(socketFile)
			Expect(exists).To(BeFalse())
		})

		It("should skip guest info with empty hostname", func() {
			By("adding guest info with empty hostname to cache")
			emptyGuestInfo := &k6tv1.VirtualMachineInstanceGuestAgentInfo{}
			scraper.addCacheEntry(socketFile, time.Now(), emptyGuestInfo)

			By("verifying cache remains empty")
			_, _, exists := scraper.getCacheEntry(socketFile)
			Expect(exists).To(BeFalse())
		})

		It("should identify cache entries within timeout", func() {
			By("creating a fresh cache entry")
			scraper.addCacheEntry(socketFile, time.Now(), guestInfo)

			By("checking if entry is within timeout")
			_, timestamp, exists := scraper.getCacheEntry(socketFile)
			Expect(exists).To(BeTrue())
			Expect(time.Since(timestamp)).To(BeNumerically("<", cacheTimeout))
		})

		It("should identify cache entries outside timeout", func() {
			By("creating an expired cache entry")
			scraper.addCacheEntry(socketFile, time.Now().Add(-2*cacheTimeout), guestInfo)

			By("checking if entry is outside timeout")
			_, timestamp, exists := scraper.getCacheEntry(socketFile)
			Expect(exists).To(BeFalse())
			Expect(timestamp).To(Equal(time.Time{}))
		})
	})
})
