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

package virthandler

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("virt-handler retry manager", func() {
	const initialWait, maxWait, maxFailResponseTime = 500 * time.Millisecond, 5 * time.Second, 1 * time.Second
	const buffer = 100 * time.Millisecond
	var retryManager *FailRetryManager
	const key = "c4ab4ae0-db63-45d8-aa0f-fc53dc84bdab"

	BeforeEach(func() {
		retryManager = NewFailRetryManager("test", initialWait, maxWait, maxFailResponseTime)
	})

	It("Should not delay if no failure occurred", func() {
		shouldDelay, _ := retryManager.ShouldDelay(key, getBoolFunc(false))
		Expect(shouldDelay).To(BeFalse())
	})

	It("Should not delay on first failure", func() {
		shouldDelay, _ := retryManager.ShouldDelay(key, func() bool { return true })
		Expect(shouldDelay).To(BeFalse())
		Expect(retryManager.retryStates[key].lastRunFailed).To(BeFalse())

		// Call again as a failure response to the above call.
		shouldDelay, _ = retryManager.ShouldDelay(key, getBoolFunc(true))
		Expect(shouldDelay).To(BeTrue())
		Expect(retryManager.retryStates[key].lastRunFailed).To(BeTrue())
	})

	It("Should do exponential backoff on failure retries", func() {
		retryManager.ShouldDelay(key, getBoolFunc(true))
		retryManager.ShouldDelay(key, getBoolFunc(true))

		// check waitInterval before the timer ends
		time.Sleep(initialWait - buffer)
		shouldDelay, _ := retryManager.ShouldDelay(key, getBoolFunc(true))
		Expect(shouldDelay).To(BeTrue())
		Expect(retryManager.retryStates[key].waitInterval).To(BeEquivalentTo(initialWait))

		// check waitInterval just after the timer ends
		time.Sleep(buffer)
		shouldDelay, _ = retryManager.ShouldDelay(key, getBoolFunc(true))
		Expect(shouldDelay).To(BeFalse())
		Expect(retryManager.retryStates[key].waitInterval).To(BeEquivalentTo(initialWait * 2))
	})

	It("Should reset when no failure is reported within the max response time", func() {
		shouldDelay, _ := retryManager.ShouldDelay(key, getBoolFunc(true))
		Expect(shouldDelay).To(BeFalse())
		firstEventFirstFail := retryManager.retryStates[key].firstFail

		time.Sleep(maxFailResponseTime + buffer)
		shouldDelay, _ = retryManager.ShouldDelay(key, getBoolFunc(true))
		Expect(shouldDelay).To(BeFalse())
		lastEventFirstFail := retryManager.retryStates[key].firstFail
		Expect(lastEventFirstFail).ToNot(BeEquivalentTo(firstEventFirstFail))
	})
})

func getBoolFunc(b bool) func() bool {
	return func() bool {
		return b
	}
}
