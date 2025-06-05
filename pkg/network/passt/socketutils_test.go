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

package passt_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/passt"
)

var _ = Describe("CreateShortenedSymlink", func() {
	var baseDir string

	BeforeEach(func() {
		var err error
		baseDir, err = os.MkdirTemp("", "symlink-test")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(baseDir)
	})

	It("should create a symlink at shortened path", func() {
		inputPath := filepath.Join(baseDir, "pods", "podUID", "volumes", "kubernetes.io~empty-dir", "libvirt-runtime", "qemu", "run", "passt")
		err := os.MkdirAll(inputPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		symlinkPath, err := passt.CreateShortenedSymlink(inputPath, baseDir+string(filepath.Separator))
		Expect(err).ToNot(HaveOccurred())

		info, err := os.Lstat(symlinkPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(info.Mode() & os.ModeSymlink).To(Equal(os.ModeSymlink))

		target, err := os.Readlink(symlinkPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(target).To(Equal(inputPath))
	})

	It("should return the same path if symlink already exists and is correct", func() {
		inputPath := filepath.Join(baseDir, "pods", "uid", "volumes", "qemu", "run", "passt")
		err := os.MkdirAll(inputPath, 0755)
		Expect(err).ToNot(HaveOccurred())

		expectedLink := filepath.Join(baseDir, "pods", "uid", "p")
		err = os.Symlink(inputPath, expectedLink)
		Expect(err).ToNot(HaveOccurred())

		resultPath, err := passt.CreateShortenedSymlink(inputPath, baseDir+string(filepath.Separator))
		Expect(err).ToNot(HaveOccurred())
		Expect(resultPath).To(Equal(expectedLink))
	})
})
