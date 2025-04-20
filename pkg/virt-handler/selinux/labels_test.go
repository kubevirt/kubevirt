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

package selinux

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("selinux", func() {

	var tempDir string
	var selinux *SELinuxImpl

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(tempDir, "/usr/sbin"), 0777)).ToNot(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(tempDir, "/usr/bin"), 0777)).ToNot(HaveOccurred())
		selinux = &SELinuxImpl{
			Paths:         []string{"/usr/bin", "/usr/sbin"},
			procOnePrefix: tempDir,
		}
	})

	Context("detecting if selinux is present", func() {
		It("should detect that it is disabled if getenforce returns Disabled", func() {
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("disabled"), nil
			}
			present, mode, err := selinux.IsPresent()
			Expect(err).ToNot(HaveOccurred())
			Expect(present).To(BeFalse())
			Expect(mode).To(Equal("disabled"))
		})
		It("should detect that it is enabled if getenforce returns Permissive", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "getenforce"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("Permissive"), nil
			}
			present, _, err := selinux.IsPresent()
			Expect(err).ToNot(HaveOccurred())
			Expect(present).To(BeTrue())
		})
		It("should detect that it is enabled if getenforce does not return Disabled", func() {
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("enforcing"), nil
			}
			present, mode, err := selinux.IsPresent()
			Expect(err).ToNot(HaveOccurred())
			Expect(present).To(BeTrue())
			Expect(mode).To(Equal("enforcing"))
		})
	})
})

func touch(path string) {
	f, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	Expect(f.Close()).To(Succeed())
}
