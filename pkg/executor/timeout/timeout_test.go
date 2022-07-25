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

package timeout_test

import (
	"errors"
	"time"

	"k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/executor/timeout"
)

var _ = Describe("executor", func() {
	It("should execute successful command before timeout", func() {
		executed := false
		executor := timeout.NewExecutor(timeout.NewTimer(time.Minute))
		Expect(executor.Exec(func() error {
			executed = true
			return nil
		})).To(Succeed())
		Expect(executed).To(BeTrue())
	})

	It("should execute failing command before timeout", func() {
		testError := errors.New("test")
		executor := timeout.NewExecutor(timeout.NewTimer(time.Minute))
		Expect(executor.Exec(func() error {
			return testError
		})).To(MatchError(testError))
	})

	It("should not execute command after timeout", func() {
		executor := timeout.NewExecutor(timeout.NewTimer(0))
		Expect(executor.Exec(func() error {
			return errors.New("test")
		})).To(Succeed())
	})
})

var _ = Describe("executor pool", func() {

	const (
		key1 = types.UID("20007000")
		key2 = types.UID("20006000")
	)

	var pool *timeout.ExecutorPool

	BeforeEach(func() {
		pool = timeout.NewExecutorPool(timeout.NewTimerCreator(time.Minute))
	})

	It("should not override data element if key exists", func() {
		initialElement := pool.LoadOrStore(key1)
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
