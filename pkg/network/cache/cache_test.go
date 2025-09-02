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

package cache_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
)

var _ = Describe("cache", func() {
	var cacheCreator tempCacheCreator
	var cache *cache.Cache

	type data struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	var testData data

	BeforeEach(func() {
		cache = cacheCreator.New("/this/is/a/test/cache")
		Expect(cache).NotTo(BeNil())
		dutils.MockDefaultOwnershipManager()

		testData = data{
			Key:   "mykey",
			Value: "myvalue",
		}
	})

	AfterEach(func() { Expect(cache.Delete()).To(Succeed()) })

	It("should return os.ErrNotExist if no cache entry exists", func() {
		var newData data
		_, err := cache.Read(&newData)
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	It("should save and restore data", func() {
		Expect(cache.Write(testData)).To(Succeed())

		var newData data
		Expect(cache.Read(&newData)).To(Equal(&testData))
	})

	It("should remove the cache file", func() {
		Expect(cache.Write(testData)).To(Succeed())

		var newData data
		_, err := cache.Read(&newData)
		Expect(err).NotTo(HaveOccurred())

		Expect(cache.Delete()).To(Succeed())

		_, err = cache.Read(&newData)
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	It("should save, restore and remove a cache entry (sub-cache)", func() {
		const subcacheName = "subcache"
		subCache, err := cache.Entry(subcacheName)
		Expect(err).NotTo(HaveOccurred())

		var newData data
		_, err = subCache.Read(&newData)
		Expect(err).To(MatchError(os.ErrNotExist))

		Expect(subCache.Write(testData)).To(Succeed())
		Expect(subCache.Read(&newData)).To(Equal(&testData))

		Expect(subCache.Delete()).To(Succeed())
		_, err = subCache.Read(&newData)
		Expect(err).To(MatchError(os.ErrNotExist))
	})

	It("should not create an entry to a cache that has an existing backend store (e.g. data file)", func() {
		Expect(cache.Write(testData)).To(Succeed())

		const subcacheName = "subcache"
		_, err := cache.Entry(subcacheName)
		Expect(err).To(MatchError("unable to define entry: parent cache has an existing store"))
	})

	It("should not be able to write to a cache that has child entries", func() {
		const subcacheName = "subcache"
		subCache, err := cache.Entry(subcacheName)
		Expect(err).NotTo(HaveOccurred())
		Expect(subCache.Write(testData)).To(Succeed())

		err = cache.Write(testData)
		Expect(err).NotTo(Succeed())
		Expect(err.Error()).To(HaveSuffix("is a directory"))
	})
})
