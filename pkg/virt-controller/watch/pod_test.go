package watch

import (
	"net/http"

	"github.com/facebookgo/inject"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
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

var _ = Describe("Pod", func() {
	var server *ghttp.Server

	var vmCache cache.Store
	var podCache cache.Store
	var vmService services.VMService
	var dispatch kubecli.ControllerDispatch
	var migrationQueue workqueue.RateLimitingInterface

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		server = ghttp.NewServer()
		restClient, err := kubecli.GetRESTClientFromFlags(server.URL(), "")
		Expect(err).To(Not(HaveOccurred()))
		vmCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		podCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)

		var g inject.Graph
		vmService = services.NewVMService()
		server = ghttp.NewServer()
		config := rest.Config{}
		config.Host = server.URL()
		clientSet, _ := kubernetes.NewForConfig(&config)
		templateService, _ := services.NewTemplateService("kubevirt/virt-launcher")
		restClient, _ = kubecli.GetRESTClientFromFlags(server.URL(), "")
		migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

		g.Provide(
			&inject.Object{Value: restClient},
			&inject.Object{Value: clientSet},
			&inject.Object{Value: vmService},
			&inject.Object{Value: templateService},
		)
		g.Populate()

		dispatch = NewPodControllerDispatch(vmCache, restClient, vmService, clientSet, migrationQueue)

	})

	Context("Running Pod for unscheduled VM given", func() {
		It("should update the VM with the node of the running Pod", func(done Done) {

			// Create a VM which is being scheduled
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = v1.Scheduling
			vm.ObjectMeta.SetUID(uuid.NewUUID())

			// Add the VM to the cache
			vmCache.Add(vm)

			// Create a target Pod for the VM
			temlateService, err := services.NewTemplateService("whatever")
			Expect(err).ToNot(HaveOccurred())
			var pod *kubev1.Pod
			pod, err = temlateService.RenderLaunchManifest(vm)
			Expect(err).ToNot(HaveOccurred())
			pod.Spec.NodeName = "mynode"

			// Create the expected VM after the update
			obj, err := conversion.NewCloner().DeepCopy(vm)
			Expect(err).ToNot(HaveOccurred())

			expectedVM := obj.(*v1.VM)
			expectedVM.Status.Phase = v1.Pending
			expectedVM.Status.NodeName = pod.Spec.NodeName
			expectedVM.ObjectMeta.Labels = map[string]string{v1.NodeNameLabel: pod.Spec.NodeName}

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.VerifyJSONRepresenting(expectedVM),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedVM),
				),
			)

			// Tell the controller that there is a new running Pod

			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			key, _ := cache.MetaNamespaceKeyFunc(pod)
			podCache.Add(pod)
			queue.Add(key)
			dispatch.Execute(podCache, queue, key)

			Expect(len(server.ReceivedRequests())).To(Equal(1))
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

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
				),
			)

			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			key, _ := cache.MetaNamespaceKeyFunc(pod)
			podCache.Add(pod)
			queue.Add(key)
			dispatch.Execute(podCache, queue, key)

			Expect(len(server.ReceivedRequests())).To(Equal(1))
			Expect(migrationQueue.Len()).To(Equal(1))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		server.Close()
	})
})

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

func handleGetNode(node kubev1.Node) http.HandlerFunc {
	return ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/api/v1/nodes/"+node.ObjectMeta.Name),
		ghttp.RespondWithJSONEncoded(http.StatusOK, node),
	)
}
