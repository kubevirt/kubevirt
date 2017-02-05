package watch

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/conversion"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/pkg/runtime/serializer"
	"k8s.io/client-go/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/clientcmd"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"net/http"
)

var _ = Describe("Pod", func() {
	var server *ghttp.Server
	var stopChan chan struct{}

	BeforeEach(func() {
		stopChan = make(chan struct{})
		server = ghttp.NewServer()
	})

	Context("Running Pod for unscheduled VM given", func() {
		It("should update the VM with the node of the running Pod", func(done Done) {

			// Wire a Pod controller with a fake source
			restClient := getRestClient(server.URL())
			vmCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
			lw := framework.NewFakeControllerSource()
			_, podController := NewPodControllerWithListWatch(vmCache, nil, lw, restClient)

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

			// Start the controller
			go podController.Run(1, stopChan)

			// Tell the controller that there is a new running Pod
			lw.Add(pod)

			// Wait until we have proof that a REST call happened
			Eventually(func() int { return len(server.ReceivedRequests()) }).Should(Equal(1))
			close(done)
		}, 10)
	})

	AfterEach(func() {
		close(stopChan)
		server.Close()
	})
})

func getRestClient(url string) *rest.RESTClient {
	restConfig, err := clientcmd.BuildConfigFromFlags(url, "")
	Expect(err).NotTo(HaveOccurred())
	restConfig.GroupVersion = &v1.GroupVersion
	restConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}
	restConfig.APIPath = "/apis"
	restConfig.ContentType = runtime.ContentTypeJSON
	restClient, err := rest.RESTClientFor(restConfig)
	Expect(err).ToNot(HaveOccurred())
	return restClient
}
