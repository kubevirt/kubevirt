package virthandler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	. "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

var _ = Describe("Domain", func() {
	var stopChan chan struct{}
	var fakeWatcher *framework.FakeControllerSource
	var vmStore cache.Store
	var vmQueue workqueue.RateLimitingInterface
	var restClient rest.RESTClient

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		stopChan = make(chan struct{})

		fakeWatcher = framework.NewFakeControllerSource()
		vmStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		informer := cache.NewSharedInformer(fakeWatcher, &virtwrap.Domain{}, 0)
		_, controller := NewDomainController(vmQueue, vmStore, informer, restClient, record.NewFakeRecorder(100))
		controller.StartInformer(stopChan)
		controller.WaitForSync(stopChan)
		go controller.Run(1, stopChan)
	})

	Context("A new domain appears on the host", func() {
		It("should inform vm controller if no correspnding VM is in the cache", func() {
			fakeWatcher.Add(virtwrap.NewMinimalDomain("testvm"))
			Eventually(vmQueue.Len).Should(Equal(1))
		})
		It("should inform vm controller if a VM with a different UUID is in the cache", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.GetObjectMeta().SetUID(types.UID("uuid1"))
			domain := virtwrap.NewMinimalDomain("testvm")
			domain.GetObjectMeta().SetUID(types.UID("uuid2"))
			vmStore.Add(vm)
			fakeWatcher.Add(domain)
			Eventually(vmQueue.Len).Should(Equal(1))
		})
		It("should not inform vm controller if a correspnding VM is in the cache", func() {
			vmStore.Add(v1.NewMinimalVM("testvm"))
			fakeWatcher.Add(virtwrap.NewMinimalDomain("testvm"))
			Consistently(vmQueue.Len).Should(Equal(0))
		})
	})

	AfterEach(func() {
		close(stopChan)
	})
})
