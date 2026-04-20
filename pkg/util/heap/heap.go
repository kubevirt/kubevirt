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

import "cmp"

type LessFunc[T any] func(left, right T) bool

// Heap implements a generic binary heap.
//
// less defines higher priority elements. For a min-heap use left < right.
// For a max-heap use left > right.
type Heap[T any] struct {
	items []T
	less  LessFunc[T]
}

func New[T any](less LessFunc[T]) *Heap[T] {
	return &Heap[T]{less: less}
}

func NewWithItems[T any](less LessFunc[T], items []T) *Heap[T] {
	copiedItems := append([]T(nil), items...)
	h := &Heap[T]{
		items: copiedItems,
		less:  less,
	}
	h.heapify()
	return h
}

func NewMin[T cmp.Ordered]() *Heap[T] {
	return New(func(left, right T) bool { return left < right })
}

func NewMax[T cmp.Ordered]() *Heap[T] {
	return New(func(left, right T) bool { return left > right })
}

func (h *Heap[T]) Len() int {
	return len(h.items)
}

func (h *Heap[T]) Reset() {
	h.items = nil
}

func (h *Heap[T]) Push(item T) {
	h.items = append(h.items, item)
	h.siftUp(len(h.items) - 1)
}

func (h *Heap[T]) Peek() (T, bool) {
	var zero T
	if len(h.items) == 0 {
		return zero, false
	}
	return h.items[0], true
}

func (h *Heap[T]) Pop() (T, bool) {
	var zero T
	if len(h.items) == 0 {
		return zero, false
	}

	lastIndex := len(h.items) - 1
	top := h.items[0]
	h.swap(0, lastIndex)
	h.items = h.items[:lastIndex]
	if len(h.items) > 0 {
		h.siftDown(0)
	}
	return top, true
}

func (h *Heap[T]) Items() []T {
	return append([]T(nil), h.items...)
}

func (h *Heap[T]) heapify() {
	lastParent := len(h.items)/2 - 1
	for parent := lastParent; parent >= 0; parent-- {
		h.siftDown(parent)
	}
}

func (h *Heap[T]) siftUp(child int) {
	for child > 0 {
		parent := (child - 1) / 2
		if !h.less(h.items[child], h.items[parent]) {
			return
		}
		h.swap(child, parent)
		child = parent
	}
}

func (h *Heap[T]) siftDown(parent int) {
	lastIndex := len(h.items) - 1
	for {
		leftChild := 2*parent + 1
		if leftChild > lastIndex {
			return
		}

		priorityChild := leftChild
		rightChild := leftChild + 1
		if rightChild <= lastIndex && h.less(h.items[rightChild], h.items[leftChild]) {
			priorityChild = rightChild
		}

		if !h.less(h.items[priorityChild], h.items[parent]) {
			return
		}

		h.swap(parent, priorityChild)
		parent = priorityChild
	}
}

func (h *Heap[T]) swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}
