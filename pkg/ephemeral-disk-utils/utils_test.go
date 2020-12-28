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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package ephemeraldiskutils

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
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
var _ = Describe("RemoveFile", func() {
	It("silently ignores non-existing file", func() {
		// Ingnoring missing files is typically a sloppy behavior. This test
		// documents it, not aproves of its usage.
		Expect(RemoveFile("no one would ever have this file")).To(BeNil())
	})
	It("removes a file", func() {
		tmpfile, err := ioutil.TempFile("", "file_to_remove")
		Expect(err).To(BeNil())
		defer tmpfile.Close()
		Expect(FileExists(tmpfile.Name())).To(BeTrue())
		Expect(RemoveFile(tmpfile.Name())).To(BeNil())
		Expect(FileExists(tmpfile.Name())).To(BeFalse())
	})
})
