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
)

// OneShotCache is a cache that calculates its value only once.
// It will use the provided calculation function to get the value. Once a first call to this function results
// in a non-error value, the cache will return this value for all subsequent calls.
type OneShotCache[T any] struct {
	value     T
	isSet     bool
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

func (c *OneShotCache[T]) Get() (T, error) {
	if c.isSet {
		return c.value, nil
	}

	c.valueLock.Lock()
	defer c.valueLock.Unlock()

	value, err := c.calcValue()
	if err != nil {
		return value, err
	}

	c.value = value
	c.isSet = true

	return c.value, nil
}

func (c *OneShotCache[T]) calcValue() (T, error) {
	if c.isSet {
		return c.value, nil
	}

	value, err := c.calcFunc()
	if err != nil {
		var zeroValue T
		return zeroValue, fmt.Errorf("failed to set value: %w", err)
	}

	return value, nil
}
