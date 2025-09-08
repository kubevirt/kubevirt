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

package cache_test

import (
	"fmt"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtcache "kubevirt.io/kubevirt/tools/cache"
)

var _ = Describe("one-shot cache", func() {
	It("should execute the calculation function only once", func() {
		mockValue := 0
		getMockCalcFunc := func() func() (int, error) {
			return func() (int, error) {
				mockValue++
				return mockValue, nil
			}
		}

		cache, err := virtcache.NewOneShotCache(getMockCalcFunc())
		Expect(err).ToNot(HaveOccurred())

		const goroutineCount = 20
		const callsPerGoroutine = 100
		wg := sync.WaitGroup{}
		wg.Add(goroutineCount)
		getReturnValues := make(chan int, goroutineCount*callsPerGoroutine)

		getValueFromCache := func() {
			defer GinkgoRecover()
			defer wg.Done()

			for i := 0; i < callsPerGoroutine; i++ {
				ret, err := cache.Get()
				Expect(err).ShouldNot(HaveOccurred())
				getReturnValues <- ret
			}
		}

		for i := 0; i < goroutineCount; i++ {
			go getValueFromCache()
		}

		wg.Wait()
		Expect(getReturnValues).To(HaveLen(goroutineCount*callsPerGoroutine), "some of the calls were missed")
		Expect(mockValue).To(Equal(1), "the calculation function should be called only once")
	})

	It("ensure re-calculation is performed on error", func() {
		const (
			failureCount   = 5
			valueOnErr     = -1
			valueOnSuccess = 1
		)

		executions := 0
		getMockCalcFunc := func() func() (int, error) {
			return func() (int, error) {
				if executions < failureCount {
					executions++
					return valueOnErr, fmt.Errorf("mock error")
				}
				return valueOnSuccess, nil
			}
		}

		cache, err := virtcache.NewOneShotCache(getMockCalcFunc())
		Expect(err).ToNot(HaveOccurred())

		for i := 0; i < failureCount; i++ {
			ret, err := cache.Get()
			Expect(err).Should(HaveOccurred(), fmt.Sprintf("expected an error on the first %d calls", failureCount))
			Expect(ret).To(Equal(valueOnErr))
		}

		ret, err := cache.Get()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ret).To(Equal(1))
	})
})
