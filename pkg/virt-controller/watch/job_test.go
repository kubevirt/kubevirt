package watch

import (
	"net/http"

	"github.com/facebookgo/inject"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	corev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	kvirtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
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
	var migrationQueue workqueue.RateLimitingInterface

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
		migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
		vm = kvirtv1.NewMinimalVM("test-vm")
		vm.Status.Phase = kvirtv1.Migrating
		vm.GetObjectMeta().SetLabels(map[string]string{"a": "b"})

		dispatch = NewJobControllerDispatch(vmService, restClient, migrationQueue)
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
				handlerToFetchTestMigration(migration),
			)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(0))
			Expect(migrationQueue.Len()).Should(Equal(1))
			close(done)
		}, 10)

		It("should have valid listSelector", func() {

			f := fields.Set{"Status.Phase": string(corev1.PodSucceeded)}
			matches := listOptions.FieldSelector.Matches(f)
			Expect(matches).Should(BeTrue())

			s := labels.Set(job.Labels)
			matches = listOptions.LabelSelector.Matches(s)
			Expect(matches).Should(BeTrue())
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(0))
		}, 100)

		It("Error Fetching Migration should requeue", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handlerToFetchTestMigrationAuthError(migration),
			)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(jobQueue.NumRequeues(jobKey)).Should(Equal(1))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		server.Close()
	})
})

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
