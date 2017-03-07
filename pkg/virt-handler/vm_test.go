package virthandler_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	. "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"net/http"
)

var _ = Describe("VM", func() {
	var server *ghttp.Server
	var stopChan chan struct{}
	var fakeWatcher *framework.FakeControllerSource
	var vmStore cache.Store
	var vmQueue workqueue.RateLimitingInterface
	var ctrl *gomock.Controller
	var mockManager *virtwrap.MockDomainManager
	var controller *kubecli.Controller

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		stopChan = make(chan struct{})
		server = ghttp.NewServer()

		fakeWatcher = framework.NewFakeControllerSource()
		ctrl = gomock.NewController(GinkgoT())
		mockManager = virtwrap.NewMockDomainManager(ctrl)
		restClient, err := kubecli.GetRESTClientFromFlags(server.URL(), "")
		Expect(err).ToNot(HaveOccurred())
		vmStore, vmQueue, controller = NewVMController(fakeWatcher, mockManager, record.NewFakeRecorder(100), *restClient, "master")
		controller.StartInformer(stopChan)
		controller.WaitForSync(stopChan)
		go controller.Run(1, stopChan)
	})

	Context("VM controller gets informed about a Domain change through the Domain controller", func() {
		It("should kill the Domain if not cluster wide aequivalent exists", func(done Done) {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			mockManager.EXPECT().KillVM(v1.NewVMReferenceFromName("testvm")).Do(func(vm *v1.VM) {
				close(done)
			})
			vmQueue.Add("default/testvm")
		}, 1)
		It("should leave the Domain alone if the VM is migrating to its host", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.MigrationNodeName = "master"
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)
			vmQueue.Add("default/testvm")
			vmQueue.ShutDown()
			controller.WaitUntilDone()
		})
	})

	AfterEach(func() {
		close(stopChan)
		server.Close()
		ctrl.Finish()
	})
})
