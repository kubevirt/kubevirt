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

package vsock

import (
	"sync"
)

type RefCounter[K comparable, T any] struct {
	lock sync.Mutex
	refs map[K]*refEntry[T]
}

type refEntry[T any] struct {
	entryRefCount int

	valueLock     sync.Mutex
	valueRefCount int
	value         *T
	destroyFn     func()
}

func NewRefCounter[K comparable, T any]() *RefCounter[K, T] {
	return &RefCounter[K, T]{
		refs: make(map[K]*refEntry[T]),
	}
}

func (r *RefCounter[K, T]) Get(key K, createFn func() (T, func(), error)) (value T, release func(), err error) {
	entry := r.getOrCreateEntry(key)

	entry.valueLock.Lock()
	if entry.value == nil {
		val, destroyFn, err := createFn()
		if err != nil {
			entry.valueLock.Unlock()
			r.removeEntry(key, entry)
			var zero T
			return zero, nil, err
		}
		entry.value = &val
		entry.destroyFn = destroyFn
	}
	entry.valueRefCount++
	entry.valueLock.Unlock()

	var once sync.Once
	release = func() {
		once.Do(func() {
			// Wrapping in func in case entry.destroyFn panics.
			func() {
				entry.valueLock.Lock()
				defer entry.valueLock.Unlock()

				entry.valueRefCount--
				if entry.valueRefCount > 0 {
					return
				}

				if entry.destroyFn != nil {
					entry.destroyFn()
				}
				entry.value = nil
				entry.destroyFn = nil
			}()

			r.removeEntry(key, entry)
		})
	}
	return *entry.value, release, nil
}

func (r *RefCounter[K, T]) getOrCreateEntry(key K) *refEntry[T] {
	r.lock.Lock()
	defer r.lock.Unlock()
	entry, exists := r.refs[key]
	if !exists {
		entry = &refEntry[T]{}
		r.refs[key] = entry
	}
	entry.entryRefCount++

	return entry
}

func (r *RefCounter[K, T]) removeEntry(key K, entry *refEntry[T]) {
	r.lock.Lock()
	defer r.lock.Unlock()
	entry.entryRefCount--
	if entry.entryRefCount == 0 {
		delete(r.refs, key)
	}
}
