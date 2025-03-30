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

package executor_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/executor"
)

var _ = Describe("rate limited executor pool", func() {
	const (
		key1 = types.UID("20007000")
		key2 = types.UID("20006000")
	)

	var pool *executor.RateLimitedExecutorPool

	BeforeEach(func() {
		pool = executor.NewRateLimitedExecutorPool(executor.NewExponentialLimitedBackoffCreator())
	})

	It("should not override pool element if key exists", func() {
		initialElement := pool.LoadOrStore(key1)
		Expect(initialElement).To(BeIdenticalTo(pool.LoadOrStore(key1)))
		// mutate the element
		Expect(initialElement.Exec(failingCommandStub())).To(MatchError(testsExecError))
		Expect(initialElement).To(BeIdenticalTo(pool.LoadOrStore(key1)))
	})

	It("should delete element if key exists", func() {
		initialElement := pool.LoadOrStore(key1)
		pool.Delete(key1)
		Expect(initialElement).ToNot(Equal(pool.LoadOrStore(key1)))
	})

	It("should create unique elements", func() {
		element1 := pool.LoadOrStore(key1)
		Expect(element1).ToNot(Equal(pool.LoadOrStore(key2)))
	})
})
