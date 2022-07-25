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

package timeout

import (
	"time"
)

type Timer struct {
	expirationTime time.Time
}

func NewTimer(duration time.Duration) Timer {
	now := time.Now()
	return Timer{expirationTime: now.Add(duration)}
}

func (t Timer) Expired() bool {
	now := time.Now()
	return now.After(t.expirationTime)
}

type TimerCreator struct {
	timeout time.Duration
}

func NewTimerCreator(timeout time.Duration) TimerCreator {
	return TimerCreator{timeout: timeout}
}

func (t TimerCreator) New() Timer {
	return NewTimer(t.timeout)
}
