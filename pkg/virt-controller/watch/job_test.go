package watch

import (
	"net/http"

	"github.com/facebookgo/inject"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	corev1 "k8s.io/client-go/pkg/api/v1"
	batchv1 "k8s.io/client-go/pkg/apis/batch/v1"
	kvirtv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Migration", func() {
	var server *ghttp.Server
	var stopChan chan struct{}
	var jobController *kubecli.Controller
	var lw *framework.FakeControllerSource
	var jobCache cache.Store

	var vmService services.VMService
	var restClient *rest.RESTClient
	var vm *kvirtv1.VM

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		var g inject.Graph
		vmService = services.NewVMService()
		server = ghttp.NewServer()
		config := rest.Config{}
		config.Host = server.URL()
		clientSet, _ := kubernetes.NewForConfig(&config)
		templateService, _ := services.NewTemplateService("kubevirt/virt-launcher")
		restClient, _ = kubecli.GetRESTClientFromFlags(server.URL(), "")

		g.Provide(
			&inject.Object{Value: restClient},
			&inject.Object{Value: clientSet},
			&inject.Object{Value: vmService},
			&inject.Object{Value: templateService},
		)
		g.Populate()

		stopChan = make(chan struct{})
		lw = framework.NewFakeControllerSource()
		jobCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)

		_, jobController = NewJobControllerWithListWatch(vmService, nil, lw, restClient)

		vm = kvirtv1.NewMinimalVM("test-vm")

		// Start the controller
		jobController.StartInformer(stopChan)
		go jobController.Run(1, stopChan)
	})

	Context("Running job with out migration labels", func() {
		It("should not attempt to update the VM", func(done Done) {

			job := &batchv1.Job{}

			// Register the expected REST call
			//server.AppendHandlers()

			// Tell the controller that there is a new Migration
			lw.Add(job)

			// Wait until we have processed the added item
			finishController(jobController, stopChan)

			Expect(len(server.ReceivedRequests())).To(Equal(0))
			close(done)
		}, 10)
	})

	Context("Running job with migration labels but no success", func() {
		It("should ignore the the VM ", func(done Done) {

			job := &batchv1.Job{
				ObjectMeta: corev1.ObjectMeta{
					Labels: map[string]string{
						kvirtv1.DomainLabel: "something",
						"vmname":            vm.ObjectMeta.Name,
					},
				},
			}

			// No registered REST calls
			//server.AppendHandlers()

			// Tell the controller that there is a new Job
			lw.Add(job)

			// Wait until we have processed the added item
			finishController(jobController, stopChan)

			Expect(len(server.ReceivedRequests())).To(Equal(0))
			close(done)
		}, 10)
	})

	Context("Running job with migration labels and one success", func() {
		It("should update the VM to Running", func(done Done) {

			job := &batchv1.Job{
				ObjectMeta: corev1.ObjectMeta{
					Labels: map[string]string{
						kvirtv1.DomainLabel: "something",
						"vmname":            vm.ObjectMeta.Name,
					},
				},
				Status: batchv1.JobStatus{
					Succeeded: 1,
					Failed:    0,
					Active:    0,
				},
			}

			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVM(vm),
				handlerToUpdateTestVM(vm),
			)

			// Tell the controller that there is a new Job
			lw.Add(job)
			finishController(jobController, stopChan)
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		close(stopChan)
		server.Close()
	})
})

func handlerToFetchTestVM(vm *kvirtv1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
	)
}

func handlerToUpdateTestVM(vm *kvirtv1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
	)
}

func finishController(jobController *kubecli.Controller, stopChan chan struct{}) {
	// Wait until we have processed the added item

	jobController.WaitForSync(stopChan)
	jobController.ShutDownQueue()
	jobController.WaitUntilDone()
}
