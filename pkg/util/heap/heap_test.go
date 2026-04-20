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

package heap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heap", func() {
	It("min heap orders ascending", func() {
		h := NewMin[int]()
		input := []int{8, 3, 5, 1, 4}
		for _, number := range input {
			h.Push(number)
		}

		var got []int
		for h.Len() > 0 {
			item, ok := h.Pop()
			Expect(ok).To(BeTrue(), "expected item while heap has elements")
			got = append(got, item)
		}

		expected := []int{1, 3, 4, 5, 8}
		Expect(got).To(Equal(expected), "unexpected order")
	})

	It("max heap orders descending", func() {
		h := NewMax[int]()
		input := []int{8, 3, 5, 1, 4}
		for _, number := range input {
			h.Push(number)
		}

		var got []int
		for h.Len() > 0 {
			item, ok := h.Pop()
			Expect(ok).To(BeTrue(), "expected item while heap has elements")
			got = append(got, item)
		}

		expected := []int{8, 5, 4, 3, 1}
		Expect(got).To(Equal(expected), "unexpected order")
	})

	It("NewWithItems heapifies input", func() {
		items := []int{7, 2, 9, 1, 6}
		h := NewWithItems(func(left, right int) bool { return left < right }, items)

		items[0] = 42

		top, ok := h.Peek()
		Expect(ok).To(BeTrue(), "expected non-empty heap")
		Expect(top).To(Equal(1), "unexpected top item")
	})

	It("CustomComparator", func() {
		type migrationProgress struct {
			remainingBytes uint64
		}

		h := New(func(left, right migrationProgress) bool {
			return left.remainingBytes < right.remainingBytes
		})

		h.Push(migrationProgress{remainingBytes: 9})
		h.Push(migrationProgress{remainingBytes: 4})
		h.Push(migrationProgress{remainingBytes: 6})

		item, ok := h.Pop()
		Expect(ok).To(BeTrue(), "expected item from non-empty heap")
		Expect(item.remainingBytes).To(Equal(uint64(4)), "unexpected pop result")
	})

	It("Pop on empty heap", func() {
		h := NewMin[int]()
		_, ok := h.Pop()
		Expect(ok).To(BeFalse(), "expected empty pop to return ok=false")
	})

	It("Peek on empty heap", func() {
		h := NewMin[int]()
		_, ok := h.Peek()
		Expect(ok).To(BeFalse(), "expected empty peek to return ok=false")
	})

	It("Reset clears heap", func() {
		h := NewMin[int]()
		h.Push(5)
		h.Push(2)
		h.Reset()
		Expect(h.Len()).To(Equal(0), "expected empty heap after reset")
	})
})
