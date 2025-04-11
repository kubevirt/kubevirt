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

package metadata_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
)

var _ = Describe("Metadata", func() {
	var metadataCache *metadata.Cache

	BeforeEach(func() {
		metadataCache = metadata.NewCache()
	})

	It("Load uninitialized data", func() {
		data, exists := metadataCache.Migration.Load()
		Expect(exists).To(BeFalse())
		Expect(data).To(Equal(api.MigrationMetadata{}))
	})

	It("Store and load data", func() {
		const test123 = "test123"
		origData := api.MigrationMetadata{FailureReason: test123}

		metadataCache.Migration.Store(origData)

		newData, exists := metadataCache.Migration.Load()
		Expect(exists).To(BeTrue())
		Expect(newData).To(Equal(origData))
	})

	It("Mutate uninitialized data in a safe block", func() {
		const test123 = "test123"

		metadataCache.Migration.WithSafeBlock(func(m *api.MigrationMetadata, initialized bool) {
			Expect(initialized).To(BeFalse())
			Expect(*m).To(Equal(api.MigrationMetadata{}))

			m.FailureReason = test123
		})
		newData, exists := metadataCache.Migration.Load()
		Expect(exists).To(BeTrue())
		Expect(newData).To(Equal(api.MigrationMetadata{FailureReason: test123}))
	})

	It("Mutate existing data in a safe block", func() {
		const test123 = "test123"

		origData := api.MigrationMetadata{FailureReason: "origin-data"}
		metadataCache.Migration.Store(origData)

		metadataCache.Migration.WithSafeBlock(func(m *api.MigrationMetadata, initialized bool) {
			Expect(initialized).To(BeTrue())
			Expect(*m).To(Equal(origData))

			m.FailureReason = test123
		})
		newData, exists := metadataCache.Migration.Load()
		Expect(exists).To(BeTrue())
		Expect(newData).To(Equal(api.MigrationMetadata{FailureReason: test123}))
	})

	It("Notify and listen when cache data store", func() {
		const test123 = "test123"
		metadataCache.Migration.Store(api.MigrationMetadata{FailureReason: test123})

		Expect(metadataCache.Listen()).Should(Receive())
	})

	It("Do not notify when cache data set", func() {
		const test123 = "test123"
		metadataCache.Migration.Set(api.MigrationMetadata{FailureReason: test123})

		Expect(metadataCache.Listen()).ShouldNot(Receive())
	})

	It("Reset notification signal", func() {
		metadataCache.Migration.Store(api.MigrationMetadata{FailureReason: "test123"})

		metadataCache.ResetNotification()

		Expect(metadataCache.Listen()).ShouldNot(Receive())
	})

	It("Notify multiple times and listen to a single cache data change", func() {
		metadataCache.Migration.Store(api.MigrationMetadata{FailureReason: "test123"})
		metadataCache.Migration.Store(api.MigrationMetadata{FailureReason: "test456"})
		metadataCache.Migration.Store(api.MigrationMetadata{FailureReason: "test789"})

		Expect(metadataCache.Listen()).Should(Receive())
		Expect(metadataCache.Listen()).ShouldNot(Receive())
	})

	It("Notify when the data is mutated in a safe block", func() {
		metadataCache.Migration.WithSafeBlock(func(m *api.MigrationMetadata, initialized bool) {
			m.FailureReason = "test123"
		})
		Expect(metadataCache.Listen()).Should(Receive())
	})

	It("Do not notify when the data is not mutated in a safe block", func() {
		const test123 = "test123"
		metadataCache.Migration.Store(api.MigrationMetadata{FailureReason: test123})
		Expect(metadataCache.Listen()).Should(Receive())

		metadataCache.Migration.WithSafeBlock(func(m *api.MigrationMetadata, initialized bool) {
			m.FailureReason = test123
		})

		Expect(metadataCache.Listen()).ShouldNot(Receive())
	})
})
