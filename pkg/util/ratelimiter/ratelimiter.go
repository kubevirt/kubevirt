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

package ratelimiter

import (
	"context"
	"sync"

	"k8s.io/client-go/util/flowcontrol"
)

type ReloadableRateLimiter struct {
	lock        *sync.Mutex
	rateLimiter flowcontrol.RateLimiter
}

func (r *ReloadableRateLimiter) TryAccept() bool {
	return r.get().TryAccept()
}

func (r *ReloadableRateLimiter) Accept() {
	r.get().Accept()
}

func (r *ReloadableRateLimiter) Stop() {
	r.get().Stop()
}

func (r *ReloadableRateLimiter) QPS() float32 {
	return r.get().QPS()
}

func (r *ReloadableRateLimiter) Wait(ctx context.Context) error {
	return r.get().Wait(ctx)
}

func (r *ReloadableRateLimiter) get() flowcontrol.RateLimiter {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.rateLimiter
}

func (r *ReloadableRateLimiter) Set(limiter flowcontrol.RateLimiter) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.rateLimiter = limiter
}

func NewReloadableRateLimiter(limiter flowcontrol.RateLimiter) *ReloadableRateLimiter {
	return &ReloadableRateLimiter{
		lock:        &sync.Mutex{},
		rateLimiter: limiter,
	}
}
