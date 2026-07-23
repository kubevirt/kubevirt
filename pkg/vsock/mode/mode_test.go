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

package mode_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/vsock/mode"
)

var _ = Describe("VSOCK mode", func() {
	var (
		mockProcDir string
		sysDir      string
	)

	BeforeEach(func() {
		mockProcDir = GinkgoT().TempDir()
		sysDir = filepath.Join(mockProcDir, "sys", "net", "vsock")
		Expect(os.MkdirAll(sysDir, 0o755)).To(Succeed())
	})

	const (
		childNsMode = "child_ns_mode"
		nsMode      = "ns_mode"
	)

	VsockModeTestBlock := func(text string, testFn func(string) string, sysctl string) {
		Context(text, func() {
			DescribeTable("should return mode",
				func(fileContent, expectedMode string) {
					filePath := filepath.Join(sysDir, sysctl)
					Expect(os.WriteFile(filePath, []byte(fileContent), 0o600)).To(Succeed())

					Expect(testFn(mockProcDir)).To(Equal(expectedMode))
				},
				Entry("'local' when sysctl is 'local'", "local\n", mode.ModeLocal),
				Entry("'global' when sysctl is 'global'", "global\n", mode.ModeGlobal),
				Entry("'global', when sysctl contains any other value", "unknown\n", mode.ModeGlobal),
			)

			It("should return global mode when sysctl does not exist", func() {
				Expect(testFn(mockProcDir)).To(Equal(mode.ModeGlobal))
			})
		})
	}

	VsockModeTestBlock("VsockChildNsMode", mode.VsockChildNsMode, childNsMode)

	VsockModeTestBlock("VsockNsMode", mode.VsockNsMode, nsMode)
})
