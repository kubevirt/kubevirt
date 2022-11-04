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
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

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
type MockWorkQueue struct {
	workqueue.RateLimitingInterface
	addWG            *sync.WaitGroup
	rateLimitedEnque int32
	addAfterEnque    int32
	wgLock           sync.Mutex
}

func (q *MockWorkQueue) Add(obj interface{}) {
	q.RateLimitingInterface.Add(obj)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockWorkQueue) AddRateLimited(item interface{}) {
	q.RateLimitingInterface.AddRateLimited(item)
	atomic.AddInt32(&q.rateLimitedEnque, 1)
}

func (q *MockWorkQueue) AddAfter(item interface{}, duration time.Duration) {
	q.RateLimitingInterface.AddAfter(item, duration)
	atomic.AddInt32(&q.addAfterEnque, 1)
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockWorkQueue) GetRateLimitedEnqueueCount() int {
	return int(atomic.LoadInt32(&q.rateLimitedEnque))
}

func (q *MockWorkQueue) GetAddAfterEnqueueCount() int {
	return int(atomic.LoadInt32(&q.addAfterEnque))
}

// ExpectAdds allows setting the amount of expected enqueues.
func (q *MockWorkQueue) ExpectAdds(diff int) {
	q.wgLock.Lock()
	defer q.wgLock.Unlock()
	q.addWG = &sync.WaitGroup{}
	q.addWG.Add(diff)
}

// Wait waits until the expected amount of ExpectedAdds has happened.
// It will not block if there were no expectations set.
func (q *MockWorkQueue) Wait() {
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

func NewMockWorkQueue(queue workqueue.RateLimitingInterface) *MockWorkQueue {
	return &MockWorkQueue{queue, nil, 0, 0, sync.Mutex{}}
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

type VirtualMachineFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *VirtualMachineFeeder) Add(vmi *v1.VirtualMachineInstance) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(vmi)
	v.MockQueue.Wait()
}

func (v *VirtualMachineFeeder) Modify(vmi *v1.VirtualMachineInstance) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(vmi)
	v.MockQueue.Wait()
}

func (v *VirtualMachineFeeder) Delete(vmi *v1.VirtualMachineInstance) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(vmi)
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

type PodDisruptionBudgetFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *PodDisruptionBudgetFeeder) Add(pdb *policyv1.PodDisruptionBudget) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(pdb)
	v.MockQueue.Wait()
}

func (v *PodDisruptionBudgetFeeder) Modify(pdb *policyv1.PodDisruptionBudget) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(pdb)
	v.MockQueue.Wait()
}

func (v *PodDisruptionBudgetFeeder) Delete(pdb *policyv1.PodDisruptionBudget) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(pdb)
	v.MockQueue.Wait()
}

func NewPodDisruptionBudgetFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *PodDisruptionBudgetFeeder {
	return &PodDisruptionBudgetFeeder{
		MockQueue: queue,
		Source:    source,
	}
}

type MigrationFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *MigrationFeeder) Add(migration *v1.VirtualMachineInstanceMigration) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(migration)
	v.MockQueue.Wait()
}

func (v *MigrationFeeder) Modify(migration *v1.VirtualMachineInstanceMigration) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(migration)
	v.MockQueue.Wait()
}

func (v *MigrationFeeder) Delete(migration *v1.VirtualMachineInstanceMigration) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(migration)
	v.MockQueue.Wait()
}

func NewMigrationFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *MigrationFeeder {
	return &MigrationFeeder{
		MockQueue: queue,
		Source:    source,
	}
}

type DomainFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *DomainFeeder) Add(vmi *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(vmi)
	v.MockQueue.Wait()
}

func (v *DomainFeeder) Modify(vmi *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(vmi)
	v.MockQueue.Wait()
}

func (v *DomainFeeder) Delete(vmi *api.Domain) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(vmi)
	v.MockQueue.Wait()
}

func NewDomainFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *DomainFeeder {
	return &DomainFeeder{
		MockQueue: queue,
		Source:    source,
	}
}

type DataVolumeFeeder struct {
	MockQueue *MockWorkQueue
	Source    *framework.FakeControllerSource
}

func (v *DataVolumeFeeder) Add(dataVolume *cdiv1.DataVolume) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Add(dataVolume)
	v.MockQueue.Wait()
}

func (v *DataVolumeFeeder) Modify(dataVolume *cdiv1.DataVolume) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Modify(dataVolume)
	v.MockQueue.Wait()
}

func (v *DataVolumeFeeder) Delete(dataVolume *cdiv1.DataVolume) {
	v.MockQueue.ExpectAdds(1)
	v.Source.Delete(dataVolume)
	v.MockQueue.Wait()
}

func NewDataVolumeFeeder(queue *MockWorkQueue, source *framework.FakeControllerSource) *DataVolumeFeeder {
	return &DataVolumeFeeder{
		MockQueue: queue,
		Source:    source,
	}
}
