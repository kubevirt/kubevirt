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

package ratelimit_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/executor/ratelimit"
)

var _ = Describe("rate limited executor", func() {
	It("should execute command when the underlying rate-limiter is not blocking", func() {
		executor := ratelimit.NewExecutor(&rateLimiterStub{block: false})
		expectCommandExec(executor)
		expectCommandExec(executor)
	})

	It("should not execute command when the underlying rate-limiter is blocking", func() {
		executor := ratelimit.NewExecutor(&rateLimiterStub{block: true})
		expectSkipCommandExec(executor)
		expectSkipCommandExec(executor)
	})
})

func expectCommandExec(executor *ratelimit.Executor) {
	ExpectWithOffset(1, executor.Exec(failingCommandStub())).To(MatchError(errTestsExec))
}

func expectSkipCommandExec(executor *ratelimit.Executor) {
	ExpectWithOffset(1, executor.Exec(failingCommandStub())).To(Succeed())
}

type rateLimiterStub struct {
	block bool
}

func (r *rateLimiterStub) Ready() bool {
	return !r.block
}

func (r *rateLimiterStub) Step() {}
