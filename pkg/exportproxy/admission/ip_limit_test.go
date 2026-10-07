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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IPConcurrencyLimiter", func() {
	It("limits concurrent acquires per IP independently", func() {
		limiter := NewIPConcurrencyLimiter(2)

		Expect(limiter.TryAcquire("10.0.0.1")).To(BeTrue())
		Expect(limiter.TryAcquire("10.0.0.1")).To(BeTrue())
		Expect(limiter.TryAcquire("10.0.0.1")).To(BeFalse())

		Expect(limiter.TryAcquire("10.0.0.2")).To(BeTrue())

		limiter.Release("10.0.0.1")
		Expect(limiter.TryAcquire("10.0.0.1")).To(BeTrue())
		Expect(limiter.TryAcquire("10.0.0.1")).To(BeFalse())
	})

	It("treats empty IP as unknown", func() {
		limiter := NewIPConcurrencyLimiter(1)
		Expect(limiter.TryAcquire("")).To(BeTrue())
		Expect(limiter.TryAcquire("")).To(BeFalse())
		limiter.Release("")
		Expect(limiter.TryAcquire("unknown")).To(BeTrue())
	})
})

var _ = Describe("ClientIP", func() {
	DescribeTable("parses RemoteAddr",
		func(remoteAddr, expected string) {
			Expect(ClientIP(remoteAddr)).To(Equal(expected))
		},
		Entry("IPv4 host port", "192.0.2.1:12345", "192.0.2.1"),
		Entry("IPv6 host port", "[2001:db8::1]:443", "2001:db8::1"),
		Entry("bare IPv4", "192.0.2.1", "192.0.2.1"),
	)
})
