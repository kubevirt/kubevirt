package imageupload_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
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
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/kubevirt/pkg/virtctl/imageupload"
	"kubevirt.io/kubevirt/tests"
)

const (
	commandName                     = "image-upload"
	uploadRequestAnnotation         = "cdi.kubevirt.io/storage.upload.target"
	forceImmediateBindingAnnotation = "cdi.kubevirt.io/storage.bind.immediate.requested"
	podPhaseAnnotation              = "cdi.kubevirt.io/storage.pod.phase"
	podReadyAnnotation              = "cdi.kubevirt.io/storage.pod.ready"
)

const (
	targetNamespace = "default"
	targetName      = "test-volume"
	pvcSize         = "500Mi"
	configName      = "config"
)

var _ = Describe("ImageUpload", func() {

	var (
		ctrl       *gomock.Controller
		kubeClient *fakek8sclient.Clientset
		cdiClient  *fakecdiclient.Clientset
		server     *httptest.Server

		dvCreateCalled  = &atomicBool{lock: &sync.Mutex{}}
		pvcCreateCalled = &atomicBool{lock: &sync.Mutex{}}
		updateCalled    = &atomicBool{lock: &sync.Mutex{}}

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
				Name:        targetName,
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
				Name:      targetName,
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

	dvSpecWithWaitForFirstConsumer := func() *cdiv1.DataVolume {
		dv := &cdiv1.DataVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name:      targetName,
				Namespace: "default",
			},
			TypeMeta: metav1.TypeMeta{
				APIVersion: cdiv1.SchemeGroupVersion.String(),
				Kind:       "DataVolume",
			},
			Status: cdiv1.DataVolumeStatus{Phase: cdiv1.WaitForFirstConsumer},
			Spec: cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{Upload: &cdiv1.DataVolumeSourceUpload{}},
				PVC:    &v1.PersistentVolumeClaimSpec{},
			},
		}
		return dv
	}

	addPodPhaseAnnotation := func() {
		defer GinkgoRecover()
		time.Sleep(10 * time.Millisecond)
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		pvc.Annotations[podPhaseAnnotation] = "Running"
		pvc.Annotations[podReadyAnnotation] = "true"
		pvc, err = kubeClient.CoreV1().PersistentVolumeClaims(targetNamespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error: %v\n", err)
		}
		Expect(err).To(BeNil())
	}

	addDvPhase := func() {
		defer GinkgoRecover()
		time.Sleep(10 * time.Millisecond)
		dv, err := cdiClient.CdiV1beta1().DataVolumes(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		dv.Status.Phase = cdiv1.UploadReady
		dv, err = cdiClient.CdiV1beta1().DataVolumes(targetNamespace).Update(context.Background(), dv, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error: %v\n", err)
		}
		Expect(err).To(BeNil())
	}

	createPVC := func(dv *cdiv1.DataVolume) {
		defer GinkgoRecover()
		time.Sleep(10 * time.Millisecond)
		pvc := pvcSpecWithUploadAnnotation()

		pvc.Spec.VolumeMode = getVolumeMode(dv.Spec)
		pvc.Spec.AccessModes = append([]v1.PersistentVolumeAccessMode(nil), getAccessModes(dv.Spec)...)
		pvc.Spec.StorageClassName = getStorageClassName(dv.Spec)
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(targetNamespace).Create(context.Background(), pvc, metav1.CreateOptions{})
		Expect(err).To(BeNil())
	}

	addReactors := func() {
		cdiClient.Fake.PrependReactor("create", "datavolumes", func(action testing.Action) (bool, runtime.Object, error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			dv, ok := create.GetObject().(*cdiv1.DataVolume)
			Expect(ok).To(BeTrue())
			Expect(dv.Name).To(Equal(targetName))

			Expect(dvCreateCalled.IsTrue()).To(BeFalse())
			dvCreateCalled.True()

			go createPVC(dv)
			go addDvPhase()

			return false, nil, nil
		})

		kubeClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			pvc, ok := create.GetObject().(*v1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal(targetName))

			Expect(pvcCreateCalled.IsTrue()).To(BeFalse())
			pvcCreateCalled.True()

			if !dvCreateCalled.IsTrue() {
				go addPodPhaseAnnotation()
			}

			return false, nil, nil
		})

		kubeClient.Fake.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			update, ok := action.(testing.UpdateAction)
			Expect(ok).To(BeTrue())

			pvc, ok := update.GetObject().(*v1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal(targetName))

			if !dvCreateCalled.IsTrue() && !pvcCreateCalled.IsTrue() && !updateCalled.IsTrue() {
				go addPodPhaseAnnotation()
			}

			updateCalled.True()

			return false, nil, nil
		})
	}

	validateDvStorageSpec := func(spec cdiv1.DataVolumeSpec, mode v1.PersistentVolumeMode) {
		resource, ok := getResourceRequestedStorageSize(spec)

		Expect(ok).To(BeTrue())
		Expect(resource.String()).To(Equal(pvcSize))

		volumeMode := getVolumeMode(spec)
		if volumeMode == nil {
			vm := v1.PersistentVolumeFilesystem
			volumeMode = &vm
		}
		Expect(mode).To(Equal(*volumeMode))
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
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
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

	validateDataVolumeArgs := func(dv *cdiv1.DataVolume, mode v1.PersistentVolumeMode) {
		validateDvStorageSpec(dv.Spec, mode)
	}

	validateDataVolume := func() {
		dv, err := cdiClient.CdiV1beta1().DataVolumes(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(BeNil())

		validateDataVolumeArgs(dv, v1.PersistentVolumeFilesystem)
	}

	validateBlockDataVolume := func() {
		dv, err := cdiClient.CdiV1beta1().DataVolumes(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(BeNil())

		validateDataVolumeArgs(dv, v1.PersistentVolumeBlock)
	}

	validateDataVolumeWithForceBind := func() {
		dv, err := cdiClient.CdiV1beta1().DataVolumes(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		_, ok := dv.Annotations[forceImmediateBindingAnnotation]
		Expect(ok).To(BeTrue(), "storage.bind.immediate.requested annotation")

		validateDataVolumeArgs(dv, v1.PersistentVolumeFilesystem)
	}

	expectedStorageClassMatchesActual := func(storageClass string) {
		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(targetNamespace).Get(context.Background(), targetName, metav1.GetOptions{})
		Expect(err).To(BeNil())
		_, ok := pvc.Annotations[uploadRequestAnnotation]
		Expect(ok).To(BeTrue())
		Expect(storageClass).To(Equal(*pvc.Spec.StorageClassName))
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
		_, err := cdiClient.CdiV1beta1().CDIConfigs().Update(context.Background(), config, metav1.UpdateOptions{})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Error: %v\n", err)
		}
		Expect(err).To(BeNil())
	}

	waitProcessingComplete := func(client kubernetes.Interface, namespace, name string, interval, timeout time.Duration) error {
		return nil
	}

	testInitAsyncWithCdiObjects := func(statusCode int, async bool, kubeobjects []runtime.Object, cdiobjects []runtime.Object) {
		dvCreateCalled.False()
		pvcCreateCalled.False()
		updateCalled.False()

		config := createCDIConfig()
		cdiobjects = append(cdiobjects, config)

		kubeClient = fakek8sclient.NewSimpleClientset(kubeobjects...)
		cdiClient = fakecdiclient.NewSimpleClientset(cdiobjects...)

		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().StorageV1().Return(kubeClient.StorageV1()).AnyTimes()

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

	testInitAsync := func(statusCode int, async bool, kubeobjects ...runtime.Object) {
		testInitAsyncWithCdiObjects(statusCode, async, kubeobjects, nil)
	}

	testInit := func(statusCode int, kubeobjects ...runtime.Object) {
		testInitAsync(statusCode, true, kubeobjects...)
	}

	testDone := func() {
		imageupload.SetDefaultHTTPClientCreator()
		server.Close()
	}

	Context("Successful upload to PVC", func() {

		It("PVC does not exist deprecated args", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", targetName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(pvcCreateCalled.IsTrue()).To(BeTrue())
			validatePVC()
		})

		It("PVC exists deprecated args", func() {
			testInit(http.StatusOK, pvcSpec())
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "--pvc-name", targetName, "--no-create",
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(pvcCreateCalled.IsTrue()).To(BeFalse())
			validatePVC()
		})

		DescribeTable("DV does not exist", func(async bool) {
			testInitAsync(http.StatusOK, async)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(dvCreateCalled.IsTrue()).To(BeTrue())
			validatePVC()
			validateDataVolume()
		},
			Entry("DV does not exist, async", true),
			Entry("DV does not exist sync", false),
		)

		It("DV does not exist --pvc-size", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(dvCreateCalled.IsTrue()).To(BeTrue())
			validatePVC()
			validateDataVolume()
		})

		It("DV does not exist --force-bind", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath, "--force-bind")
			Expect(cmd()).To(BeNil())
			Expect(dvCreateCalled.IsTrue()).To(BeTrue())
			validatePVC()
			validateDataVolumeWithForceBind()
		})

		It("DV does not exist and --no-create", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--pvc-size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath, "--no-create")
			err := cmd()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(fmt.Sprintf("persistentvolumeclaims %q not found", targetName)))
			Expect(dvCreateCalled.IsTrue()).To(BeFalse())
		})

		It("Use CDI Config UploadProxyURL", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(dvCreateCalled.IsTrue()).To(BeTrue())
			validatePVC()
			validateDataVolume()
		})

		It("Create a VolumeMode=Block PVC", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath, "--block-volume")
			Expect(cmd()).To(BeNil())
			Expect(dvCreateCalled.IsTrue()).To(BeTrue())
			validateBlockPVC()
			validateBlockDataVolume()
		})

		It("Create a non-default storage class PVC", func() {
			testInit(http.StatusOK)
			expectedStorageClass := "non-default-sc"
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath, "--storage-class", expectedStorageClass)
			Expect(cmd()).To(BeNil())
			Expect(dvCreateCalled.IsTrue()).To(BeTrue())
			expectedStorageClassMatchesActual(expectedStorageClass)
		})

		DescribeTable("PVC does not exist", func(async bool) {
			testInitAsync(http.StatusOK, async)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "pvc", targetName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(pvcCreateCalled.IsTrue()).To(BeTrue())
			validatePVC()
		},
			Entry("PVC does not exist, async", true),
			Entry("PVC does not exist sync", false),
		)

		DescribeTable("PVC does exist", func(pvc *v1.PersistentVolumeClaim) {
			testInit(http.StatusOK, pvc)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "pvc", targetName,
				"--uploadproxy-url", server.URL, "--no-create", "--insecure", "--image-path", imagePath)
			Expect(cmd()).To(BeNil())
			Expect(pvcCreateCalled.IsTrue()).To(BeFalse())
			validatePVC()
		},
			Entry("PVC with upload annotation", pvcSpecWithUploadAnnotation()),
			Entry("PVC without upload annotation", pvcSpec()),
			Entry("PVC without upload annotation and no annotation map", pvcSpecNoAnnotationMap()),
		)

		It("Show error when uploading to ReadOnly volume", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath, "--access-mode", string(v1.ReadOnlyMany))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("cannot upload to a readonly volume, use either ReadWriteOnce or ReadWriteMany if supported"))
			Expect(dvCreateCalled.IsTrue()).To(BeFalse())
		})

		AfterEach(func() {
			testDone()
		})
	})

	Context("Upload fails", func() {
		It("PVC already uploaded", func() {
			testInit(http.StatusOK, pvcSpecWithUploadSucceeded())
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			err := cmd()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).Should(ContainSubstring("No DataVolume is associated with the existing PVC"))
		})

		It("DV in phase WaitForFirstConsumer", func() {
			testInitAsyncWithCdiObjects(
				http.StatusOK,
				true,
				[]runtime.Object{pvcSpecWithUploadAnnotation()},
				[]runtime.Object{dvSpecWithWaitForFirstConsumer()},
			)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--uploadproxy-url", server.URL, "--insecure", "--image-path", imagePath)
			err := cmd()
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).Should(ContainSubstring("cannot upload to DataVolume in WaitForFirstConsumer state"))
		})

		It("uploadProxyURL not configured", func() {
			testInit(http.StatusOK)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
				"--insecure", "--image-path", imagePath)
			config, err := cdiClient.CdiV1beta1().CDIConfigs().Get(context.Background(), configName, metav1.GetOptions{})
			Expect(err).To(BeNil())
			config.Status.UploadProxyURL = nil
			updateCDIConfig(config)
			Expect(cmd()).NotTo(BeNil())
		})

		It("Upload fails", func() {
			testInit(http.StatusInternalServerError)
			cmd := tests.NewRepeatableVirtctlCommand(commandName, "dv", targetName, "--size", pvcSize,
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
				[]string{"targetName", "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("No name", "expecting two args",
				[]string{"--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("No size", "when creating a resource, the size must be specified",
				[]string{"dv", targetName, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("Size invalid", "validation failed for size=500Zb: quantities must match the regular expression '^([+-]?[0-9.]+)([eEinumkKMGTP]*[-+]?[0-9]*)$'",
				[]string{"dv", targetName, "--size", "500Zb", "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("No image path", "required flag(s) \"image-path\" not set",
				[]string{"dv", targetName, "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure"}),
			Entry("PVC name and args", "cannot use --pvc-name and args",
				[]string{"foo", "--pvc-name", targetName, "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("Unexpected resource type", "invalid resource type foo",
				[]string{"foo", targetName, "--size", pvcSize, "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
			Entry("Size twice", "--pvc-size deprecated, use --size",
				[]string{"dv", targetName, "--size", "500G", "--pvc-size", "50G", "--uploadproxy-url", "https://doesnotexist", "--insecure", "--image-path", "/dev/null"}),
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

func getResourceRequestedStorageSize(dvSpec cdiv1.DataVolumeSpec) (resource.Quantity, bool) {
	if dvSpec.PVC != nil {
		resource, ok := dvSpec.PVC.Resources.Requests[v1.ResourceStorage]
		return resource, ok
	}
	resource, ok := dvSpec.Storage.Resources.Requests[v1.ResourceStorage]
	return resource, ok
}

func getStorageClassName(dvSpec cdiv1.DataVolumeSpec) *string {
	if dvSpec.PVC != nil {
		return dvSpec.PVC.StorageClassName
	}
	return dvSpec.Storage.StorageClassName
}

func getAccessModes(dvSpec cdiv1.DataVolumeSpec) []v1.PersistentVolumeAccessMode {
	if dvSpec.PVC != nil {
		return dvSpec.PVC.AccessModes
	}
	return dvSpec.Storage.AccessModes
}

func getVolumeMode(dvSpec cdiv1.DataVolumeSpec) *v1.PersistentVolumeMode {
	if dvSpec.PVC != nil {
		return dvSpec.PVC.VolumeMode
	}
	return dvSpec.Storage.VolumeMode
}

type atomicBool struct {
	lock  *sync.Mutex
	value bool
}

func (b *atomicBool) IsTrue() bool {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.value
}

func (b *atomicBool) True() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.value = true
}

func (b *atomicBool) False() {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.value = false
}
