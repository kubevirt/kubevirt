package watch

import (
	"net/http"

	"github.com/facebookgo/inject"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
	clientv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/conversion"
	"k8s.io/client-go/pkg/util/uuid"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("VM watcher", func() {
	var server *ghttp.Server
	var migrationCache cache.Store

	var vmService services.VMService
	var restClient *rest.RESTClient

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)
	var dispatch kubecli.ControllerDispatch

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

		dispatch = NewVMControllerFunc(restClient, vmService)

		migrationCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)

	})

	Context("Creating a VM ", func() {
		It("should should schedule a POD.", func(done Done) {

			// Create a VM to be scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = ""
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			// Create the expected VM after the update
			obj, err := conversion.NewCloner().DeepCopy(vm)
			Expect(err).ToNot(HaveOccurred())

			// Create a Pod for the VM
			temlateService, err := services.NewTemplateService("whatever")
			Expect(err).ToNot(HaveOccurred())
			pod, err := temlateService.RenderLaunchManifest(vm)
			Expect(err).ToNot(HaveOccurred())
			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = clientv1.PodSucceeded

			podListInitial := clientv1.PodList{}
			podListInitial.Items = []clientv1.Pod{}

			podListPostCreate := clientv1.PodList{}
			podListPostCreate.Items = []clientv1.Pod{*pod}

			expectedVM := obj.(*v1.VM)
			expectedVM.Status.Phase = v1.Scheduling
			expectedVM.Status.MigrationNodeName = pod.Spec.NodeName

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podListInitial),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),

				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
				),
			)

			// Tell the controller that there is a new VM
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			key, _ := cache.MetaNamespaceKeyFunc(vm)
			migrationCache.Add(vm)
			queue.Add(key)
			dispatch.Execute(migrationCache, queue, key)

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		server.Close()
	})
})
