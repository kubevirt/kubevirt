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

package archdefaulter

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Arch Defaulter", func() {

	ginkgo.DescribeTable("Should create a new ArchDefaulter for the correct architecture", func(arch string, result ArchDefaulter) {
		ac := NewArchDefaulter(arch)

		Expect(ac).To(Equal(result))
	},
		ginkgo.Entry("amd64", "amd64", defaulterAMD64{}),
		ginkgo.Entry("arm64", "arm64", defaulterARM64{}),
		ginkgo.Entry("s390x", "s390x", defaulterS390X{}),
		ginkgo.Entry("unkown", "unknown", defaulterAMD64{}),
	)
})
