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

func (q *MockPriorityQueue[T]) AddWithOpts(o priorityqueue.AddOpts, Items ...T) {
	q.PriorityQueue.AddWithOpts(o, Items...)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		for range Items {
			q.addWG.Done()
		}
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
