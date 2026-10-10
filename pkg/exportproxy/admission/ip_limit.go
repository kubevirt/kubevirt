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

package admission

import (
	"net"
	"sync"
)

// IPConcurrencyLimiter caps concurrent in-flight requests per client IP.
// It keys only on Request.RemoteAddr (see ClientIP); X-Forwarded-For is not used.
type IPConcurrencyLimiter struct {
	mu     sync.Mutex
	counts map[string]int
	max    int
}

// NewIPConcurrencyLimiter returns a limiter that allows at most max concurrent
// acquires per IP. max <= 0 falls back to MaxConcurrentRequestsPerIP.
func NewIPConcurrencyLimiter(max int) *IPConcurrencyLimiter {
	if max <= 0 {
		max = MaxConcurrentRequestsPerIP
	}
	return &IPConcurrencyLimiter{
		counts: make(map[string]int),
		max:    max,
	}
}

// TryAcquire increments the in-flight count for ip when below the limit.
func (l *IPConcurrencyLimiter) TryAcquire(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.counts[ip] >= l.max {
		return false
	}
	l.counts[ip]++
	return true
}

// Release decrements the in-flight count for ip.
func (l *IPConcurrencyLimiter) Release(ip string) {
	if ip == "" {
		ip = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.counts[ip]
	if n <= 1 {
		delete(l.counts, ip)
		return
	}
	l.counts[ip] = n - 1
}

// ClientIP returns the host portion of remoteAddr (host:port). When remoteAddr
// is already a bare address, it is returned unchanged. Forwarded headers are
// intentionally ignored so clients cannot spoof the limit key.
func ClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
