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
 * Copyright The KubeVirt Authors
 *
 */

package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// OneShotCache is a cache that uses a provided function to calculate a value, while ensuring that the calculation
// is performed only once after the first successful call. Later calls to the `Get` method will return the
// cached value without invoking the calculation function again.
type OneShotCache[T any] struct {
	value     atomic.Pointer[T]
	calcFunc  func() (T, error)
	valueLock sync.Mutex
}

func NewOneShotCache[T any](calcFunc func() (T, error)) (*OneShotCache[T], error) {
	if calcFunc == nil {
		return nil, fmt.Errorf("calculation function is not set")
	}

	return &OneShotCache[T]{
		calcFunc:  calcFunc,
		valueLock: sync.Mutex{},
	}, nil
}

// Get retrieves the cached value, or calculates it if it has not been set yet.
// It ensures that the calculation function is called only once after the first successful call
// It supports being called concurrently from multiple goroutines.
func (c *OneShotCache[T]) Get() (T, error) {
	if valuePtr := c.value.Load(); valuePtr != nil {
		return *valuePtr, nil
	}

	c.valueLock.Lock()
	defer c.valueLock.Unlock()

	// Double-check if the value was set while we were waiting for the lock.
	if valuePtr := c.value.Load(); valuePtr != nil {
		return *valuePtr, nil
	}

	value, err := c.calcFunc()
	if err != nil {
		return value, fmt.Errorf("failed to calculate value: %w", err)
	}

	c.value.Store(&value)

	return value, nil
}

// ForceUpdate forces a spin of the calcFunc
func (c *OneShotCache[T]) ForceUpdate() error {
	c.valueLock.Lock()
	defer c.valueLock.Unlock()
	value, err := c.calcFunc()
	if err != nil {
		return fmt.Errorf("failed to calculate value: %w", err)
	}

	c.value.Store(&value)
	return nil
}
