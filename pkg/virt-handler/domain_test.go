package virthandler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/pkg/types"
	"k8s.io/client-go/pkg/util/uuid"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	. "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
)

var _ = Describe("Domain", func() {

	var vmStore cache.Store
	var vmQueue workqueue.RateLimitingInterface
	var migrationStore cache.Store
	var migrationQueue workqueue.RateLimitingInterface
	var dispatch kubecli.ControllerDispatch

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		vmStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		dispatch = NewDomainDispatch(vmQueue, vmStore)

		migrationStore = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	})

	Context("A new domain appears on the host", func() {
		It("should inform vm controller if no correspnding VM is in the cache", func() {
			dom := virtwrap.NewMinimalDomain("testvm")

			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = ""
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			vmStore.Add(vm)
			key, _ := cache.MetaNamespaceKeyFunc(dom)
			migrationStore.Add(dom)
			migrationQueue.Add(key)
			dispatch.Execute(migrationStore, migrationQueue, key)

			Eventually(vmQueue.Len).Should(Equal(1))
		})
		It("should inform vm controller if a VM with a different UUID is in the cache", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.GetObjectMeta().SetUID(types.UID("uuid1"))
			vmStore.Add(vm)

			domain := virtwrap.NewMinimalDomain("testvm")
			domain.GetObjectMeta().SetUID(types.UID("uuid2"))
			migrationStore.Add(domain)
			key, _ := cache.MetaNamespaceKeyFunc(domain)
			migrationQueue.Add(key)
			dispatch.Execute(migrationStore, migrationQueue, key)
			Eventually(vmQueue.Len).Should(Equal(1))
		})
		It("should not inform vm controller if a correspnding VM is in the cache", func() {
			vmStore.Add(v1.NewMinimalVM("testvm"))
			domain := virtwrap.NewMinimalDomain("testvm")
			migrationStore.Add(domain)
			key, _ := cache.MetaNamespaceKeyFunc(domain)
			migrationQueue.Add(key)
			dispatch.Execute(migrationStore, migrationQueue, key)
			Consistently(vmQueue.Len).Should(Equal(0))
		})
	})

	AfterEach(func() {
	})
})
