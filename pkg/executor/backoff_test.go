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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	clock "k8s.io/utils/clock/testing"

	"kubevirt.io/kubevirt/pkg/executor"
)

var _ = Describe("exponential limited backoff", func() {

	var testsClock *clock.FakeClock
	var backoff executor.LimitedBackoff

	BeforeEach(func() {
		testsClock = clock.NewFakeClock(time.Time{})
		backoff = executor.NewExponentialLimitedBackoffWithClock(executor.DefaultMaxStep, testsClock)

		testsClock.Step(time.Nanosecond)
	})

	It("should be ready before first step", func() {
		Expect(backoff.Ready()).To(BeTrue())
	})

	It("should not be ready before step end time", func() {
		Expect(backoff.Ready()).To(BeTrue())
		backoff.Step()
		Expect(backoff.Ready()).To(BeFalse())
	})

	It("should be ready after step end time passed", func() {
		firstStepDuration := executor.DefaultDuration + time.Nanosecond
		secondStepDuration := time.Duration(float64(firstStepDuration)*executor.DefaultFactor) + time.Nanosecond

		Expect(backoff.Ready()).To(BeTrue())
		backoff.Step()

		Expect(backoff.Ready()).To(BeFalse())
		testsClock.Step(firstStepDuration)

		Expect(backoff.Ready()).To(BeTrue())
		backoff.Step()

		Expect(backoff.Ready()).To(BeFalse())
		testsClock.Step(secondStepDuration)

		Expect(backoff.Ready()).To(BeTrue())
	})

	It("should not be ready after max time is passed", func() {
		testsClock.Step(executor.DefaultMaxStep + time.Nanosecond)

		Expect(backoff.Ready()).To(BeFalse())
		backoff.Step()
		Expect(backoff.Ready()).To(BeFalse())
		backoff.Step()
		Expect(backoff.Ready()).To(BeFalse())
	})
})
