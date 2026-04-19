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

package vmitrait_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/vmitrait"
)

var _ = Describe("VMI traits", func() {
	Context("IsNonRoot", func() {
		It("should return false when annotation is absent and RuntimeUser is 0", func() {
			Expect(vmitrait.IsNonRoot(&v1.VirtualMachineInstance{})).To(BeFalse())
		})

		DescribeTable("should return true", func(annotations map[string]string, runtimeUser uint64) {
			vmi := &v1.VirtualMachineInstance{}
			vmi.Annotations = annotations
			vmi.Status.RuntimeUser = runtimeUser
			Expect(vmitrait.IsNonRoot(vmi)).To(BeTrue())
		},
			Entry("when the deprecated non-root annotation is present",
				map[string]string{v1.DeprecatedNonRootVMIAnnotation: ""},
				uint64(0),
			),
			Entry("when RuntimeUser is non-zero",
				nil,
				uint64(107),
			),
			Entry("when both annotation and non-zero RuntimeUser are present",
				map[string]string{v1.DeprecatedNonRootVMIAnnotation: ""},
				uint64(107),
			),
		)
	})
})
