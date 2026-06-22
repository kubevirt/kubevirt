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

package stats

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CgroupMemoryStatReader", func() {
	It("should parse all memory.stat fields correctly", func() {
		content := "anon 1073741824\nfile 536870912\nanon_thp 1006632960\ninactive_anon 805306368\nactive_anon 268435456\n"
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "memory.stat")
		Expect(os.WriteFile(path, []byte(content), 0644)).To(Succeed())

		result, err := NewCgroupMemoryStatReaderWithPath(path).Read()
		Expect(err).ToNot(HaveOccurred())

		Expect(result.AnonSet).To(BeTrue())
		Expect(result.Anon).To(Equal(uint64(1073741824)))

		Expect(result.AnonTHPSet).To(BeTrue())
		Expect(result.AnonTHP).To(Equal(uint64(1006632960)))

		Expect(result.InactiveAnonSet).To(BeTrue())
		Expect(result.InactiveAnon).To(Equal(uint64(805306368)))

		Expect(result.ActiveAnonSet).To(BeTrue())
		Expect(result.ActiveAnon).To(Equal(uint64(268435456)))
	})

	It("should return an error for a missing file", func() {
		_, err := NewCgroupMemoryStatReaderWithPath("/nonexistent").Read()
		Expect(err).To(HaveOccurred())
	})
})
