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
 * Copyright The KubeVirt Authors.
 *
 */

package metadata

import (
	"fmt"
	"sync"
)

type SafeData[T comparable] struct {
	m           sync.Mutex
	initialized bool
	dirtyChanel chan<- struct{}
	data        T
}

// Load reads and returns safely the data and a flag.
// The flag specifies if the data is already initialized (true) or not (false).
// Data which is not yet initialized has never been stored.
func (d *SafeData[T]) Load() (T, bool) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.data, d.initialized
}

// Store persists safely the inputted data.
// As a side effect, it marks the data as initialized and
// in case a notification channel exists, a signal is sent.
func (d *SafeData[T]) Store(data T) {
	d.Set(data)
	d.notify()
}

// Set persists safely the inputted data.
// As a side effect, it marks the data as initialized.
//
// Note: Unlike Store, this method does not have a notification side effect.
func (d *SafeData[T]) Set(data T) {
	d.m.Lock()
	defer d.m.Unlock()
	d.data = data
	d.initialized = true
}

// WithSafeBlock calls the provided function with the data (reference) and a flag that specifies if the
// data is already initialized (true) or not (false).
// Data which is not yet initialized has never been stored.
// As a side effect, the method marks the data as initialized.
// In case a notification channel exists and the data changed, a signal is sent.
//
// Access to the data is protected by locks during the execution.
func (d *SafeData[T]) WithSafeBlock(f func(data *T, initialized bool)) {
	d.m.Lock()
	defer d.m.Unlock()
	oldData := d.data
	f(&d.data, d.initialized)
	d.initialized = true
	if oldData != d.data {
		d.notify()
	}
}

// notify sends a signal to notify listeners of a change in the data.
// The operation is non-blocking.
func (d *SafeData[T]) notify() {
	if d.dirtyChanel == nil {
		return
	}
	select {
	case d.dirtyChanel <- struct{}{}:
	default:
	}
}

func (d *SafeData[T]) String() string {
	d.m.Lock()
	defer d.m.Unlock()
	return fmt.Sprintf("%v", d.data)
}
