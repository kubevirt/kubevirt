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

package cache

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"kubevirt.io/client-go/log"

	k6tpointer "kubevirt.io/kubevirt/pkg/pointer"
)

type TimeDefinedCache[T any] struct {
	minRefreshDuration time.Duration
	lastRefresh        *time.Time
	value              T
	reCalcFunc         func() (T, error)
	valueLock          *sync.Mutex
}

// NewTimeDefinedCache creates a new cache that will refresh the value every minRefreshDuration. If the value is requested
// before the minRefreshDuration has passed, the cached value will be returned. If minRefreshDuration is zero, the value will always be
// recalculated.
// In addition, a Set() can be used to explicitly set the value.
// If useValueLock is set to true, the value will be locked when being set. If the cache won't be used concurrently, it's safe
// to set this to false.
func NewTimeDefinedCache[T any](minRefreshDuration time.Duration, useValueLock bool, reCalcFunc func() (T, error)) (*TimeDefinedCache[T], error) {
	if reCalcFunc == nil {
		return nil, fmt.Errorf("re-calculation function is not set")
	}

	t := &TimeDefinedCache[T]{
		minRefreshDuration: minRefreshDuration,
		reCalcFunc:         reCalcFunc,
	}

	if useValueLock {
		t.valueLock = &sync.Mutex{}
	}

	return t, nil
}

func (t *TimeDefinedCache[T]) Get() (T, error) {
	if t.valueLock != nil {
		t.valueLock.Lock()
		defer t.valueLock.Unlock()
	}

	if t.lastRefresh != nil && t.minRefreshDuration.Nanoseconds() != 0 && time.Since(*t.lastRefresh) <= t.minRefreshDuration {
		return t.value, nil
	}

	value, err := t.reCalcFunc()
	if err != nil {
		return t.value, err
	}

	t.setWithoutLock(value)

	return t.value, nil
}

func (t *TimeDefinedCache[T]) Set(value T) {
	if t.valueLock != nil {
		t.valueLock.Lock()
		defer t.valueLock.Unlock()
	}

	t.setWithoutLock(value)
}

// KeepValueUpdated will keep the value updated in the cache by calling the re-calculation function every minRefreshDuration
// until the stopChannel is closed.
func (t *TimeDefinedCache[T]) KeepValueUpdated(stopChannel chan struct{}) error {
	if t.minRefreshDuration.Nanoseconds() == 0 {
		return fmt.Errorf("KeepValueUpdated can only be used if minRefreshDuration is non-zero, but it is %s", t.minRefreshDuration.String())
	}

	go func() {
		updateCachedValue := func() {
			_, err := t.Get()
			if err != nil {
				log.Log.Errorf("Error updating cache: %v", err)
			}
		}

		wait.JitterUntil(updateCachedValue, t.minRefreshDuration, 0, false, stopChannel)
	}()

	return nil
}

func (t *TimeDefinedCache[T]) setWithoutLock(value T) {
	t.value = value

	if t.lastRefresh == nil || t.minRefreshDuration.Nanoseconds() != 0 {
		t.lastRefresh = k6tpointer.P(time.Now())
	}
}

func (t *TimeDefinedCache[T]) SetReCalcFunc(reCalcFunc func() (T, error)) {
	t.reCalcFunc = reCalcFunc
}
