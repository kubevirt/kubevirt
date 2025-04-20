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
 */

package testutils

import (
	"sync"
	"sync/atomic"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/controller/priorityqueue"
)

/*
MockPriorityQueue is a helper workqueue which can be wrapped around
any RateLimitingInterface implementing queue. This allows synchronous
testing of the controller. The typical pattern is:

	MockQueue.ExpectAdd(3)
	vmiSource.Add(vmi)
	vmiSource.Add(vmi1)
	vmiSource.Add(vmi2)
	MockQueue.Wait()

This ensures that Source callbacks which are listening on vmiSource
enqueued three times an object. Since enqueing is typically the last
action in listener callbacks, we can assume that the wanted scenario for
a controller is set up, and an execution will process this scenario.
*/
type MockPriorityQueue[T comparable] struct {
	priorityqueue.PriorityQueue[T]
	addWG            *sync.WaitGroup
	rateLimitedEnque int32
	addAfterEnque    int32
	wgLock           sync.Mutex
}

func (q *MockPriorityQueue[T]) Add(item T) {
	q.PriorityQueue.Add(item)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockPriorityQueue[T]) AddRateLimited(item T) {
	q.PriorityQueue.AddRateLimited(item)
	atomic.AddInt32(&q.rateLimitedEnque, 1)
}

func (q *MockPriorityQueue[T]) AddAfter(item T, duration time.Duration) {
	q.PriorityQueue.AddAfter(item, duration)
	atomic.AddInt32(&q.addAfterEnque, 1)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockPriorityQueue[T]) GetRateLimitedEnqueueCount() int {
	return int(atomic.LoadInt32(&q.rateLimitedEnque))
}

func (q *MockPriorityQueue[T]) GetAddAfterEnqueueCount() int {
	return int(atomic.LoadInt32(&q.addAfterEnque))
}

// ExpectAdds allows setting the amount of expected enqueues.
func (q *MockPriorityQueue[T]) ExpectAdds(diff int) {
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	q.addWG = &sync.WaitGroup{}
	q.addWG.Add(diff)
}

// Wait waits until the expected amount of ExpectedAdds has happened.
// It will not block if there were no expectations set.
func (q *MockPriorityQueue[T]) Wait() {
	q.wgLock.Lock()
	wg := q.addWG
	q.wgLock.Unlock()
	if wg != nil {
		wg.Wait()
		q.wgLock.Lock()
		q.addWG = nil
		q.wgLock.Unlock()
	}
}

func NewMockPriorityQueue[T comparable](queue priorityqueue.PriorityQueue[T]) *MockPriorityQueue[T] {
	return &MockPriorityQueue[T]{queue, nil, 0, 0, sync.Mutex{}}
}
