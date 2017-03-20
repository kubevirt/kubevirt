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

var _ = Describe("Migration", func() {
	var server *ghttp.Server
	var migrationCache cache.Store

	var vmService services.VMService
	var restClient *rest.RESTClient
	var dispatch kubecli.ControllerDispatch

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
		migrationCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		dispatch = NewMigrationControllerFunc(vmService)
	})

	Context("Running Migration target Pod for a running VM given", func() {
		It("should update the VM with the migration target node of the running Pod", func(done Done) {

			// Create a VM which is being scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = v1.Running
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			migration := v1.NewMinimalMigration(vm.ObjectMeta.Name+"-migration", vm.ObjectMeta.Name)
			migration.ObjectMeta.SetUID(uuid.NewUUID())

			// Create the expected VM after the update
			obj, err := conversion.NewCloner().DeepCopy(vm)
			Expect(err).ToNot(HaveOccurred())

			// Create a target Pod for the VM
			temlateService, err := services.NewTemplateService("whatever")
			Expect(err).ToNot(HaveOccurred())
			pod, err := temlateService.RenderLaunchManifest(vm)
			Expect(err).ToNot(HaveOccurred())
			pod.Spec.NodeName = "mynode"
			pod.Status.Phase = clientv1.PodSucceeded
			pod.Labels[v1.DomainLabel] = migration.ObjectMeta.Name

			podList := clientv1.PodList{}
			podList.Items = []clientv1.Pod{*pod}

			expectedVM := obj.(*v1.VM)
			expectedVM.Status.Phase = v1.Running
			expectedVM.Status.MigrationNodeName = pod.Spec.NodeName

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
				),
			)

			// Tell the controller that there is a new Migration

			// Tell the controller that there is a new VM
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			key, _ := cache.MetaNamespaceKeyFunc(migration)
			migrationCache.Add(migration)
			queue.Add(key)
			dispatch.Execute(migrationCache, queue, key)

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		server.Close()
	})
})
