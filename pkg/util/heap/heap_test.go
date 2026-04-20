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
	"reflect"
	"testing"
)

func TestMinHeapOrdersAscending(t *testing.T) {
	h := NewMin[int]()
	input := []int{8, 3, 5, 1, 4}
	for _, number := range input {
		h.Push(number)
	}

	var got []int
	for h.Len() > 0 {
		item, ok := h.Pop()
		if !ok {
			t.Fatalf("expected item while heap has elements")
		}
		got = append(got, item)
	}

	expected := []int{1, 3, 4, 5, 8}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected order: got %v, expected %v", got, expected)
	}
}

func TestMaxHeapOrdersDescending(t *testing.T) {
	h := NewMax[int]()
	input := []int{8, 3, 5, 1, 4}
	for _, number := range input {
		h.Push(number)
	}

	var got []int
	for h.Len() > 0 {
		item, ok := h.Pop()
		if !ok {
			t.Fatalf("expected item while heap has elements")
		}
		got = append(got, item)
	}

	expected := []int{8, 5, 4, 3, 1}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected order: got %v, expected %v", got, expected)
	}
}

func TestNewWithItemsHeapifiesInput(t *testing.T) {
	items := []int{7, 2, 9, 1, 6}
	h := NewWithItems(func(left, right int) bool { return left < right }, items)

	items[0] = 42

	top, ok := h.Peek()
	if !ok {
		t.Fatalf("expected non-empty heap")
	}
	if top != 1 {
		t.Fatalf("unexpected top item: got %d, expected %d", top, 1)
	}
}

func TestCustomComparator(t *testing.T) {
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
	if !ok {
		t.Fatalf("expected item from non-empty heap")
	}
	if item.remainingBytes != 4 {
		t.Fatalf("unexpected pop result: got %d, expected %d", item.remainingBytes, 4)
	}
}

func TestPopOnEmptyHeap(t *testing.T) {
	h := NewMin[int]()
	_, ok := h.Pop()
	if ok {
		t.Fatalf("expected empty pop to return ok=false")
	}
}

func TestPeekOnEmptyHeap(t *testing.T) {
	h := NewMin[int]()
	_, ok := h.Peek()
	if ok {
		t.Fatalf("expected empty peek to return ok=false")
	}
}

func TestResetClearsHeap(t *testing.T) {
	h := NewMin[int]()
	h.Push(5)
	h.Push(2)
	h.Reset()
	if h.Len() != 0 {
		t.Fatalf("expected empty heap after reset")
	}
}
