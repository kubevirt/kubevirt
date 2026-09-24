package framework

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
)

// blockingQueue pauses one worker before or after Get marks an item as processing.
type blockingQueue struct {
	workqueue.TypedRateLimitingInterface[string]
	afterGet     bool
	entered      chan struct{}
	release      chan struct{}
	shutdown     chan struct{}
	done         chan string
	blockOnce    sync.Once
	shutdownOnce sync.Once
}

func (q *blockingQueue) Get() (string, bool) {
	if !q.afterGet {
		q.block()
	}
	key, quit := q.TypedRateLimitingInterface.Get()
	if q.afterGet && !quit {
		q.block()
	}
	return key, quit
}

func (q *blockingQueue) block() {
	q.blockOnce.Do(func() {
		close(q.entered)
		<-q.release
	})
}

func (q *blockingQueue) ShutDown() {
	q.TypedRateLimitingInterface.ShutDown()
	q.shutdownOnce.Do(func() { close(q.shutdown) })
}

func (q *blockingQueue) Done(key string) {
	q.TypedRateLimitingInterface.Done(key)
	q.done <- key
}

var _ = Describe("Framework shutdown", func() {
	DescribeTable("waits for controller workers before stopping",
		func(ctx SpecContext, useVMI, afterGet bool) {
			f := new(Framework)
			f.ctx, f.cancel = context.WithCancel(ctx)
			factory := newFakeFrameworkInformerFactory()
			q := &blockingQueue{
				TypedRateLimitingInterface: workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[string]()),
				afterGet:                   afterGet,
				entered:                    make(chan struct{}),
				release:                    make(chan struct{}),
				shutdown:                   make(chan struct{}),
				done:                       make(chan string, 1),
			}
			unblock := sync.OnceFunc(func() { close(q.release) })
			DeferCleanup(f.Stop)
			DeferCleanup(unblock)
			var run func(int, <-chan struct{})
			clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
			if useVMI {
				f.vmiController = buildVMIController(nil, factory, clusterConfig, new(record.FakeRecorder))
				f.vmiController.Queue.ShutDown()
				f.vmiController.Queue = q
				run = f.vmiController.Run
			} else {
				f.vmController = buildVMController(nil, factory, clusterConfig, new(record.FakeRecorder))
				f.vmController.Queue.ShutDown()
				f.vmController.Queue = q
				run = f.vmController.Run
			}
			var syncs []cache.InformerSynced
			for _, informer := range frameworkInformers(factory) {
				syncs = append(syncs, informer.HasSynced)
				f.wg.Go(func() { informer.RunWithContext(f.ctx) })
			}
			Expect(cache.WaitForCacheSync(ctx.Done(), syncs...)).To(BeTrue())
			q.Add("test/queued")
			f.wg.Go(func() { run(1, f.ctx.Done()) })
			Eventually(ctx, q.entered).Should(BeClosed())

			stopped := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				f.Stop()
				close(stopped)
			}()
			Eventually(ctx, q.shutdown).Should(BeClosed())
			Consistently(ctx, stopped, 100*time.Millisecond).ShouldNot(BeClosed())
			unblock()
			Eventually(ctx, stopped).Should(BeClosed())
			Expect(q.done).To(Receive(Equal("test/queued")))
		},
		Entry("VM with queued work", false, false, SpecTimeout(5*time.Second)),
		Entry("VM with in-flight work", false, true, SpecTimeout(5*time.Second)),
		Entry("VMI with queued work", true, false, SpecTimeout(5*time.Second)),
		Entry("VMI with in-flight work", true, true, SpecTimeout(5*time.Second)),
	)
})

type fakeFrameworkInformerFactory struct {
	controller.KubeInformerFactory
	informers map[string]cache.SharedIndexInformer
}

func newFakeFrameworkInformerFactory() *fakeFrameworkInformerFactory {
	return &fakeFrameworkInformerFactory{informers: make(map[string]cache.SharedIndexInformer)}
}

func (f *fakeFrameworkInformerFactory) informer(obj runtime.Object) cache.SharedIndexInformer {
	key := fmt.Sprintf("%T", obj)
	if informer, exists := f.informers[key]; exists {
		return informer
	}
	informer, _ := testutils.NewFakeInformerFor(obj)
	f.informers[key] = informer
	return informer
}

func (f *fakeFrameworkInformerFactory) VMI() cache.SharedIndexInformer {
	return f.informer(&virtv1.VirtualMachineInstance{})
}

func (f *fakeFrameworkInformerFactory) VirtualMachine() cache.SharedIndexInformer {
	return f.informer(&virtv1.VirtualMachine{})
}

func (f *fakeFrameworkInformerFactory) KubeVirtPod() cache.SharedIndexInformer {
	return f.informer(&k8sv1.Pod{})
}

func (f *fakeFrameworkInformerFactory) PersistentVolumeClaim() cache.SharedIndexInformer {
	return f.informer(&k8sv1.PersistentVolumeClaim{})
}

func (f *fakeFrameworkInformerFactory) VirtualMachineInstanceMigration() cache.SharedIndexInformer {
	return f.informer(&virtv1.VirtualMachineInstanceMigration{})
}

func (f *fakeFrameworkInformerFactory) StorageClass() cache.SharedIndexInformer {
	return f.informer(&storagev1.StorageClass{})
}

func (f *fakeFrameworkInformerFactory) Namespace() cache.SharedIndexInformer {
	return f.informer(&k8sv1.Namespace{})
}

func (f *fakeFrameworkInformerFactory) ControllerRevision() cache.SharedIndexInformer {
	return f.informer(&appsv1.ControllerRevision{})
}

func (f *fakeFrameworkInformerFactory) KubeVirt() cache.SharedIndexInformer {
	return f.informer(&virtv1.KubeVirt{})
}

func (f *fakeFrameworkInformerFactory) ResourceQuota() cache.SharedIndexInformer {
	return f.informer(&k8sv1.ResourceQuota{})
}

func (f *fakeFrameworkInformerFactory) DummyDataVolume() cache.SharedIndexInformer {
	return f.informer(&cdiv1.DataVolume{})
}

func (f *fakeFrameworkInformerFactory) DummyDataSource() cache.SharedIndexInformer {
	return f.informer(&cdiv1.DataSource{})
}

func (f *fakeFrameworkInformerFactory) DummyStorageProfile() cache.SharedIndexInformer {
	return f.informer(&cdiv1.StorageProfile{})
}

func (f *fakeFrameworkInformerFactory) DummyCDI() cache.SharedIndexInformer {
	return f.informer(&cdiv1.CDI{})
}

func (f *fakeFrameworkInformerFactory) DummyCDIConfig() cache.SharedIndexInformer {
	return f.informer(&cdiv1.CDIConfig{})
}

func (f *fakeFrameworkInformerFactory) VirtualMachineBackup() cache.SharedIndexInformer {
	return f.informer(&backupv1.VirtualMachineBackup{})
}

func (f *fakeFrameworkInformerFactory) VirtualMachineBackupTracker() cache.SharedIndexInformer {
	return f.informer(&backupv1.VirtualMachineBackupTracker{})
}
