package watch

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/facebookgo/inject"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
	clientv1 "k8s.io/client-go/pkg/api/v1"
	kubev1 "k8s.io/client-go/pkg/api/v1"
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

	var (
		server               *ghttp.Server
		migrationCache       cache.Store
		podCache             cache.Store
		vmCache              cache.Store
		vmService            services.VMService
		restClient           *rest.RESTClient
		dispatch             kubecli.ControllerDispatch
		migrationPodDispatch kubecli.ControllerDispatch
		migrationQueue       workqueue.RateLimitingInterface
		migration            *v1.Migration
		vm                   *v1.VM
		pod                  *clientv1.Pod
		podList              clientv1.PodList
		migrationKey         interface{}
		srcIp                clientv1.NodeAddress
		destIp               kubev1.NodeAddress
		srcNodeWithIp        kubev1.Node
		destNodeWithIp       kubev1.Node
		srcNode              kubev1.Node
		destNode             kubev1.Node
	)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	destinationNodeName := "mynode"
	sourceNodeName := "sourcenode"
	BeforeEach(func() {
		var g inject.Graph
		vmService = services.NewVMService()
		server = ghttp.NewServer()
		config := rest.Config{}
		config.Host = server.URL()
		clientSet, _ := kubernetes.NewForConfig(&config)
		templateService, _ := services.NewTemplateService("kubevirt/virt-launcher")
		restClient, _ = kubecli.GetRESTClientFromFlags(server.URL(), "")

		podCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		vmCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		g.Provide(
			&inject.Object{Value: restClient},
			&inject.Object{Value: clientSet},
			&inject.Object{Value: vmService},
			&inject.Object{Value: templateService},
		)
		g.Populate()
		migrationCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		dispatch = NewMigrationControllerDispatch(vmService, restClient, clientSet)
		migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

		// Create a VM which is being scheduled

		vm = v1.NewMinimalVM("testvm")
		vm.Status.Phase = v1.Running
		vm.ObjectMeta.SetUID(uuid.NewUUID())

		migration = v1.NewMinimalMigration(vm.ObjectMeta.Name+"-migration", vm.ObjectMeta.Name)
		migration.ObjectMeta.SetUID(uuid.NewUUID())
		migration.Spec.NodeSelector = map[string]string{"beta.kubernetes.io/arch": "amd64"}
		migrationPodDispatch = NewMigrationPodControllerDispatch(vmCache, restClient, vmService, clientSet, migrationQueue)

		// Create a target Pod for the VM
		templateService, err := services.NewTemplateService("whatever")
		Expect(err).ToNot(HaveOccurred())
		pod, err = templateService.RenderLaunchManifest(vm)
		Expect(err).ToNot(HaveOccurred())

		pod.Spec.NodeName = destinationNodeName
		pod.Status.Phase = clientv1.PodSucceeded
		pod.Labels[v1.DomainLabel] = migration.ObjectMeta.Name

		podList = clientv1.PodList{}
		podList.Items = []clientv1.Pod{*pod}

		srcIp = kubev1.NodeAddress{
			Type:    kubev1.NodeInternalIP,
			Address: "127.0.0.2",
		}
		destIp = kubev1.NodeAddress{
			Type:    kubev1.NodeInternalIP,
			Address: "127.0.0.3",
		}
		srcNodeWithIp = kubev1.Node{
			Status: kubev1.NodeStatus{
				Addresses: []kubev1.NodeAddress{srcIp},
			},
		}
		destNodeWithIp = kubev1.Node{
			Status: kubev1.NodeStatus{
				Addresses: []kubev1.NodeAddress{destIp},
			},
		}

		srcNode = clientv1.Node{
			ObjectMeta: clientv1.ObjectMeta{
				Name: sourceNodeName,
			},
			Status: clientv1.NodeStatus{
				Addresses: []clientv1.NodeAddress{srcIp, destIp},
			},
		}
		destNode = clientv1.Node{
			ObjectMeta: clientv1.ObjectMeta{
				Name: destinationNodeName,
			},
			Status: clientv1.NodeStatus{
				Addresses: []clientv1.NodeAddress{destIp, srcIp},
			},
		}
	})

	doExecute := func() {
		migrationKey, _ = cache.MetaNamespaceKeyFunc(migration)
		dispatch.Execute(migrationCache, migrationQueue, migrationKey)
	}

	buildExpectedVM := func(phase v1.VMPhase) *v1.VM {

		obj, err := conversion.NewCloner().DeepCopy(vm)
		Expect(err).ToNot(HaveOccurred())

		expectedVM := obj.(*v1.VM)
		expectedVM.Status.Phase = phase
		expectedVM.Status.MigrationNodeName = pod.Spec.NodeName
		expectedVM.Spec.NodeSelector = map[string]string{"beta.kubernetes.io/arch": "amd64"}
		expectedVM.ObjectMeta.Labels = map[string]string{}

		return expectedVM
	}

	handlePutMigration := func(migration *v1.Migration, expectedStatus v1.MigrationPhase) http.HandlerFunc {

		obj, err := conversion.NewCloner().DeepCopy(migration)
		Expect(err).ToNot(HaveOccurred())

		expectedMigration := obj.(*v1.Migration)
		expectedMigration.Status.Phase = expectedStatus

		return ghttp.CombineHandlers(
			ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
			ghttp.VerifyJSONRepresenting(expectedMigration),
			ghttp.RespondWithJSONEncoded(http.StatusOK, expectedMigration),
		)
	}

	Context("Running Migration target Pod for a running VM given", func() {
		It("should update the VM with the migration target node of the running Pod", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(podList),
				handleCreatePod(pod),
				handlePutMigration(migration, v1.MigrationScheduled),
			)
			migrationCache.Add(migration)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))

			close(done)
		}, 10)

		It("failed GET oF VM should requeue", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVMAuthError(buildExpectedVM(v1.Running)),
			)
			migrationCache.Add(migration)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("failed GET oF Pod List should requeue", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodListAuthError(podList),
			)
			migrationCache.Add(migration)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("Should Mark Migration as failed if VM Not found.", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVMNotFound(),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			migrationCache.Add(migration)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			close(done)
		}, 10)

		It("should requeue if VM Not found and Migration update error.", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVMNotFound(),
				handlePutMigrationAuthError(),
			)
			migrationCache.Add(migration)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("Should mark Migration failed if VM not running ", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Pending)),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			migrationCache.Add(migration)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			close(done)
		}, 10)

		It("Should Requeue if VM not running and updateMigratio0n Failure", func(done Done) {
			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Pending)),
				handlePutMigrationAuthError(),
			)
			migrationCache.Add(migration)
			doExecute()
			Expect(len(server.ReceivedRequests())).To(Equal(2))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("should requeue if Migration update fails", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(podList),
				handleCreatePod(pod),
				handlePutMigrationAuthError(),
			)
			migrationCache.Add(migration)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(4))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("should fail if conflicting VM and Migration have conflicting Node Selectors", func(done Done) {
			vm := buildExpectedVM(v1.Running)
			vm.Spec.NodeSelector = map[string]string{"beta.kubernetes.io/arch": "i386"}

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(vm),
			)
			migrationCache.Add(migration)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("should requeue if create of the Target Pod fails ", func(done Done) {

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(podList),
				handleCreatePodAuthError(),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			migrationCache.Add(migration)

			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))
			close(done)
		}, 10)

		It("should fail if another migration is in process.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}
			unmatchedPodList.Items = []clientv1.Pod{mockPod(1, "bogus"), mockPod(2, "bogus")}

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))
			close(done)
		}, 10)

		It("should requeue if another migration is in process and migration update fails.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}
			unmatchedPodList.Items = []clientv1.Pod{mockPod(1, "bogus"), mockPod(2, "bogus")}

			// Register the expected REST call
			server.AppendHandlers(
				handleGetTestVM(buildExpectedVM(v1.Running)),
				handleGetPodList(unmatchedPodList),
				handlePutMigrationAuthError(),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(1))

			close(done)
		}, 10)

		It("should succeed if many migrations, and this is one.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.NodeName = sourceNodeName

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationScheduled),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))

			close(done)
		}, 10)

		It("should create migration Pod if migration and pod not created.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM1 := buildExpectedVM(v1.Migrating)
			expectedVM1.Status.MigrationNodeName = destinationNodeName
			expectedVM2 := buildExpectedVM(v1.Running)
			expectedVM2.Status.MigrationNodeName = destinationNodeName

			migrationPodList := clientv1.PodList{}
			migrationPodList.Items = []clientv1.Pod{
				*mockMigrationPod(expectedVM0),
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationScheduled),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))

			close(done)
		}, 10)

		It("should set vm.Status.MigrationNodeName if Not set.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.MigrationNodeName = ""

			expectedVM1 := buildExpectedVM(v1.Migrating)

			expectedVM2 := buildExpectedVM(v1.Migrating)
			expectedVM2.Status.MigrationNodeName = destinationNodeName

			migrationPodList := clientv1.PodList{}
			migrationPodList.Items = []clientv1.Pod{
				*mockMigrationPod(expectedVM0),
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handlePutMigration(migration, v1.MigrationScheduled),
				handlePutVM(expectedVM1),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(3))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))

			close(done)
		}, 10)

		It("should mark migration as successful if migration pod completes successfully.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}
			migration.Status.Phase = v1.MigrationRunning

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.NodeName = sourceNodeName
			expectedVM1 := buildExpectedVM(v1.Migrating)
			expectedVM1.Status.MigrationNodeName = destinationNodeName
			expectedVM2 := buildExpectedVM(v1.Running)
			expectedVM2.Status.MigrationNodeName = ""
			expectedVM2.Status.NodeName = destinationNodeName
			expectedVM2.ObjectMeta.Labels = map[string]string{v1.NodeNameLabel: destinationNodeName}

			migrationPod := *mockMigrationPod(expectedVM2)
			migrationPod.Status.Phase = clientv1.PodSucceeded

			migrationPodList := clientv1.PodList{
				Items: []clientv1.Pod{
					migrationPod,
				},
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handleGetPodList(migrationPodList),
				handlePutVM(expectedVM2),
				handlePutMigration(migration, v1.MigrationSucceeded),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(5))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))

			close(done)
		}, 10)

		It("should mark migration as failed if migration pod fails.", func(done Done) {

			unmatchedPodList := clientv1.PodList{}
			migration.Status.Phase = v1.MigrationRunning

			migrationLabel := string(migration.GetObjectMeta().GetUID())

			targetPod := mockPod(3, migrationLabel)
			targetPod.Spec = clientv1.PodSpec{
				NodeName: destinationNodeName,
			}
			unmatchedPodList.Items = []clientv1.Pod{
				mockPod(1, "bogus"),
				mockPod(2, "bogus"),
				targetPod}

			// Register the expected REST call
			expectedVM0 := buildExpectedVM(v1.Running)
			expectedVM0.Status.NodeName = sourceNodeName
			expectedVM1 := buildExpectedVM(v1.Running)
			expectedVM1.Status.MigrationNodeName = ""
			expectedVM1.Status.NodeName = sourceNodeName

			migrationPod := *mockMigrationPod(expectedVM1)
			migrationPod.Status.Phase = clientv1.PodFailed

			migrationPodList := clientv1.PodList{
				Items: []clientv1.Pod{
					migrationPod,
				},
			}

			server.AppendHandlers(
				handleGetTestVM(expectedVM0),
				handleGetPodList(unmatchedPodList),
				handleGetPodList(migrationPodList),
				handlePutVM(expectedVM1),
				handlePutMigration(migration, v1.MigrationFailed),
			)
			migrationCache.Add(migration)
			doExecute()

			Expect(len(server.ReceivedRequests())).To(Equal(5))
			Expect(migrationQueue.NumRequeues(migrationKey)).Should(Equal(0))

			close(done)
		}, 10)

	})

	Context("Running Migration target Pod for a running VM given", func() {
		var (
			srcIp      = kubev1.NodeAddress{}
			destIp     = kubev1.NodeAddress{}
			srcNodeIp  = kubev1.Node{}
			destNodeIp = kubev1.Node{}
			srcNode    kubev1.Node
			targetNode kubev1.Node
		)

		BeforeEach(func() {
			srcIp = kubev1.NodeAddress{
				Type:    kubev1.NodeInternalIP,
				Address: "127.0.0.2",
			}
			destIp = kubev1.NodeAddress{
				Type:    kubev1.NodeInternalIP,
				Address: "127.0.0.3",
			}
			srcNodeIp = kubev1.Node{
				Status: kubev1.NodeStatus{
					Addresses: []kubev1.NodeAddress{srcIp},
				},
			}
			destNodeIp = kubev1.Node{
				Status: kubev1.NodeStatus{
					Addresses: []kubev1.NodeAddress{destIp},
				},
			}
			srcNode = kubev1.Node{
				ObjectMeta: kubev1.ObjectMeta{
					Name: "sourceNode",
				},
				Status: kubev1.NodeStatus{
					Addresses: []kubev1.NodeAddress{srcIp, destIp},
				},
			}
			targetNode = kubev1.Node{
				ObjectMeta: kubev1.ObjectMeta{
					Name: "targetNode",
				},
				Status: kubev1.NodeStatus{
					Addresses: []kubev1.NodeAddress{destIp, srcIp},
				},
			}
		})

		It("should update the VM Phase and migration target node of the running Pod", func(done Done) {

			// Create a VM which is being scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = v1.Running
			vm.ObjectMeta.SetUID(uuid.NewUUID())
			vm.Status.NodeName = "sourceNode"

			// Add the VM to the cache
			vmCache.Add(vm)

			// Create a target Pod for the VM
			pod := mockMigrationPod(vm)

			// Create the expected VM after the update
			obj, err := conversion.NewCloner().DeepCopy(vm)
			Expect(err).ToNot(HaveOccurred())

			vmWithMigrationNodeName := obj.(*v1.VM)
			vmWithMigrationNodeName.Status.MigrationNodeName = pod.Spec.NodeName

			obj, err = conversion.NewCloner().DeepCopy(vmWithMigrationNodeName)
			Expect(err).ToNot(HaveOccurred())

			vmInMigrationState := obj.(*v1.VM)
			vmInMigrationState.Status.Phase = v1.Migrating
			migration := v1.NewMinimalMigration("testvm-migration", "testvm")
			migration.Status.Phase = v1.MigrationScheduled

			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			key, _ := cache.MetaNamespaceKeyFunc(pod)
			podCache.Add(pod)
			queue.Add(key)
			migrationPodDispatch.Execute(podCache, queue, key)

			Expect(migrationQueue.Len()).To(Equal(1))
			close(done)
		}, 10)
	})

	Context("Pod Investigation", func() {
		var (
			podList      kubev1.PodList
			migration     *v1.Migration
		)

		BeforeEach(func() {
			pod1 := kubev1.Pod{
				ObjectMeta: kubev1.ObjectMeta{Name: "pod1",
					Labels: map[string]string{
						v1.MigrationUIDLabel: "ce662d9f-34c0-40fd-a4e4-abe4146a1457",
						v1.MigrationLabel: "test-migration1",
					},
					Namespace: "test",
				},
			}
			pod2 := kubev1.Pod{
				ObjectMeta: kubev1.ObjectMeta{Name: "pod2",
					Labels: map[string]string{
						v1.MigrationUIDLabel: "99a8ac71-4ced-48fa-9bd4-0b4322dcc3dd",
						v1.MigrationLabel: "test-migration1",
					},
					Namespace: "test",
				},
			}
			pod3 := kubev1.Pod{
				ObjectMeta: kubev1.ObjectMeta{Name: "pod3",
					Labels: map[string]string{
						v1.MigrationUIDLabel: "7efc4067-039e-4c21-a494-0b52c09fe6fb",
						v1.MigrationLabel: "test-migration2",
					},
					Namespace: "test",
				},
			}
			podList.Items = []kubev1.Pod{pod1, pod2, pod3}

			migrationCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			migration = new(v1.Migration)
			migration.ObjectMeta.Name = "test-migration1"
			migration.ObjectMeta.Namespace = "test"
			migration.Status.Phase = v1.MigrationFailed
		})

		It("should count exact matches", func() {
			migration.ObjectMeta.UID = "ce662d9f-34c0-40fd-a4e4-abe4146a1457"
			podList.Items = []kubev1.Pod{podList.Items[0]}
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(1))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[0].ObjectMeta.Name))
		})

		It("should count uncached pods", func() {
			migration.ObjectMeta.UID = "99a8ac71-4ced-48fa-9bd4-0b4322dcc3dd"
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(3))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[1].ObjectMeta.Name))
		})

		It("should ignore finalized migrations", func() {
			migration.ObjectMeta.UID = "ce662d9f-34c0-40fd-a4e4-abe4146a1457"
			migrationCache.Add(migration)
			count, targetPod, err := investigateTargetPodSituation(migration, &podList, migrationCache)
			Expect(err).To(BeNil())
			Expect(count).To(Equal(2))
			Expect(targetPod.ObjectMeta.Name).To(Equal(podList.Items[0].ObjectMeta.Name))
		})
	})

	AfterEach(func() {
		server.Close()
	})

})

func handleCreatePod(pod *clientv1.Pod) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
		//TODO: Validate that posted Pod request is sane
		ghttp.RespondWithJSONEncoded(http.StatusOK, pod),
	)
}

func handleCreatePodAuthError() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/api/v1/namespaces/default/pods"),
		//TODO: Validate that posted Pod request is sane
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, ""),
	)
}

func handlePutMigrationAuthError() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, ""),
	)
}

func handleGetPodList(podList clientv1.PodList) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
		ghttp.RespondWithJSONEncoded(http.StatusOK, podList),
	)
}

func handleGetPodListAuthError(podList clientv1.PodList) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/namespaces/default/pods"),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, podList),
	)
}

func handleGetTestVM(expectedVM *v1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
		ghttp.RespondWithJSONEncoded(http.StatusOK, expectedVM),
	)
}

func handleGetTestVMNotFound() http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
		ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
	)
}

func handleGetTestVMAuthError(expectedVM *v1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
		ghttp.RespondWithJSONEncoded(http.StatusForbidden, expectedVM),
	)
}

func mockPod(i int, label string) clientv1.Pod {
	return clientv1.Pod{
		ObjectMeta: clientv1.ObjectMeta{
			Name: "virt-migration" + strconv.Itoa(i),
			Labels: map[string]string{
				v1.DomainLabel:       "testvm",
				v1.AppLabel:          "virt-launcher",
				v1.MigrationUIDLabel: label,
			},
		},
		Status: clientv1.PodStatus{
			Phase: clientv1.PodRunning,
		},
	}
}

func mockMigrationPod(vm *v1.VM) *kubev1.Pod {
	temlateService, err := services.NewTemplateService("whatever")
	Expect(err).ToNot(HaveOccurred())
	pod, err := temlateService.RenderLaunchManifest(vm)
	Expect(err).ToNot(HaveOccurred())
	pod.Spec.NodeName = "targetNode"
	pod.Labels[v1.MigrationLabel] = "testvm-migration"
	return pod
}

func handlePutVM(vm *v1.VM) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/"+vm.ObjectMeta.Name),
		ghttp.VerifyJSONRepresenting(vm),
		ghttp.RespondWithJSONEncoded(http.StatusOK, vm),
	)
}

func TestMigration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Migration")
}