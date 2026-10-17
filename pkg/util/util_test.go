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

package util

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Misc Capacity", func() {
	var (
		originalMiscCapacityPath string
		originalMiscMaxPath      string
		tempDir                  string
	)

	BeforeEach(func() {
		originalMiscCapacityPath, originalMiscMaxPath = miscCapacityPath, miscMaxPath
		tempDir = GinkgoT().TempDir()
		// A case that writes neither file must see neither, rather than falling
		// through to the misc cgroup of whatever host the test runs on.
		miscCapacityPath = path.Join(tempDir, "misc.capacity")
		miscMaxPath = path.Join(tempDir, "misc.max")
	})

	AfterEach(func() {
		miscCapacityPath, miscMaxPath = originalMiscCapacityPath, originalMiscMaxPath
	})

	Context("when reading secure guest capacity from misc.capacity", func() {
		It("should successfully parse TDX capacity", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("tdx 15\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["tdx"]).To(Equal(15))
		})

		It("should successfully parse SEV-SNP capacity", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("sev 410\nsev_es 99\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(2))
			Expect(caps["sev"]).To(Equal(410))
			Expect(caps["sev_es"]).To(Equal(99))
		})

		It("should successfully handle empty file", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte(""), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(BeEmpty())
		})

		It("should return error when file does not exist", func() {
			miscCapacityPath = "/nonexisted_path/misc.capacity"
			caps, err := GetMiscCapacity()
			Expect(err).To(HaveOccurred())
			Expect(caps).To(BeNil())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
		It("should skip malformed lines and parse the rest", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("tdx abc\nsev_es 99\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["sev_es"]).To(Equal(99))
		})
	})

	Context("when misc.capacity is absent and the node has only a limit", func() {
		It("should report one for a limit the kernel left unlimited", func() {
			Expect(os.WriteFile(miscMaxPath, []byte("sev max\nsev_es max\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(2))
			Expect(caps["sev"]).To(Equal(1))
			Expect(caps["sev_es"]).To(Equal(1))
		})

		It("should report one for a positive limit", func() {
			Expect(os.WriteFile(miscMaxPath, []byte("sev_es 100\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["sev_es"]).To(Equal(1))
		})

		It("should report zero when the limit is zero", func() {
			Expect(os.WriteFile(miscMaxPath, []byte("sev_es 0\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["sev_es"]).To(Equal(0))
		})

		It("should prefer the capacity where the kernel provided one", func() {
			Expect(os.WriteFile(miscCapacityPath, []byte("sev_es 99\n"), 0644)).To(Succeed())
			Expect(os.WriteFile(miscMaxPath, []byte("sev_es max\n"), 0644)).To(Succeed())
			caps, err := GetMiscCapacity()
			Expect(err).ToNot(HaveOccurred())
			Expect(caps).To(HaveLen(1))
			Expect(caps["sev_es"]).To(Equal(99))
		})

		It("should return a not exist error when neither file is present", func() {
			caps, err := GetMiscCapacity()
			Expect(err).To(HaveOccurred())
			Expect(caps).To(BeNil())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
