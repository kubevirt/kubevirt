package imageupload_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	fakecdiclient "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/tests"
)

const (
	commandName             = "image-upload"
	uploadRequestAnnotation = "cdi.kubevirt.io/storage.upload.target"
	podPhaseAnnotation      = "cdi.kubevirt.io/storage.pod.phase"
	podReadyAnnotation      = "cdi.kubevirt.io/storage.pod.ready"
)

const (
	dvNamespace = "default"
	dvName      = "test-dv"
	pvcSize     = "500Mi"
	configName  = "config"
)

var _ = Describe("ImageUpload", func() {

	var (
		ctrl       *gomock.Controller
		kubeClient *fakek8sclient.Clientset
		cdiClient  *fakecdiclient.Clientset
		server     *httptest.Server

		createCalled bool
		updateCalled bool

		imagePath string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		imageFile, err := ioutil.TempFile("", "test_image")
		Expect(err).ToNot(HaveOccurred())

		_, err = imageFile.Write([]byte("hello world"))
		Expect(err).ToNot(HaveOccurred())
		defer imageFile.Close()

		imagePath = imageFile.Name()
	})

	AfterEach(func() {
		ctrl.Finish()
		os.Remove(imagePath)
	})

	pvcSpec := func() *v1.PersistentVolumeClaim {
		quantity, _ := resource.ParseQuantity(pvcSize)

		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:        dvName,
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

	pvcSpecNoAnnotationMap := func() *v1.PersistentVolumeClaim {
		quantity, _ := resource.ParseQuantity(pvcSize)

		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dvName,
				Namespace: "default",
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
			podReadyAnnotation:      "true",
		}
		return spec
	}

	pvcSpecWithUploadSucceeded := func() *v1.PersistentVolumeClaim {
		spec := pvcSpec()
		spec.Annotations = map[string]string{
			uploadRequestAnnotation: "",
			podPhaseAnnotation:      "Succeeded",
			podReadyAnnotation:      "false",
		}
		return spec
	}

	addPodPhaseAnnotation := func() {
		defer GinkgoRecover()
		time.Sleep(10 * time.Millisecond)
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(dvNamespace).Get(dvName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		pvc.Annotations[podPhaseAnnotation] = "Running"
		pvc.Annotations[podReadyAnnotation] = "true"
		pvc, err = kubeClient.CoreV1().PersistentVolumeClaims(dvNamespace).Update(pvc)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error: %v\n", err)
		}
		Expect(err).To(BeNil())
	}

	createPVC := func(dv *cdiv1.DataVolume) {
		defer GinkgoRecover()
		time.Sleep(10 * time.Millisecond)
		pvc := pvcSpecWithUploadAnnotation()
		pvc.Spec.VolumeMode = dv.Spec.PVC.VolumeMode
		pvc.Spec.AccessModes = append([]v1.PersistentVolumeAccessMode(nil), dv.Spec.PVC.AccessModes...)
		pvc.Spec.StorageClassName = dv.Spec.PVC.StorageClassName
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(dvNamespace).Create(pvc)
		Expect(err).To(BeNil())
	}

	addReactors := func() {
		cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (bool, runtime.Object, error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			dv, ok := create.GetObject().(*cdiv1.DataVolume)
			Expect(ok).To(BeTrue())
			Expect(dv.Name).To(Equal(dvName))

			Expect(createCalled).To(BeFalse())
			createCalled = true

			go createPVC(dv)

			return false, nil, nil
		})

		kubeClient.Fake.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())

			pvc, ok := update.GetObject().(*v1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal(dvName))

			if !createCalled && !updateCalled {
				go addPodPhaseAnnotation()
			}

			updateCalled = true

			return false, nil, nil
		})
	}

	validatePVCSpec := func(spec *v1.PersistentVolumeClaimSpec, mode v1.PersistentVolumeMode) {
		resource, ok := spec.Resources.Requests[v1.ResourceStorage]
		Expect(ok).To(BeTrue())
		Expect(resource.String()).To(Equal(pvcSize))

		volumeMode := spec.VolumeMode
		if volumeMode == nil {
			vm := v1.PersistentVolumeFilesystem
			volumeMode = &vm
		}
		Expect(mode).To(Equal(*volumeMode))
	}

	validatePVCArgs := func(mode v1.PersistentVolumeMode) {
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(dvNamespace).Get(dvName, metav1.GetOptions{})
		Expect(err).To(BeNil())

		_, ok := pvc.Annotations[uploadRequestAnnotation]
		Expect(ok).To(BeTrue())

		validatePVCSpec(&pvc.Spec, mode)
	}

	validatePVC := func() {
		validatePVCArgs(v1.PersistentVolumeFilesystem)
	}

	validateBlockPVC := func() {
		validatePVCArgs(v1.PersistentVolumeBlock)
	}

	validateDataVolumeArgs := func(mode v1.PersistentVolumeMode) {
		dv, err := cdiClient.CdiV1alpha1().DataVolumes(dvNamespace).Get(dvName, metav1.GetOptions{})
		Expect(err).To(BeNil())

		validatePVCSpec(dv.Spec.PVC, mode)
	}

	validateDataVolume := func() {
		validateDataVolumeArgs(v1.PersistentVolumeFilesystem)
	}

	validateBlockDataVolume := func() {
		validateDataVolumeArgs(v1.PersistentVolumeBlock)
	}

	createCDIConfig := func() *cdiv1.CDIConfig {
		return &cdiv1.CDIConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: configName,
			},
			Spec: cdiv1.CDIConfigSpec{
				UploadProxyURLOverride: nil,
			},
			Status: cdiv1.CDIConfigStatus{
				UploadProxyURL: nil,
			},
		}
	}

	updateCDIConfig := func(config *cdiv1.CDIConfig) {
		config, err := cdiClient.CdiV1alpha1().CDIConfigs().Update(config)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error: %v\n", err)
		}
		Expect(err).To(BeNil())
	}

	waitProcessingComplete := func(client kubernetes.Interface, namespace, name string, interval, timeout time.Duration) error {
		return nil
	}

	testInitAsync := func(statusCode int, async bool, kubeobjects ...runtime.Object) {
		createCalled = false
		updateCalled = false

		config := createCDIConfig()

		kubeClient = fakek8sclient.NewSimpleClientset(kubeobjects...)
		cdiClient = fakecdiclient.NewSimpleClientset(config)

		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient).AnyTimes()

		addReactors()

		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "HEAD" {
				if async {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
				return
			}
			w.WriteHeader(statusCode)
		}))
		config.Status.UploadProxyURL = &server.URL
		updateCDIConfig(config)

		imageupload.UploadProcessingCompleteFunc = waitProcessingComplete
		imageupload.SetHTTPClientCreator(func(bool) *http.Client {
			return server.Client()
		})
	}

	testInit := func(statusCode int, kubeobjects ...runtime.Object) {
		testInitAsync(statusCode, true, kubeobjects...)
	}

	testDone := func() {
		imageupload.SetDefaultHTTPClientCreator()
		server.Close()
	}

	Context("Successful upload to PVC", func() {
		DescribeTable("PVC does exist", func(async bool) {
			testInitAsync(http.StatusOK, async)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeTrue())
			validatePVC()
			validateDataVolume()
		},
			Entry("PVC does not exist, async", true),
			Entry("PVC does not exist sync", false),
		)

		It("PVC does not exist --pcvc-size", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeTrue())
			validatePVC()
			validateDataVolume()
		})

		It("PVC does not exist deprecated args", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", dvName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeTrue())
			validatePVC()
			validateDataVolume()
		})

		It("Use CDI Config UploadProxyURL", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeTrue())
			validatePVC()
			validateDataVolume()
		})

		It("Create a VolumeMode=Block PVC", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath, "--block-volume")
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeTrue())
			validateBlockPVC()
			validateBlockDataVolume()
		})

		DescribeTable("PVC does exist", func(pvc *v1.PersistentVolumeClaim) {
			testInit(http.StatusOK, pvc)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "pvc", dvName,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeFalse())
			validatePVC()
		},
			Entry("PVC with upload annotation", pvcSpecWithUploadAnnotation()),
			Entry("PVC without upload annotation", pvcSpec()),
			Entry("PVC without upload annotation and no annotation map", pvcSpecNoAnnotationMap()),
		)

		It("PVC exists deprecated args", func() {
			testInit(http.StatusOK, pvcSpec())
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", dvName, "--no-create",
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(createCalled).To(BeFalse())
			validatePVC()
		})

		AfterEach(func() {
			testDone()
		})
	})

	Context("Upload fails", func() {
		It("PVC already uploaded", func() {
			testInit(http.StatusOK, pvcSpecWithUploadSucceeded())
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).NotTo(BeNil())
		})

		It("uploadProxyURL not configured", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath)
			config, err := cdiClient.CdiV1alpha1().CDIConfigs().Get(configName, metav1.GetOptions{})
			Expect(err).To(BeNil())
			config.Status.UploadProxyURL = nil
			updateCDIConfig(config)
			Expect(cmd()).NotTo(BeNil())
		})

		It("Upload fails", func() {
			testInit(http.StatusInternalServerError)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", dvName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).NotTo(BeNil())
		})

		DescribeTable("Bad args", func(errString string, args []string) {
			testInit(http.StatusOK)
			args = append([]string{commandName}, args...)
			cmd := tests.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).Should(Equal(errString))
		},
			Entry("No args", "required flag(s) \"image-path\" not set", []string{}),
			Entry("Missing arg", "expecting two args",
				[]string{"dvName", "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("No name", "expecting two args",
				[]string{"--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("No size", "when creating DataVolume, the size must be specified",
				[]string{"dv", dvName, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("Size invalid", "validation failed for size=500Zb: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'",
				[]string{"dv", dvName, "--size", "500Zb", "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("No image path", "required flag(s) \"image-path\" not set",
				[]string{"dv", dvName, "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure"}),
			Entry("PVC name and args", "cannot use --pvc-name and args",
				[]string{"foo", "--pvc-name", dvName, "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("Unexpected resource type", "invalid resource type foo",
				[]string{"foo", dvName, "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("Size twice", "--pvc-size deprecated, use --size",
				[]string{"dv", dvName, "--size", "500G", "--pvc-size", "50G", "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
		)

		AfterEach(func() {
			testDone()
		})
	})

	Context("URL validation", func() {
		serverURL := "http://localhost:12345"
		DescribeTable("Server URL validations", func(serverUrl string, expected string) {
			path, err := imageupload.ConstructUploadProxyPath(serverUrl)
			Expect(err).To(BeNil())
			Expect(strings.Compare(path, expected)).To(BeZero())
		},
			Entry("Server URL with trailing slash should pass", serverURL+"/", serverURL+imageupload.UploadProxyURI),
			Entry("Server URL with URI should pass", serverURL+imageupload.UploadProxyURI, serverURL+imageupload.UploadProxyURI),
			Entry("Server URL only should pass", serverURL, serverURL+imageupload.UploadProxyURI),
		)
	})
})
