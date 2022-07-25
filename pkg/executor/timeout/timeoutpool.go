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
	"sync"
)

type ExecutorPool struct {
	sync.Map
	creator TimerCreator
}

func NewExecutorPool(creator TimerCreator) *ExecutorPool {
	return &ExecutorPool{
		Map:     sync.Map{},
		creator: creator,
	}
}

// LoadOrStore returns the existing executor for the key if present.
// Otherwise, it will create a new timeout executor, store and return it.
func (c *ExecutorPool) LoadOrStore(key interface{}) Executor {
	newTimer := c.creator.New()
	executor, _ := c.Map.LoadOrStore(key, NewExecutor(newTimer))
	return executor.(Executor)
}
