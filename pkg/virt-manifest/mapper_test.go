/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virt_manifest

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Mapper", func() {
	Context("Map PersistentVolumeClaims", func() {
		var dom *v1.DomainSpec
		BeforeEach(func() {
			dom = new(v1.DomainSpec)
			dom.Devices.Disks = []v1.Disk{
				v1.Disk{Type: Type_Network,
					Source: v1.DiskSource{Name: "network"}},
				v1.Disk{Type: Type_PersistentVolumeClaim,
					Source: v1.DiskSource{Name: "pvc"}},
			}
		})

		It("Should extract PVCs from disks", func() {
			domCopy, pvcs := ExtractPvc(dom)
			Expect(len(domCopy.Devices.Disks)).To(Equal(2))
			Expect(domCopy.Devices.Disks[0].Type).To(Equal(Type_Network))

			Expect(len(pvcs)).To(Equal(1))
			Expect(pvcs[0].disk.Type).To(Equal(Type_PersistentVolumeClaim))

		})
	})

})

func TestMapper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mapper")
}
