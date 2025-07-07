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

package executor

import (
	"math"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"
)

// LimitedBackoff provides backoff rate limiter with limit functionality,
// when the limit is reached it stops to calculate next backoff.
type LimitedBackoff struct {
	clock       clock.Clock
	limit       time.Duration
	backoff     wait.Backoff
	maxStepTime time.Time
	stepEnd     time.Time
}

// Ready return true when the current backoff end-time passed and limit is not reached.
func (l *LimitedBackoff) Ready() bool {
	now := l.clock.Now()
	return now.After(l.stepEnd) && now.Before(l.maxStepTime)
}

// Step calculates the next backoff.
func (l *LimitedBackoff) Step() {
	if !l.Ready() {
		return
	}
	l.stepEnd = l.clock.Now().Add(l.backoff.Step())
}

const (
	DefaultMaxStep  = 5 * time.Hour
	DefaultDuration = 3 * time.Second
	DefaultFactor   = 1.8
)

func NewExponentialLimitedBackoffWithClock(limit time.Duration, clk clock.Clock) LimitedBackoff {
	// Create backoff object with initial duration of 3 seconds,
	// maximal step of 10 minutes, increasing factor of 1.8, no jitter.
	// 3s, 5.4s,... 10m, 10m, ..., 5h
	backoff := wait.Backoff{
		Duration: DefaultDuration,
		Cap:      10 * time.Minute,
		// the underlying wait.Backoff will stop to calculate the next duration once Steps reaches zero,
		// thus for an exponential backoff, Steps should approach infinity.
		Steps:  math.MaxInt64,
		Factor: DefaultFactor,
		Jitter: 0,
	}
	return newLimitedBackoffWithClock(backoff, limit, clk)
}

func newLimitedBackoffWithClock(backoff wait.Backoff, limit time.Duration, clk clock.Clock) LimitedBackoff {
	now := clk.Now()
	return LimitedBackoff{
		backoff:     backoff,
		limit:       limit,
		clock:       clk,
		stepEnd:     now,
		maxStepTime: now.Add(limit),
	}
}

type LimitedBackoffCreator struct {
	baseBackoff LimitedBackoff
}

func (l *LimitedBackoffCreator) New() LimitedBackoff {
	return newLimitedBackoffWithClock(l.baseBackoff.backoff, l.baseBackoff.limit, l.baseBackoff.clock)
}

func NewExponentialLimitedBackoffCreator() LimitedBackoffCreator {
	return LimitedBackoffCreator{
		baseBackoff: NewExponentialLimitedBackoffWithClock(DefaultMaxStep, clock.RealClock{}),
	}
}
