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

package executor

import (
	"sync"
)

// RateLimitedExecutorPool aggregates RateLimiterExecutor's by keys,
// each key element is self-contained and have its own rate-limiter.
// Each element rate-limiter is created by the given creator.
type RateLimitedExecutorPool struct {
	pool               sync.Map
	rateLimiterCreator LimitedBackoffCreator
}

func NewRateLimitedExecutorPool(creator LimitedBackoffCreator) *RateLimitedExecutorPool {
	return &RateLimitedExecutorPool{
		pool:               sync.Map{},
		rateLimiterCreator: creator,
	}
}

// LoadOrStore returns the existing RateLimitedExecutor for the key if present.
// Otherwise, it will create new RateLimitedExecutor with a new underlying rate-limiter, store and return it.
func (c *RateLimitedExecutorPool) LoadOrStore(key interface{}) *RateLimitedExecutor {
	element, exists := c.pool.Load(key)
	if !exists {
		rateLimiter := c.rateLimiterCreator.New()
		element, _ := c.pool.LoadOrStore(key, NewRateLimitedExecutor(&rateLimiter))
		return element.(*RateLimitedExecutor)
	}
	return element.(*RateLimitedExecutor)
}

func (c *RateLimitedExecutorPool) Delete(key interface{}) {
	c.pool.Delete(key)
}
