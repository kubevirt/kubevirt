/* Licensed under the Apache License, Version 2.0 (the "License");
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
 * Copyright the KubeVirt Authors.
 *
 */

package converter

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Arch Converter", func() {

	DescribeTable("Should create a new archConverter for the correct architecture", func(arch string, result archConverter) {
		ac := newArchConverter(arch)

		Expect(ac).To(Equal(result))
	},
		Entry("amd64", "amd64", archConverterAMD64{}),
		Entry("arm64", "arm64", archConverterARM64{}),
		Entry("ppc64le", "ppc64le", archConverterPPC64{}),
		Entry("unkown", "unknown", archConverterAMD64{}),
	)
})
