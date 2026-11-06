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

package storage

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"libvirt.org/go/libvirtxml"
)

var _ = Describe("CBT migration disk size metadata", func() {
	It("round-trips disk sizes through DestXML metadata", func() {
		dom := &libvirtxml.Domain{}
		Expect(SetMigrationDiskSizes(dom, map[string]int64{
			"vol-a": 2028994560,
			"vol-b": 3221225472,
		})).To(Succeed())
		Expect(dom.Metadata).NotTo(BeNil())
		Expect(dom.Metadata.XML).To(ContainSubstring("migrationDiskSizes"))
		Expect(dom.Metadata.XML).To(ContainSubstring(`name="vol-a"`))

		size, ok := MigrationDiskSize(dom, "vol-a")
		Expect(ok).To(BeTrue())
		Expect(size).To(Equal(int64(2028994560)))
		size, ok = MigrationDiskSize(dom, "vol-b")
		Expect(ok).To(BeTrue())
		Expect(size).To(Equal(int64(3221225472)))
		_, ok = MigrationDiskSize(dom, "missing")
		Expect(ok).To(BeFalse())

		ClearMigrationDiskSizes(dom)
		Expect(dom.Metadata).To(BeNil())
		_, ok = MigrationDiskSize(dom, "vol-a")
		Expect(ok).To(BeFalse())
	})

	It("ignores empty or non-positive sizes", func() {
		dom := &libvirtxml.Domain{}
		Expect(SetMigrationDiskSizes(dom, map[string]int64{
			"":      100,
			"vol-a": 0,
			"vol-b": -1,
		})).To(Succeed())
		Expect(dom.Metadata).To(BeNil())
	})
})
