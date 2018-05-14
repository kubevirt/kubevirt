package testutils

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/util/workqueue"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

/*
MockWorkQueue is a helper workqueue which can be wrapped around
any RateLimitingInterface implementing queue. This allows synchronous
testing of the controller. The typical pattern is:

    MockQueue.ExpectAdd(3)
    vmSource.Add(vm)
    vmSource.Add(vm1)
    vmSource.Add(vm2)
    MockQueue.Wait()

This ensures that Source callbacks which are listening on vmSource
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

func NewFakeInformerFor(obj runtime.Object) (cache.SharedIndexInformer, *framework.FakeControllerSource) {
	objSource := framework.NewFakeControllerSource()
	objInformer := cache.NewSharedIndexInformer(objSource, obj, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	return objInformer, objSource
}

type VirtualMachineFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *VirtualMachineFeeder) Add(vm *v1.VirtualMachine) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(vm)
	v.MockQueue.Wait()
}

func (v *VirtualMachineFeeder) Modify(vm *v1.VirtualMachine) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(vm)
	v.MockQueue.Wait()
}

func (v *VirtualMachineFeeder) Delete(vm *v1.VirtualMachine) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(vm)
	v.MockQueue.Wait()
}

func NewVirtualMachineFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *VirtualMachineFeeder {
	return &VirtualMachineFeeder{
		MockQueue: queue,
		Source:    source,
	}
}

type PodFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *PodFeeder) Add(pod *k8sv1.Pod) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(pod)
	v.MockQueue.Wait()
}

func (v *PodFeeder) Modify(pod *k8sv1.Pod) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(pod)
	v.MockQueue.Wait()
}

func (v *PodFeeder) Delete(pod *k8sv1.Pod) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(pod)
	v.MockQueue.Wait()
}

func NewPodFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *PodFeeder {
	return &PodFeeder{
		MockQueue: queue,
		Source:    source,
	}
}

type DomainFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *DomainFeeder) Add(vm *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(vm)
	v.MockQueue.Wait()
}

func (v *DomainFeeder) Modify(vm *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(vm)
	v.MockQueue.Wait()
}

func (v *DomainFeeder) Delete(vm *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(vm)
	v.MockQueue.Wait()
}

func NewDomainFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *DomainFeeder {
	return &DomainFeeder{
		MockQueue: queue,
		Source:    source,
	}
}
