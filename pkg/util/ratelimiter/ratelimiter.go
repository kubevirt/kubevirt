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
