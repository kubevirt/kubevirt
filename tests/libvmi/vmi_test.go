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
 * Copyright the KubeVirt Authors.
 *
 */

package libvmi

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("libvmi annotations", func() {
	Context("common test cases", func() {
		It("creates map with hooksidecars key", func() {
			Expect(New(WithExampleHookSideCarAndVersionAnnotation("")).Annotations).To(HaveKey(annotationKeyHookSideCars))
		})
		It("creates map with hooksidecars key", func() {
			Expect(New(WithExampleHookSideCarAndNoVersionAnnotation()).Annotations).To(HaveKey(annotationKeyHookSideCars))
		})
		It("creates map with base board manufacturer key", func() {
			Expect(New(WithBaseBoardManufacturerAnnotation()).Annotations).To(HaveKey(annotationKeyBaseBoardManufacturer))
		})
	})
})
