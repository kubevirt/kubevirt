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

package ephemeraldiskutils

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileExists", func() {
	It("recognizes am existing file", func() {
		Expect(FileExists("/etc/passwd")).To(BeTrue())
	})
	It("recognizes non-existing file", func() {
		Expect(FileExists("no one would ever have this file")).To(BeFalse())
	})
})
var _ = Describe("RemoveFilesIfExist", func() {
	It("silently ignores non-existing file", func() {
		Expect(RemoveFilesIfExist("no one would ever have this file")).To(Succeed())
	})
	It("removes a file", func() {
		tmpfile, err := os.CreateTemp("", "file_to_remove")
		Expect(err).ToNot(HaveOccurred())
		defer tmpfile.Close()
		Expect(FileExists(tmpfile.Name())).To(BeTrue())
		Expect(RemoveFilesIfExist(tmpfile.Name())).To(Succeed())
		Expect(FileExists(tmpfile.Name())).To(BeFalse())
	})
	It("removes multiple files", func() {
		tmpfile1, err := os.CreateTemp("", "file_to_remove1")
		Expect(err).ToNot(HaveOccurred())
		defer tmpfile1.Close()
		tmpfile2, err := os.CreateTemp("", "file_to_remove2")
		Expect(err).ToNot(HaveOccurred())
		defer tmpfile2.Close()
		Expect(RemoveFilesIfExist(tmpfile1.Name(), tmpfile2.Name())).To(Succeed())
		Expect(FileExists(tmpfile1.Name())).To(BeFalse())
		Expect(FileExists(tmpfile2.Name())).To(BeFalse())
	})
})
