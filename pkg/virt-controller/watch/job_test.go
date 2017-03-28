package watch

import (
	"github.com/facebookgo/inject"
	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
	"net/http"

	kubeapi "k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	corev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/util/workqueue"
	kvirtv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Migration", func() {
	var server *ghttp.Server
	var jobCache cache.Store
	var vmService services.VMService
	var restClient *rest.RESTClient
	var vm *kvirtv1.VM
	var dispatch kubecli.ControllerDispatch
	var migration *kvirtv1.Migration
	var job *corev1.Pod
	var listOptions kubeapi.ListOptions = migrationJobSelector()
	var jobQueue workqueue.RateLimitingInterface
	var jobKey interface{}

	doExecute := func() {
		// Tell the controller function that there is a new Job
		jobKey, _ = cache.MetaNamespaceKeyFunc(job)
		jobCache.Add(job)
		jobQueue.Add(jobKey)
		dispatch.Execute(jobCache, jobQueue, jobKey)
	}

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

		jobCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		jobQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

		vm = kvirtv1.NewMinimalVM("test-vm")
		vm.Status.Phase = kvirtv1.Migrating
		vm.GetObjectMeta().SetLabels(map[string]string{"a": "b"})

		dispatch = NewJobControllerDispatch(vmService, restClient)
		migration = kvirtv1.NewMinimalMigration("test-migration", "test-vm")
		job = &corev1.Pod{
			ObjectMeta: corev1.ObjectMeta{
				Labels: map[string]string{
					kvirtv1.AppLabel:       "migration",
					kvirtv1.DomainLabel:    "test-vm",
					kvirtv1.MigrationLabel: migration.ObjectMeta.Name,
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodSucceeded,
			},
		}

	})

	Context("Running job", func() {
		It("Success should update the VM to Running", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVM(vm),
				handlerToUpdateTestVM(vm),
				handlerToFetchTestMigration(migration),
				handlerToUpdateTestMigration(migration, kvirtv1.MigrationSucceeded),
			)

			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(4))
			close(done)
		}, 10)

		It("should have valid listSelector", func() {

			f := fields.Set{"Status.Phase": string(corev1.PodSucceeded)}
			matches := listOptions.FieldSelector.Matches(f)
			Expect(matches).Should(BeTrue())

			s := labels.Set(job.Labels)
			matches = listOptions.LabelSelector.Matches(s)
			Expect(matches).Should(BeTrue())
		}, 100)

		It("Error calling Fetch VM should requeue", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVMAuthError(vm),
			)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("Error Updating VM should requeue", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVM(vm),
				handlerToUpdateTestVMAuthError(vm),
			)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("Error Fetching Migration should requeue", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVM(vm),
				handlerToUpdateTestVM(vm),
				handlerToFetchTestMigrationAuthError(migration),
			)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("Error Update Migration should requeue", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVM(vm),
				handlerToUpdateTestVM(vm),
				handlerToFetchTestMigration(migration),
				handlerToUpdateTestMigrationAuthError(migration),
			)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("should update the VM to ", func(done Done) {
			job.Status.Phase = corev1.PodFailed

			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestVM(vm),
				handlerToUpdateTestVM(vm),
				handlerToFetchTestMigration(migration),
				handlerToUpdateTestMigration(migration, kvirtv1.MigrationFailed),
			)

			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(4))
			close(done)
		}, 10)

	})

	AfterEach(func() {
		server.Close()
	})
})

func handlerToFetchTestVM(vm *kvirtv1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
	)
}

func handlerToFetchTestVMAuthError(vm *kvirtv1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, vm),
	)
}

func handlerToFetchTestMigration(migration *kvirtv1.Migration) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/"+migration.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
	)
}

func handlerToFetchTestMigrationAuthError(migration *kvirtv1.Migration) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/"+migration.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, migration),
	)
}

func handlerToUpdateTestMigration(migration *kvirtv1.Migration, expectedStatus kvirtv1.MigrationPhase) http.HandlerFunc {
	var expectedMigration kvirtv1.Migration = kvirtv1.Migration{}
	model.Copy(expectedMigration, migration)
	expectedMigration.Status.Phase = expectedStatus

	expectedMigration.Kind = "Migration"
	expectedMigration.APIVersion = "kubevirt.io/v1alpha1"
	expectedMigration.ObjectMeta.Name = "test-migration"
	expectedMigration.ObjectMeta.Namespace = "default"
	expectedMigration.ObjectMeta.SelfLink = "/apis/kubevirt.io/v1alpha1/namespaces/default/test-migration"
	expectedMigration.Spec.Selector.Name = "test-vm"

	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/"+migration.ObjectMeta.Name),
		ghttp.VerifyJSONRepresenting(expectedMigration),
		ghttp.RespondWithJSONEncoded(http.StatusOK, expectedMigration),
	)
}

func handlerToUpdateTestMigrationAuthError(migration *kvirtv1.Migration) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/"+migration.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, migration),
	)
}

func handlerToUpdateTestVM(vm *kvirtv1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
	)
}

func handlerToUpdateTestVMAuthError(vm *kvirtv1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, vm),
	)
}

func finishController(jobController *kubecli.Controller, stopChan chan struct{}) {
	// Wait until we have processed the added item

	jobController.WaitForSync(stopChan)
	jobController.ShutDownQueue()
	jobController.WaitUntilDone()
}
