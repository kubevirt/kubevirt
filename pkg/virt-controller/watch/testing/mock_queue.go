package testing

import (
	"sync"

	"k8s.io/client-go/util/workqueue"
)

/*
MockWorkQueue is a helper workqueue which can be wrapped around
any RateLimitingInterface implementing queue. This allows synchronous
testing of the controller. The typical pattern is:

    mockQueue.ExpectAdd(3)
    vmSource.Add(vm)
    vmSource.Add(vm1)
    vmSource.Add(vm2)
    mockQueue.Wait()

This ensures that informer callbacks which are listening on vmSource
enqueued three times an object. Since enqueing is typically the last
action in listener callbacks, we can assume that the wanted scenario for
a controller is set up, and an execution will process this scenario.
*/
type MockWorkQueue struct {
	workqueue.RateLimitingInterface
	addWG *sync.WaitGroup
}

func (q *MockWorkQueue) Add(obj interface{}) {
	q.RateLimitingInterface.Add(obj)
	if q.addWG != nil {
		q.addWG.Done()
	}
}

// ExpectAdds allows setting the amount of expected enqueues.
func (q *MockWorkQueue) ExpectAdds(diff int) {
	q.addWG = &sync.WaitGroup{}
	q.addWG.Add(diff)
}

// Wait waits until the expected amount of ExpectedAdds has happened.
// It will not block if there were no expectations set.
func (q *MockWorkQueue) Wait() {
	if q.addWG != nil {
		q.addWG.Wait()
		q.addWG = nil
	}
}

func NewMockWorkQueue(queue workqueue.RateLimitingInterface) *MockWorkQueue {
	return &MockWorkQueue{queue, nil}
}
