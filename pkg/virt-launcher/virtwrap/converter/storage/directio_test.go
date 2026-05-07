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

package storage_test

import (
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
)

var _ = Describe("direct IO checker", func() {
	var directIOChecker storage.DirectIOChecker
	var tmpDir string
	var existingFile string
	var nonExistingFile string
	var err error

	BeforeEach(func() {
		directIOChecker = storage.NewDirectIOChecker()
		tmpDir, err = os.MkdirTemp("", "direct-io-checker")
		Expect(err).ToNot(HaveOccurred())
		existingFile = filepath.Join(tmpDir, "disk.img")
		Expect(os.WriteFile(existingFile, []byte("test"), 0644)).To(Succeed())
		nonExistingFile = filepath.Join(tmpDir, "non-existing-file")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should not fail when file/device exists", func() {
		_, err = directIOChecker.CheckFile(existingFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = directIOChecker.CheckBlockDevice(existingFile)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not fail when file does not exist", func() {
		_, err := directIOChecker.CheckFile(nonExistingFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(nonExistingFile)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})

	It("should fail when device does not exist", func() {
		_, err := directIOChecker.CheckBlockDevice(nonExistingFile)
		Expect(err).To(HaveOccurred())
		_, err = os.Stat(nonExistingFile)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})

	It("should fail when the path does not exist", func() {
		nonExistingPath := "/non/existing/path/disk.img"
		_, err = directIOChecker.CheckFile(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
		_, err = directIOChecker.CheckBlockDevice(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
		_, err = os.Stat(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})
})
