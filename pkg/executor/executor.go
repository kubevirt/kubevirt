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

type rateLimiter interface {
	Step()
	Ready() bool
}

// RateLimitedExecutor provides self-contained entity that enables rate-limiting a given func
// execution (e.g: with an exponential backoff) without blocking the goroutine it runs on.
type RateLimitedExecutor struct {
	rateLimiter rateLimiter
}

func NewRateLimitedExecutor(rateLimiter rateLimiter) *RateLimitedExecutor {
	return &RateLimitedExecutor{
		rateLimiter: rateLimiter,
	}
}

// Exec will execute the given func when the underlying rate-limiter
// is not blocking; rate-limiter's end time is passed and limit is not reached.
func (c *RateLimitedExecutor) Exec(command func() error) error {
	if !c.rateLimiter.Ready() {
		return nil
	}
	defer c.rateLimiter.Step()

	return command()
}
