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

package seccomp

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Seccomp", func() {

	Context("Install", func() {

		var path string

		BeforeEach(func() {
			var err error
			path, err = os.MkdirTemp("", "seccomp")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(path)).To(Succeed())
		})

		It("Should install", func() {
			err := InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(path, "seccomp", "kubevirt", "kubevirt.json"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not install if equal", func() {
			err := InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())

			fInfo, err := os.Stat(filepath.Join(path, "seccomp", "kubevirt", "kubevirt.json"))
			Expect(err).NotTo(HaveOccurred())
			modified := fInfo.ModTime()

			time.Sleep(10 * time.Millisecond)

			err = InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())
			fInfo, err = os.Stat(filepath.Join(path, "seccomp", "kubevirt", "kubevirt.json"))
			Expect(err).NotTo(HaveOccurred())

			Expect(fInfo.ModTime()).To(Equal(modified))
		})

		It("Should reinstall", func() {
			policyDir := filepath.Join(path, "seccomp", "kubevirt")
			policyPath := filepath.Join(policyDir, "kubevirt.json")
			err := os.MkdirAll(policyDir, 0700)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(policyPath, []byte{}, 0777)
			Expect(err).NotTo(HaveOccurred())

			err = InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())

			b, err := os.ReadFile(policyPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).NotTo(Equal([]byte{}))
		})
	})
})
