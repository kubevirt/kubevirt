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

package nodelabeller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("vGPU host driver version", func() {
	DescribeTable("should parse /sys/module/nvidia_vgpu_vfio/version", func(output, version string, found bool) {
		got, ok := parseVGPUHostDriverVersion(output)
		Expect(ok).To(Equal(found))
		Expect(got).To(Equal(version))
	},
		Entry("when the file contains a version and a trailing newline", "595.91.04\n", "595.91.04", true),
		Entry("when the file contains only the version", "550.54.15", "550.54.15", true),
		Entry("when the version is surrounded by whitespace", " \t525.60.13 \n", "525.60.13", true),
		Entry("when the file is empty", "", "", false),
		Entry("when the file contains only whitespace", " \n", "", false),
		Entry("when the file does not contain a driver version", "not-a-version", "", false),
	)
})
