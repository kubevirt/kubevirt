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

package virtwrap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Sliding window minimum", func() {
	It("has no minimum before the first sample", func() {
		var window slidingWindowMinimum

		_, ok := window.minimum()
		Expect(ok).To(BeFalse())
	})

	It("promotes the next smallest value when the minimum reaches the window boundary", func() {
		window := slidingWindowMinimum{durationSeconds: 25}
		window.add(5_000, 100)
		window.add(10_000, 120)
		window.add(29_999, 110)

		minimum, ok := window.minimum()
		Expect(ok).To(BeTrue())
		Expect(minimum).To(Equal(minimumSample{elapsedMs: 5_000, value: 100}))

		window.add(30_000, 130)

		minimum, ok = window.minimum()
		Expect(ok).To(BeTrue())
		Expect(minimum).To(Equal(minimumSample{elapsedMs: 29_999, value: 110}))
	})

	It("keeps the newer sample when values are equal", func() {
		window := slidingWindowMinimum{durationSeconds: 25}
		window.add(5_000, 100)
		window.add(10_000, 100)

		minimum, ok := window.minimum()
		Expect(ok).To(BeTrue())
		Expect(minimum).To(Equal(minimumSample{elapsedMs: 10_000, value: 100}))
	})

	It("keeps only the current sample when the window is zero", func() {
		var window slidingWindowMinimum
		window.add(5_000, 100)
		window.add(5_000, 120)

		minimum, ok := window.minimum()
		Expect(ok).To(BeTrue())
		Expect(minimum).To(Equal(minimumSample{elapsedMs: 5_000, value: 120}))
	})
})
