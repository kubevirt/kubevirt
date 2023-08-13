package cache

import (
	"fmt"
	"sync"
	"time"
)

type TimeDefinedCache[T any] struct {
	minRefreshDuration time.Duration
	lastRefresh        time.Time
	savedValueSet      bool
	savedValue         T
	reCalcFunc         func() (T, error)
	valueLock          *sync.Mutex
}

// NewTimeDefinedCache creates a new cache that will refresh the value every minRefreshDuration. If the value is requested
// before the minRefreshDuration has passed, the cached value will be returned. If minRefreshDuration is zero, the value will always be
// recalculated.
// In addition, a Set() can be used to explicitly set the value.
// If useValueLock is set to true, the value will be locked when being set. If the cache won't be used concurrently, it's safe
// to set this to false.
func NewTimeDefinedCache[T any](minRefreshDuration time.Duration, useValueLock bool, reCalcFunc func() (T, error)) *TimeDefinedCache[T] {
	t := &TimeDefinedCache[T]{
		minRefreshDuration: minRefreshDuration,
		reCalcFunc:         reCalcFunc,
	}

	if useValueLock {
		t.valueLock = &sync.Mutex{}
	}

	return t
}

func (t *TimeDefinedCache[T]) Get() (T, error) {
	if t.valueLock != nil {
		t.valueLock.Lock()
		defer t.valueLock.Unlock()
	}

	if t.reCalcFunc == nil {
		return t.savedValue, fmt.Errorf("re-calculation function is not set")
	}

	if t.savedValueSet && t.minRefreshDuration.Nanoseconds() != 0 && time.Since(t.lastRefresh) <= t.minRefreshDuration {
		return t.savedValue, nil
	}

	value, err := t.reCalcFunc()
	if err != nil {
		return t.savedValue, err
	}

	t.setWithoutLock(value)

	return t.savedValue, nil
}

func (t *TimeDefinedCache[T]) Set(value T) {
	if t.valueLock != nil {
		t.valueLock.Lock()
		defer t.valueLock.Unlock()
	}

	t.setWithoutLock(value)
}

func (t *TimeDefinedCache[T]) setWithoutLock(value T) {
	t.savedValue = value
	t.savedValueSet = true
	t.lastRefresh = time.Now()
}
