package testutils

import (
	"sync"
	"sync/atomic"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

/*
MockWorkQueue is a helper workqueue which can be wrapped around
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
type MockWorkQueue[T comparable] struct {
	workqueue.TypedRateLimitingInterface[T]
	addWG            *sync.WaitGroup
	rateLimitedEnque int32
	addAfterEnque    int32
	wgLock           sync.Mutex
}

func (q *MockWorkQueue[T]) Add(item T) {
	q.TypedRateLimitingInterface.Add(item)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockWorkQueue[T]) AddRateLimited(item T) {
	q.TypedRateLimitingInterface.AddRateLimited(item)
	atomic.AddInt32(&q.rateLimitedEnque, 1)
}

func (q *MockWorkQueue[T]) AddAfter(item T, duration time.Duration) {
	q.TypedRateLimitingInterface.AddAfter(item, duration)
	atomic.AddInt32(&q.addAfterEnque, 1)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockWorkQueue[T]) GetRateLimitedEnqueueCount() int {
	return int(atomic.LoadInt32(&q.rateLimitedEnque))
}

func (q *MockWorkQueue[T]) GetAddAfterEnqueueCount() int {
	return int(atomic.LoadInt32(&q.addAfterEnque))
}

// ExpectAdds allows setting the amount of expected enqueues.
func (q *MockWorkQueue[T]) ExpectAdds(diff int) {
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	q.addWG = &sync.WaitGroup{}
	q.addWG.Add(diff)
}

// Wait waits until the expected amount of ExpectedAdds has happened.
// It will not block if there were no expectations set.
func (q *MockWorkQueue[T]) Wait() {
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

func NewMockWorkQueue[T comparable](queue workqueue.TypedRateLimitingInterface[T]) *MockWorkQueue[T] {
	return &MockWorkQueue[T]{queue, nil, 0, 0, sync.Mutex{}}
}

func NewFakeInformerFor(obj runtime.Object) (cache.SharedIndexInformer, *framework.FakeControllerSource) {
	objSource := framework.NewFakeControllerSource()
	objInformer := cache.NewSharedIndexInformer(objSource, obj, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	return objInformer, objSource
}

func NewFakeInformerWithIndexersFor(obj runtime.Object, indexers cache.Indexers) (cache.SharedIndexInformer, *framework.FakeControllerSource) {
	objSource := framework.NewFakeControllerSource()
	objInformer := cache.NewSharedIndexInformer(objSource, obj, 0, indexers)
	return objInformer, objSource
}

type VirtualMachineFeeder[T comparable] struct {
	MockQueue *MockWorkQueue[T]
	Source    *framework.FakeControllerSource
}

func (v *VirtualMachineFeeder[T]) Add(vmi *v1.VirtualMachineInstance) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(vmi)
	v.MockQueue.Wait()
}

func (v *VirtualMachineFeeder[T]) Modify(vmi *v1.VirtualMachineInstance) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(vmi)
	v.MockQueue.Wait()
}

func (v *VirtualMachineFeeder[T]) Delete(vmi *v1.VirtualMachineInstance) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(vmi)
	v.MockQueue.Wait()
}

func NewVirtualMachineFeeder[T comparable](queue *MockWorkQueue[T], source *framework.FakeControllerSource) *VirtualMachineFeeder[T] {
	return &VirtualMachineFeeder[T]{
		MockQueue: queue,
		Source:    source,
	}
}

type PodFeeder[T comparable] struct {
	MockQueue *MockWorkQueue[T]
	Source    *framework.FakeControllerSource
}

func (v *PodFeeder[T]) Add(pod *k8sv1.Pod) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(pod)
	v.MockQueue.Wait()
}

func (v *PodFeeder[T]) Modify(pod *k8sv1.Pod) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(pod)
	v.MockQueue.Wait()
}

func (v *PodFeeder[T]) Delete(pod *k8sv1.Pod) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(pod)
	v.MockQueue.Wait()
}

func NewPodFeeder[T comparable](queue *MockWorkQueue[T], source *framework.FakeControllerSource) *PodFeeder[T] {
	return &PodFeeder[T]{
		MockQueue: queue,
		Source:    source,
	}
}

type PodDisruptionBudgetFeeder[T comparable] struct {
	MockQueue *MockWorkQueue[T]
	Source    *framework.FakeControllerSource
}

func (v *PodDisruptionBudgetFeeder[T]) Add(pdb *policyv1.PodDisruptionBudget) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(pdb)
	v.MockQueue.Wait()
}

func (v *PodDisruptionBudgetFeeder[T]) Modify(pdb *policyv1.PodDisruptionBudget) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(pdb)
	v.MockQueue.Wait()
}

func (v *PodDisruptionBudgetFeeder[T]) Delete(pdb *policyv1.PodDisruptionBudget) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(pdb)
	v.MockQueue.Wait()
}

func NewPodDisruptionBudgetFeeder[T comparable](queue *MockWorkQueue[T], source *framework.FakeControllerSource) *PodDisruptionBudgetFeeder[T] {
	return &PodDisruptionBudgetFeeder[T]{
		MockQueue: queue,
		Source:    source,
	}
}

type MigrationFeeder[T comparable] struct {
	MockQueue *MockWorkQueue[T]
	Source    *framework.FakeControllerSource
}

func (v *MigrationFeeder[T]) Add(migration *v1.VirtualMachineInstanceMigration) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(migration)
	v.MockQueue.Wait()
}

func (v *MigrationFeeder[T]) Modify(migration *v1.VirtualMachineInstanceMigration) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(migration)
	v.MockQueue.Wait()
}

func (v *MigrationFeeder[T]) Delete(migration *v1.VirtualMachineInstanceMigration) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(migration)
	v.MockQueue.Wait()
}

func NewMigrationFeeder[T comparable](queue *MockWorkQueue[T], source *framework.FakeControllerSource) *MigrationFeeder[T] {
	return &MigrationFeeder[T]{
		MockQueue: queue,
		Source:    source,
	}
}

type DomainFeeder[T comparable] struct {
	MockQueue *MockWorkQueue[T]
	Source    *framework.FakeControllerSource
}

func (v *DomainFeeder[T]) Add(vmi *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(vmi)
	v.MockQueue.Wait()
}

func (v *DomainFeeder[T]) Modify(vmi *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(vmi)
	v.MockQueue.Wait()
}

func (v *DomainFeeder[T]) Delete(vmi *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(vmi)
	v.MockQueue.Wait()
}

func NewDomainFeeder[T comparable](queue *MockWorkQueue[T], source *framework.FakeControllerSource) *DomainFeeder[T] {
	return &DomainFeeder[T]{
		MockQueue: queue,
		Source:    source,
	}
}
