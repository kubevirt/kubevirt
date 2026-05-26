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

package vsock

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("readVSOCKMode", func() {
	DescribeTable("should return correct mode",
		func(fileContent, expectedMode string) {
			tempDir := GinkgoT().TempDir()
			testFile := filepath.Join(tempDir, "test_mode")
			Expect(os.WriteFile(testFile, []byte(fileContent), 0o600)).To(Succeed())

			Expect(readVSOCKMode(testFile)).To(Equal(expectedMode))
		},
		Entry("when file contains 'local'", "local", ModeLocal),
		Entry("when file contains 'local' with whitespace", "  local\n", ModeLocal),
		Entry("when file contains 'global'", "global", ModeGlobal),
		Entry("when file contains any other value", "unknown", ModeGlobal),
		Entry("when file is empty", "", ModeGlobal),
	)

	It("should return global mode when file does not exit", func() {
		tempDir := GinkgoT().TempDir()
		testFile := filepath.Join(tempDir, "test_mode")

		Expect(readVSOCKMode(testFile)).To(Equal(ModeGlobal))
	})
})
