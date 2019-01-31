package imageupload_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	fakecdiclient "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned/fake"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/tests"
)

const (
	commandName             = "image-upload"
	uploadRequestAnnotation = "cdi.kubevirt.io/storage.upload.target"
	podPhaseAnnotation      = "cdi.kubevirt.io/storage.pod.phase"
)

var _ = Describe("ImageUpload", func() {

	const (
		pvcNamespace = "default"
		pvcName      = "test-pvc"
		pvcSize      = "500Mi"
		imagePath    = "../../../vendor/kubevirt.io/containerized-data-importer/tests/images/cirros-qcow2.img"
	)

	var (
		ctrl       *gomock.Controller
		kubeClient *fakek8sclient.Clientset
		cdiClient  *fakecdiclient.Clientset
		server     *httptest.Server

		createCalled bool
		updateCalled bool
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	addPodPhaseAnnotation := func() {
		time.Sleep(10 * time.Millisecond)
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(pvcName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		pvc.Annotations[podPhaseAnnotation] = "Running"
		pvc, err = kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Update(pvc)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error: %v\n", err)
		}
		Expect(err).To(BeNil())
	}

	addReactors := func() {
		kubeClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			pvc, ok := create.GetObject().(*v1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal(pvcName))

			Expect(createCalled).To(BeFalse())
			createCalled = true

			go addPodPhaseAnnotation()

			return false, nil, nil
		})

		kubeClient.Fake.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())

			pvc, ok := update.GetObject().(*v1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal(pvcName))

			if !createCalled && !updateCalled {
				go addPodPhaseAnnotation()
			}

			updateCalled = true

			return false, nil, nil
		})
	}

	validatePVC := func() {
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(pvcName, metav1.GetOptions{})
		Expect(err).To(BeNil())

		resource, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]
		Expect(ok).To(BeTrue())
		Expect(resource.String()).To(Equal(pvcSize))

		_, ok = pvc.Annotations[uploadRequestAnnotation]
		Expect(ok).To(BeTrue())
	}

	createEndpoints := func() *v1.Endpoints {
		return &v1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cdi-upload-" + pvcName,
				Namespace: pvcNamespace,
			},
			Subsets: []v1.EndpointSubset{
				{
					Addresses: []v1.EndpointAddress{
						{
							IP: "10.10.10.10",
						},
					},
				},
			},
		}
	}

	testInit := func(statusCode int, kubeobjects ...runtime.Object) {
		createCalled = false
		updateCalled = false

		objs := append([]runtime.Object{createEndpoints()}, kubeobjects...)

		kubeClient = fakek8sclient.NewSimpleClientset(objs...)
		cdiClient = fakecdiclient.NewSimpleClientset()

		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

		addReactors()

		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		}))
		imageupload.SetHTTPClientCreator(func(bool) *http.Client {
			return server.Client()
		})
	}

	testDone := func() {
		imageupload.SetDefaultHTTPClientCreator()
		server.Close()
	}

	pvcSpec := func() *v1.PersistentVolumeClaim {
		quantity, _ := resource.ParseQuantity(pvcSize)

		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:        pvcName,
				Namespace:   "default",
				Annotations: map[string]string{},
			},
			Spec: v1.PersistentVolumeClaimSpec{
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceStorage: quantity,
					},
				},
			},
		}

		return pvc
	}

	pvcSpecWithUploadAnnotation := func() *v1.PersistentVolumeClaim {
		spec := pvcSpec()
		spec.Annotations = map[string]string{
			uploadRequestAnnotation: "",
			podPhaseAnnotation:      "Running",
		}
		return spec
	}

	pvcSpecWithUploadSucceeded := func() *v1.PersistentVolumeClaim {
		spec := pvcSpec()
		spec.Annotations = map[string]string{
			uploadRequestAnnotation: "",
			podPhaseAnnotation:      "Succeeded",
		}
		return spec
	}

	Context("Successful upload to PVC", func() {
		It("PVC does not exist", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", pvcName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeTrue())
			validatePVC()
		})

		DescribeTable("PVC does exist", func(pvc *v1.PersistentVolumeClaim) {
			testInit(http.StatusOK, pvc)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--no-create", "--pvc-name", pvcName,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeFalse())
			validatePVC()
		},
			Entry("PVC with upload annotation", pvcSpecWithUploadAnnotation()),
			Entry("PVC without upload annotation", pvcSpec()),
		)

		AfterEach(func() {
			testDone()
		})
	})

	Context("Upload fails", func() {
		It("PVC already uploaded", func() {
			testInit(http.StatusOK, pvcSpecWithUploadSucceeded())
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", pvcName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).NotTo(BeNil())
		})

		It("Upload fails", func() {
			testInit(http.StatusInternalServerError)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", pvcName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).NotTo(BeNil())
		})

		DescribeTable("Bad args", func(args []string) {
			testInit(http.StatusOK)
			args = append([]string{commandName}, args...)
			cmd := tests.NewRepeatableVirtctlCommand(args...)
			Expect(cmd()).NotTo(BeNil())
		},
			Entry("No args", []string{}),
			Entry("No args", []string{"--pvc-name", pvcName, "--pvc-size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", imagePath}),
			Entry("No name", []string{"--pvc-size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", imagePath}),
			Entry("No size", []string{"--pvc-name", pvcName, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", imagePath}),
			Entry("No url", []string{"--pvc-name", pvcName, "--pvc-size", pvcSize, "--insecure", "--image-path", imagePath}),
			Entry("No image path", []string{"--pvc-name", pvcName, "--pvc-size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure"}),
		)

		AfterEach(func() {
			testDone()
		})
	})
})
