package testutils

import (
	"sync"
	"sync/atomic"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/util/workqueue"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	v1 "kubevirt.io/kubevirt/pkg/api/v1"
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
}

func (q *MockWorkQueue) Add(obj interface{}) {
	q.RateLimitingInterface.Add(obj)
	if q.addWG != nil {
		q.addWG.Done()
	}
}

func (q *MockWorkQueue) AddRateLimited(item interface{}) {
	q.RateLimitingInterface.AddRateLimited(item)
	atomic.AddInt32(&q.rateLimitedEnque, 1)
}

func (q *MockWorkQueue) GetRateLimitedEnqueueCount() int {
	return int(atomic.LoadInt32(&q.rateLimitedEnque))
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
	return &MockWorkQueue{queue, nil, 0}
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
