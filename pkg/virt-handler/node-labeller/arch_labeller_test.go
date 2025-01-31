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

package nodelabeller

import (
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Arch Node Labeller", func() {

	DescribeTable("Should create a new archLabeller for the correct architecture", func(arch string, result archLabeller) {
		ac := newArchLabeller(arch)

		Expect(ac).To(Equal(result))
		if arch == "unknown" {
			arch = runtime.GOARCH
		}
		Expect(ac.arch()).To(Equal(arch))
	},
		Entry(amd64, amd64, archLabellerAMD64{}),
		Entry(arm64, arm64, archLabellerARM64{}),
		Entry(s390x, s390x, archLabellerS390X{}),
		Entry("unknown", "unknown", defaultArchLabeller{}),
	)
})
