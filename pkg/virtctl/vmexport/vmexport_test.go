package vmexport_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/utils"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	. "kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	commandName      = "vmexport"
	defaultNamespace = "default"
	vmexportName     = "test-vme"
	volumeName       = "test-volume"
	secretName       = "secret-test-vme"
)

var _ = Describe("vmexport", func() {
	var (
		ctrl           *gomock.Controller
		kubeClient     *fakek8sclient.Clientset
		vmExportClient *kubevirtfake.Clientset
		server         *httptest.Server
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)

		kubeClient = fakek8sclient.NewSimpleClientset()
		vmExportClient = kubevirtfake.NewSimpleClientset()

	})

	setflag := func(flag, parameter string) string {
		return fmt.Sprintf("%s=%s", flag, parameter)
	}

	vmexportSpec := func(name, namespace, kind, resourceName string) *exportv1.VirtualMachineExport {
		tokenSecretRef := secretName
		vmexport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: &tokenSecretRef,
				Source: v1.TypedLocalObjectReference{
					APIGroup: &v1.SchemeGroupVersion.Group,
					Kind:     kind,
					Name:     resourceName,
				},
			},
		}

		return vmexport
	}

	getVMEStatus := func(volumes []exportv1.VirtualMachineExportVolume) *exportv1.VirtualMachineExportStatus {
		tokenSecretRef := secretName
		// Mock the expected vme status
		return &exportv1.VirtualMachineExportStatus{
			Phase: exportv1.Ready,
			Links: &exportv1.VirtualMachineExportLinks{
				External: &exportv1.VirtualMachineExportLink{
					Volumes: volumes,
				},
			},
			TokenSecretRef: &tokenSecretRef,
		}
	}

	getExportVolumeFormat := func(url string, format exportv1.ExportVolumeFormat) []exportv1.VirtualMachineExportVolumeFormat {
		return []exportv1.VirtualMachineExportVolumeFormat{
			{
				Format: format,
				Url:    url,
			},
		}
	}

	waitExportCompleteDefault := func(kubecli.KubevirtClient, *VMExportInfo, time.Duration, time.Duration) error {
		return nil
	}

	waitExportCompleteError := func(kubecli.KubevirtClient, *VMExportInfo, time.Duration, time.Duration) error {
		return fmt.Errorf("Processing failed: Test error")
	}

	addDefaultReactors := func() {
		vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			vmExport, ok := create.GetObject().(*exportv1.VirtualMachineExport)
			Expect(ok).To(BeTrue())
			return true, vmExport, nil
		})

		kubeClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			secret, ok := create.GetObject().(*v1.Secret)
			Expect(ok).To(BeTrue())
			return true, secret, nil
		})
	}

	testInit := func(statusCode int) {
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().StorageV1().Return(kubeClient.StorageV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineExport(defaultNamespace).Return(vmExportClient.ExportV1alpha1().VirtualMachineExports(defaultNamespace)).AnyTimes()

		addDefaultReactors()

		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		}))

		vmexport.ExportProcessingComplete = waitExportCompleteDefault
		vmexport.SetHTTPClientCreator(func(*http.Transport, bool) *http.Client {
			return server.Client()
		})
		virtctlvmexport.SetPortForwarder(func(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}) error {
			readyChan <- struct{}{}
			return nil
		})
	}

	testDone := func() {
		virtctlvmexport.SetDefaultHTTPClientCreator()
		virtctlvmexport.SetDefaultPortForwarder()
		server.Close()
	}

	Context("VMExport fails", func() {
		It("VirtualMachineExport already exists when using 'create'", func() {
			testInit(http.StatusOK)
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			expectVMExportGet(vmExportClient, vmexport)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, CREATE, vmexportName, setflag(PVC_FLAG, "test-pvc"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("VirtualMachineExport 'default/test-vme' already exists"))
		})

		It("VirtualMachineExport doesn't exist when using 'download' without source type", func() {
			testInit(http.StatusOK)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "output.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("Unable to get 'default/test-vme' VirtualMachineExport"))
		})

		It("VirtualMachineExport processing fails when using 'download'", func() {
			testInit(http.StatusOK)
			vmexport.ExportProcessingComplete = waitExportCompleteError
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(PVC_FLAG, "test-pvc"), setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("Processing failed: Test error"))
		})

		It("VirtualMachineExport download fails when there's no volume available", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus(nil)
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to access the volume info from '%s/%s' VirtualMachineExport", defaultNamespace, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the volumes have a different name than expected", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
				{
					Name:    "no-test-volume-2",
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", defaultNamespace, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when there are multiple volumes and no volume name has been specified", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
				{
					Name:    "test-volume",
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", defaultNamespace, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when no format is available", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{{Name: volumeName}})
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", defaultNamespace, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the only available format is incompatible with download", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.Dir),
				},
			})
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", defaultNamespace, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the secret token is not attainable", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectVMExportGet(vmExportClient, vme)
			// Adding a new reactor so the client returns a nil secret
			kubeClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) { return false, nil, nil })

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("secrets \"%s\" not found", secretName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails if the server returns a bad status", func() {
			testInit(http.StatusInternalServerError)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectVMExportGet(vmExportClient, vme)
			expectSecretGet(kubeClient)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("Bad status: 500 Internal Server Error"))
		})

		It("Bad flag combination", func() {
			testInit(http.StatusOK)
			args := []string{CREATE, vmexportName, setflag(PVC_FLAG, "test"), setflag(VM_FLAG, "test2"), setflag(SNAPSHOT_FLAG, "test3")}
			args = append([]string{commandName}, args...)
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("if any flags in the group [vm snapshot pvc] are set none of the others can be; [pvc snapshot vm] were all set"))
		})

		DescribeTable("Invalid arguments/flags", func(errString string, args []string) {
			testInit(http.StatusOK)
			args = append([]string{commandName}, args...)
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal(errString))
		},
			Entry("No arguments", "argument validation failed"),
			Entry("Missing arg", "argument validation failed", virtctlvmexport.CREATE, setflag(virtctlvmexport.PVC_FLAG, vmexportName)),
			Entry("More arguments than expected create", "argument validation failed", virtctlvmexport.CREATE, virtctlvmexport.DELETE, vmexportName),
			Entry("Using 'create' without export type", virtctlvmexport.ErrRequiredExportType, virtctlvmexport.CREATE, vmexportName),
			Entry("Using 'create' with invalid flag", fmt.Sprintf(virtctlvmexport.ErrIncompatibleFlag, virtctlvmexport.INSECURE_FLAG, virtctlvmexport.CREATE), virtctlvmexport.CREATE, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test"), virtctlvmexport.INSECURE_FLAG),
			Entry("Using 'delete' with export type", virtctlvmexport.ErrIncompatibleExportType, virtctlvmexport.DELETE, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test")),
			Entry("Using 'delete' with invalid flag", fmt.Sprintf(virtctlvmexport.ErrIncompatibleFlag, virtctlvmexport.INSECURE_FLAG, virtctlvmexport.DELETE), virtctlvmexport.DELETE, vmexportName, virtctlvmexport.INSECURE_FLAG),
			Entry("More arguments than expected download and 'manifest'", "argument validation failed", virtctlvmexport.DOWNLOAD, virtctlvmexport.DELETE, virtctlvmexport.MANIFEST_FLAG, vmexportName),
			Entry("Using 'manifest' with pvc flag", fmt.Sprintf(virtctlvmexport.ErrIncompatibleFlag, virtctlvmexport.PVC_FLAG, virtctlvmexport.MANIFEST_FLAG), virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.MANIFEST_FLAG, setflag(virtctlvmexport.PVC_FLAG, "test")),
			Entry("Using 'manifest' with volume type", fmt.Sprintf(virtctlvmexport.ErrIncompatibleFlag, virtctlvmexport.VOLUME_FLAG, virtctlvmexport.MANIFEST_FLAG), virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.MANIFEST_FLAG, setflag(virtctlvmexport.VM_FLAG, "test"), setflag(virtctlvmexport.VOLUME_FLAG, "volume")),
			Entry("Using 'manifest' with invalid output_format_flag", fmt.Sprintf(virtctlvmexport.ErrInvalidValue, virtctlvmexport.OUTPUT_FORMAT_FLAG, "json/yaml"), virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.MANIFEST_FLAG, setflag(virtctlvmexport.OUTPUT_FORMAT_FLAG, "invalid")),
			Entry("Using 'port-forward' with invalid port", fmt.Sprintf(virtctlvmexport.ErrInvalidValue, virtctlvmexport.PORT_FORWARD_FLAG, "valid port numbers"), virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.PORT_FORWARD_FLAG, setflag(virtctlvmexport.PORT_FORWARD_FLAG, "test")),
		)

		AfterEach(func() {
			testDone()
		})
	})

	Context("VMExport succeeds", func() {
		BeforeEach(func() {
			testInit(http.StatusOK)
		})

		// Create tests
		It("VirtualMachineExport is created succesfully", func() {
			expectVMExportCreate(vmExportClient, nil)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, CREATE, vmexportName, setflag(PVC_FLAG, "test-pvc"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		// Delete tests
		It("VirtualMachineExport is deleted succesfully", func() {
			utils.HandleVMExportDelete(vmExportClient, vmexportName)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DELETE, vmexportName)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport doesn't exist when using 'delete'", func() {
			testInit(http.StatusOK)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DELETE, vmexportName)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		// Download tests
		It("Succesfully download from an already existing VirtualMachineExport", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			})
			expectSecretGet(kubeClient)
			expectVMExportGet(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create and download a VirtualMachineExport", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			})
			expectSecretGet(kubeClient)
			expectVMExportCreate(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(PVC_FLAG, "test-pvc"), setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully download a VirtualMachineExport with just 'raw' links", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectSecretGet(kubeClient)
			expectVMExportGet(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when the volume has a different name than expected but there's only one volume", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectSecretGet(kubeClient)
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when there's only one volume and no --volume has been specified", func() {
			testInit(http.StatusOK)
			vme := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vme.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			})
			expectSecretGet(kubeClient)
			expectVMExportGet(vmExportClient, vme)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create VirtualMachineExport with TTL", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			})
			expectSecretGet(kubeClient)
			vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				Expect(*vme.Spec.TTLDuration).To(Equal(metav1.Duration{Duration: time.Minute}))

				return true, vme, nil
			})

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, CREATE, vmexportName, setflag(PVC_FLAG, "test-pvc"), setflag(TTL_FLAG, "1m"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			testDone()
		})
	})

	Context("getUrlFromVirtualMachineExport", func() {
		// Mocking the minimum viable VMExportInfo struct
		vmeinfo := &VMExportInfo{
			Name:       vmexportName,
			VolumeName: volumeName,
		}

		It("Should get compressed URL even when there's multiple URLs", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name: volumeName,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{
						{
							Format: exportv1.KubeVirtRaw,
							Url:    "raw",
						},
						{
							Format: exportv1.KubeVirtRaw,
							Url:    "raw",
						},
						{
							Format: exportv1.KubeVirtGz,
							Url:    "compressed",
						},
						{
							Format: exportv1.KubeVirtRaw,
							Url:    "raw",
						},
					},
				},
			})
			url, err := GetUrlFromVirtualMachineExport(vmexport, vmeinfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("compressed"))
		})

		It("Should get raw URL when there's no other option", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat("raw", exportv1.KubeVirtRaw),
				},
			})
			url, err := GetUrlFromVirtualMachineExport(vmexport, vmeinfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("raw"))
		})

		It("Should not get any URL when there's no valid options", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			vmexport.Status = getVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: getExportVolumeFormat(server.URL, exportv1.Dir),
				},
			})
			url, err := GetUrlFromVirtualMachineExport(vmexport, vmeinfo)
			Expect(err).To(HaveOccurred())
			Expect(url).To(Equal(""))
		})

		AfterEach(func() {
			testDone()
		})
	})

	Context("Manifest", func() {
		var (
			orgHttpFunc virtctlvmexport.HandleHTTPRequestFunc
		)

		BeforeEach(func() {
			orgHttpFunc = virtctlvmexport.HandleHTTPRequest
			testInit(http.StatusOK)
		})

		AfterEach(func() {
			virtctlvmexport.HandleHTTPRequest = orgHttpFunc
			testDone()
		})

		DescribeTable("should successfully create VirtualMachineExport if proper arg supplied", func(arg, headerValue string) {
			virtctlvmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(Equal(manifestUrl))
				Expect(headers).To(ContainElements(headerValue))
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}
				return &resp, nil
			}
			vmexport := utils.VMExportSpecVM(vmexportName, metav1.NamespaceDefault, "test", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vmexport.Status.Links.External.Manifests = append(vmexport.Status.Links.External.Manifests, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AllManifests,
				Url:  manifestUrl,
			})
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)
			utils.HandleSecretGet(kubeClient, secretName)
			vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())

				return true, vme, nil
			})
			args := []string{commandName, virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.MANIFEST_FLAG, setflag(virtctlvmexport.VM_FLAG, "test")}
			if arg != "" {
				args = append(args, arg)
			}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("no output format arguments", "", virtctlvmexport.APPLICATION_YAML),
			Entry("output format json", setflag(virtctlvmexport.OUTPUT_FORMAT_FLAG, virtctlvmexport.OUTPUT_FORMAT_JSON), virtctlvmexport.APPLICATION_JSON),
			Entry("output format yaml", setflag(virtctlvmexport.OUTPUT_FORMAT_FLAG, virtctlvmexport.OUTPUT_FORMAT_YAML), virtctlvmexport.APPLICATION_YAML),
		)

		DescribeTable("should call both manifest and secret url if argument supplied", func(arg string, callCount int) {
			calls := 0
			virtctlvmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(BeElementOf(manifestUrl, secretUrl))
				calls++
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}
				return &resp, nil
			}
			vmexport := utils.VMExportSpecVM(vmexportName, metav1.NamespaceDefault, "test", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vmexport.Status.Links.External.Manifests = append(vmexport.Status.Links.External.Manifests, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AllManifests,
				Url:  manifestUrl,
			}, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AuthHeader,
				Url:  secretUrl,
			})
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)
			utils.HandleSecretGet(kubeClient, secretName)
			vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())

				return true, vme, nil
			})
			args := []string{commandName, virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.MANIFEST_FLAG, setflag(virtctlvmexport.VM_FLAG, "test")}
			if arg != "" {
				args = append(args, arg)
			}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
			Expect(calls).To(Equal(callCount))
		},
			Entry("without --include-secret", "", 1),
			Entry("with --include-secret", virtctlvmexport.INCLUDE_SECRET_FLAG, 2),
		)

		It("should error if http status is error", func() {
			virtctlvmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(BeElementOf(manifestUrl, secretUrl))
				resp := http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("")),
				}
				return &resp, nil
			}
			vmexport := utils.VMExportSpecVM(vmexportName, metav1.NamespaceDefault, "test", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vmexport.Status.Links.External.Manifests = append(vmexport.Status.Links.External.Manifests, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AllManifests,
				Url:  manifestUrl,
			}, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AuthHeader,
				Url:  secretUrl,
			})
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)
			utils.HandleSecretGet(kubeClient, secretName)
			vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())

				return true, vme, nil
			})
			args := []string{commandName, virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.MANIFEST_FLAG, setflag(virtctlvmexport.VM_FLAG, "test")}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Port-forward", func() {
		var (
			orgHttpFunc virtctlvmexport.HandleHTTPRequestFunc
			vme         *exportv1.VirtualMachineExport
		)

		BeforeEach(func() {
			orgHttpFunc = virtctlvmexport.HandleHTTPRequest
			testInit(http.StatusOK)
			vme = utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)
		})

		AfterEach(func() {
			virtctlvmexport.HandleHTTPRequest = orgHttpFunc
			vme = nil
			testDone()
		})

		It("VirtualMachineExport download fails when using port-forward with an invalid port", func() {
			utils.HandleServiceGet(kubeClient, fmt.Sprintf("virt-export-%s", vme.Name), 321)
			utils.HandlePodList(kubeClient, fmt.Sprintf("virt-export-pod-%s", vme.Name))
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.PORT_FORWARD_FLAG, "5432"), setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("Service virt-export-test-vme does not have a service port 443"))
		})

		It("VirtualMachineExport download with port-forward fails when the service doesn't have a valid pod ", func() {
			utils.HandleServiceGet(kubeClient, fmt.Sprintf("virt-export-%s", vme.Name), 443)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.PORT_FORWARD_FLAG, "5432"), setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("No pods found for the service virt-export-test-vme"))
		})

		It("VirtualMachineExport download with port-forward succeeds", func() {
			virtctlvmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(Equal("https://127.0.0.1:5432"))
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}
				return &resp, nil
			}
			vme.Status.Links.Internal = vme.Status.Links.External
			utils.HandleServiceGet(kubeClient, fmt.Sprintf("virt-export-%s", vme.Name), 443)
			utils.HandlePodList(kubeClient, fmt.Sprintf("virt-export-pod-%s", vme.Name))
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.PORT_FORWARD_FLAG, "5432"), setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
