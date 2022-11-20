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
 * Copyright 2022 Red Hat, Inc.
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
})
