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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package ratelimit

import (
	"sync"
)

// RateLimitedExecutorPool aggregates RateLimiterExecutor's by keys,
// each key element is self-contained and have its own rate-limiter.
// Each element rate-limiter is created by the given creator.
type RateLimitedExecutorPool struct {
	sync.Map
	creator LimitedBackoffCreator
}

func NewRateLimitedExecutorPool(creator LimitedBackoffCreator) *RateLimitedExecutorPool {
	return &RateLimitedExecutorPool{
		Map:     sync.Map{},
		creator: creator,
	}
}

// LoadOrStore returns the existing RateLimitedExecutor for the key if present.
// Otherwise, it will create new RateLimitedExecutor with a new underlying rate-limiter, store and return it.
func (c *RateLimitedExecutorPool) LoadOrStore(key interface{}) *RateLimitedExecutor {
	rateLimit := c.creator.New()
	element, _ := c.Map.LoadOrStore(key, NewRateLimitedExecutor(&rateLimit))
	return element.(*RateLimitedExecutor)
}
