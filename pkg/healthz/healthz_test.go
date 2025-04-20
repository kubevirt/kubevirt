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
 */

package healthz

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Healthz", func() {
	Context("KubeApiHealthzVersion", func() {
		apiHealthVersion := KubeApiHealthzVersion{}
		testValue := "this is a test"

		It("Should return nil by default", func() {
			Expect(apiHealthVersion.GetVersion()).To(BeNil())
		})

		It("Should store a value", func() {
			apiHealthVersion.Update(testValue)
			Expect(apiHealthVersion.GetVersion()).To(Equal(testValue))
		})

		It("Should be clearable", func() {
			apiHealthVersion.Clear()
			Expect(apiHealthVersion.GetVersion()).To(BeNil())
		})
	})
})
