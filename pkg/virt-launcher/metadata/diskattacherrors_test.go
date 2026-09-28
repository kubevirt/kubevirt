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

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("DiskAttachErrors", func() {
	var metadataCache *metadata.Cache

	BeforeEach(func() {
		metadataCache = metadata.NewCache()
	})

	It("Load returns nil when no errors are recorded", func() {
		Expect(metadataCache.DiskAttachErrors.Load()).To(BeNil())
	})

	It("Set records errors, sorted by volume name", func() {
		metadataCache.DiskAttachErrors.Set("vol-b", "error b")
		metadataCache.DiskAttachErrors.Set("vol-a", "error a")
		Expect(metadataCache.DiskAttachErrors.Load()).To(Equal([]api.DiskStatus{
			{Name: "vol-a", AttachError: "error a"},
			{Name: "vol-b", AttachError: "error b"},
		}))
	})

	It("Set notifies only when the error changes", func() {
		metadataCache.DiskAttachErrors.Set("vol-a", "error a")
		Expect(metadataCache.Listen()).To(Receive())

		metadataCache.DiskAttachErrors.Set("vol-a", "error a")
		Expect(metadataCache.Listen()).ToNot(Receive())

		metadataCache.DiskAttachErrors.Set("vol-a", "another error")
		Expect(metadataCache.Listen()).To(Receive())
	})

	It("Retain drops errors of volumes that are not kept and notifies", func() {
		metadataCache.DiskAttachErrors.Set("vol-a", "error a")
		metadataCache.DiskAttachErrors.Set("vol-b", "error b")
		metadataCache.ResetNotification()

		metadataCache.DiskAttachErrors.Retain(func(volumeName string) bool { return volumeName == "vol-b" })
		Expect(metadataCache.Listen()).To(Receive())
		Expect(metadataCache.DiskAttachErrors.Load()).To(Equal([]api.DiskStatus{
			{Name: "vol-b", AttachError: "error b"},
		}))
	})

	It("Retain does not notify when nothing is dropped", func() {
		metadataCache.DiskAttachErrors.Set("vol-a", "error a")
		metadataCache.ResetNotification()

		metadataCache.DiskAttachErrors.Retain(func(string) bool { return true })
		Expect(metadataCache.Listen()).ToNot(Receive())
	})
})
