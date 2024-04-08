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
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	virtcache "kubevirt.io/kubevirt/tools/cache"
)

var _ = Describe("time defined cache", func() {

	getMockCalcFunc := func() func() (int, error) {
		mockValue := 0
		return func() (int, error) {
			mockValue++
			return mockValue, nil
		}
	}

	DescribeTable("should get new or cached value according to refresh duration", func(refreshDurationPassed bool) {
		minRefreshDuration := 123 * time.Second
		if refreshDurationPassed {
			minRefreshDuration = 0
		}
		cache, err := virtcache.NewTimeDefinedCache(minRefreshDuration, false, getMockCalcFunc())
		Expect(err).ToNot(HaveOccurred())
		value, err := cache.Get()
		Expect(err).ToNot(HaveOccurred())
		Expect(value).To(Equal(1))

		value, err = cache.Get()
		Expect(err).ToNot(HaveOccurred())

		if refreshDurationPassed {
			Expect(value).To(Equal(2))
		} else {
			Expect(value).To(Equal(1))
		}
	},
		Entry("should get the same value if the refresh duration has not passed", false),
		Entry("should get a new value if the refresh duration has passed", true),
	)

	It("should return an error if the re-calculation function is not set", func() {
		var dummyFunc func() (int, error)
		dummyFunc = nil

		_, err := virtcache.NewTimeDefinedCache(0, false, dummyFunc)
		Expect(err).To(HaveOccurred())
	})

	It("should not allow two threads to set value in parallel", func() {
		stopChannel := make(chan struct{})
		defer close(stopChannel)
		firstCallMadeChannel := make(chan struct{})
		defer close(firstCallMadeChannel)

		recalcFunctionCalls := int64(0)

		recalcFunc := func() (int, error) {
			firstCallMadeChannel <- struct{}{}
			atomic.AddInt64(&recalcFunctionCalls, 1)

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			select {
			case <-ticker.C:
				time.Sleep(100 * time.Millisecond)
			case <-stopChannel:
				break
			}

			return int(recalcFunctionCalls), nil
		}

		cache, err := virtcache.NewTimeDefinedCache(0, true, recalcFunc)
		Expect(err).ToNot(HaveOccurred())

		getValueFromCache := func() {
			defer GinkgoRecover()
			_, err = cache.Get()
			Expect(err).ShouldNot(HaveOccurred())
		}

		for i := 0; i < 5; i++ {
			go getValueFromCache()
		}

		// To ensure the first call is already made
		<-firstCallMadeChannel

		Consistently(func() {
			Expect(recalcFunctionCalls).To(Equal(int64(1)), "value is being re-calculated, only one caller is expected")
		}).WithPolling(250 * time.Millisecond).WithTimeout(1 * time.Second)
	})

	It("should not allow two threads to get value in parallel", func() {
		firstCallBarrier := make(chan struct{})
		recalcFunctionCallsCount := int64(0)

		recalcFunc := func() (int, error) {
			atomic.AddInt64(&recalcFunctionCallsCount, 1)
			firstCallBarrier <- struct{}{}
			return int(recalcFunctionCallsCount), nil
		}

		cache, err := virtcache.NewTimeDefinedCache(time.Hour, true, recalcFunc)
		Expect(err).ToNot(HaveOccurred())

		const goroutineCount = 20
		getReturnValues := make(chan int, goroutineCount)
		getValueFromCache := func() {
			defer GinkgoRecover()
			ret, err := cache.Get()
			Expect(err).ShouldNot(HaveOccurred())
			getReturnValues <- ret
		}

		for i := 0; i < goroutineCount; i++ {
			go getValueFromCache()
		}

		Consistently(getReturnValues).Should(BeEmpty(), "all go routines should wait for the first one to finish")
		Eventually(firstCallBarrier).Should(Receive(), "first go routine should start re-calculating")
		Eventually(getReturnValues).Should(HaveLen(goroutineCount), fmt.Sprintf("expected all go routines to finish calling Get(). %d/%d finished", len(getReturnValues), goroutineCount))

		close(getReturnValues)
		for getValue := range getReturnValues {
			Expect(getValue).To(Equal(1), "Get() calls are expected to return the cached value")
		}
		Expect(firstCallBarrier).To(BeEmpty(), "ensure no other go routine called the re-calculation funtion")
	})

})
