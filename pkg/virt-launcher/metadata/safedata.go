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

package metadata

import (
	"sync"
)

type SafeData[T any] struct {
	m           sync.Mutex
	initialized bool
	data        T
}

func (d *SafeData[T]) Load() (T, bool) {
	d.m.Lock()
	defer d.m.Unlock()
	return d.data, d.initialized
}

func (d *SafeData[T]) Store(data T) {
	d.m.Lock()
	defer d.m.Unlock()
	d.initialized = true
	d.data = data
}

// WithSafeBlock calls the provided function with the data (reference) and a flag that specifies if the
// data is already initialized (true) or not (false).
// Data which is not yet initialized has never been stored.
// As a side effect, the method marks the data as initialized.
//
// Access to the data is protected by locks during the execution.
func (d *SafeData[T]) WithSafeBlock(f func(data *T, initialized bool)) {
	d.m.Lock()
	defer d.m.Unlock()
	f(&d.data, d.initialized)
	d.initialized = true
}
