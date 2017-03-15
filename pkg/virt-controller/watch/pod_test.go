package watch

import (
	"net/http"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/conversion"
	"k8s.io/client-go/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var _ = Describe("Pod", func() {
	var server *ghttp.Server
	var stopChan chan struct{}
	var vmCache cache.Store
	var podController *kubecli.Controller
	var lw *framework.FakeControllerSource
	var ctrl *gomock.Controller
	var mockVMService *services.MockVMService

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		stopChan = make(chan struct{})
		server = ghttp.NewServer()
		// Wire a Pod controller with a fake source
		restClient, err := kubecli.GetRESTClientFromFlags(server.URL(), "")
		Expect(err).To(Not(HaveOccurred()))
		coreClient, err := kubecli.GetFromFlags(server.URL(), "")
		vmCache = cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
		lw = framework.NewFakeControllerSource()
		mockVMService = services.NewMockVMService(ctrl)
		_, podController = NewPodControllerWithListWatch(vmCache, nil, lw, restClient, mockVMService, coreClient)

		// Start the controller
		podController.StartInformer(stopChan)
		go podController.Run(1, stopChan)
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
			pod, err := temlateService.RenderLaunchManifest(vm)
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
			lw.Add(pod)

			// Wait until we have processed the added item
			podController.WaitForSync(stopChan)
			podController.ShutDownQueue()
			podController.WaitUntilDone()

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
				Status: kubev1.NodeStatus{
					Addresses: []kubev1.NodeAddress{srcIp, destIp},
				},
			}
			targetNode = kubev1.Node{
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
			temlateService, err := services.NewTemplateService("whatever")
			Expect(err).ToNot(HaveOccurred())
			pod, err := temlateService.RenderLaunchManifest(vm)
			Expect(err).ToNot(HaveOccurred())
			pod.Spec.NodeName = "targetNode"
			pod.Labels[v1.MigrationLabel] = "testvm-migration"

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
			migration.Status.Phase = v1.MigrationInProgress

			// Register the expected REST call
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations/testvm-migration"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, migration),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.VerifyJSONRepresenting(vmWithMigrationNodeName),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmWithMigrationNodeName),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/nodes/sourceNode"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, srcNode),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/nodes/targetNode"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, targetNode),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/testvm"),
					ghttp.VerifyJSONRepresenting(vmInMigrationState),
					ghttp.RespondWithJSONEncoded(http.StatusOK, vmWithMigrationNodeName),
				),
			)

			mockVMService.EXPECT().GetMigrationJob(gomock.Any()).Return(nil, false, nil)
			mockVMService.EXPECT().StartMigration(gomock.Any(), gomock.Any(), &srcNode, &targetNode).Return(nil)

			// Tell the controller that there is a new running Pod
			lw.Add(pod)

			// Wait until we have processed the added item
			podController.WaitForSync(stopChan)
			podController.ShutDownQueue()
			podController.WaitUntilDone()

			Expect(len(server.ReceivedRequests())).To(Equal(5))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		close(stopChan)
		server.Close()
		ctrl.Finish()
	})
})
