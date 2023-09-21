package vmexport_test

import (
	"bytes"
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
	virtctlvmexport "kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	commandName  = "vmexport"
	vmexportName = "test-vme"
	volumeName   = "test-volume"
	secretName   = "secret-test-vme"

	manifestUrl = "https://test.something.somewhere/test/all"
	secretUrl   = "https://test.something.somewhere/test/secret"
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
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineExport(metav1.NamespaceDefault).Return(vmExportClient.ExportV1alpha1().VirtualMachineExports(metav1.NamespaceDefault)).AnyTimes()

		addDefaultReactors()

		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		}))

		virtctlvmexport.ExportProcessingComplete = utils.WaitExportCompleteDefault
		virtctlvmexport.SetHTTPClientCreator(func(*http.Transport, bool) *http.Client {
			return server.Client()
		})
		virtctlvmexport.SetPortForwarder(func(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}, portChan chan uint16) error {
			readyChan <- struct{}{}
			portChan <- uint16(5432)
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
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.CREATE, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test-pvc"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("VirtualMachineExport 'default/test-vme' already exists"))
		})

		It("VirtualMachineExport doesn't exist when using 'download' without source type", func() {
			testInit(http.StatusOK)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.OUTPUT_FLAG, "output.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("unable to get 'default/test-vme' VirtualMachineExport"))
		})

		It("VirtualMachineExport processing fails when using 'download'", func() {
			testInit(http.StatusOK)
			virtctlvmexport.ExportProcessingComplete = utils.WaitExportCompleteError
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test-pvc"), setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("processing failed: Test error"))
		})

		It("VirtualMachineExport download fails when there's no volume available", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus(nil, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("unable to access the volume info from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the volumes have a different name than expected", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
				{
					Name:    "no-test-volume-2",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when there are multiple volumes and no volume name has been specified", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
				{
					Name:    "test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when no format is available", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{{Name: volumeName}}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the only available format is incompatible with download", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.Dir),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the secret token is not attainable", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)
			// Adding a new reactor so the client returns a nil secret
			kubeClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) { return false, nil, nil })

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("secrets \"%s\" not found", secretName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails if the server returns a bad status", func() {
			testInit(http.StatusInternalServerError)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)
			utils.HandleSecretGet(kubeClient, secretName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("bad status: 500 Internal Server Error"))
		})

		It("Bad flag combination", func() {
			testInit(http.StatusOK)
			args := []string{virtctlvmexport.CREATE, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test"), setflag(virtctlvmexport.VM_FLAG, "test2"), setflag(virtctlvmexport.SNAPSHOT_FLAG, "test3")}
			args = append([]string{commandName}, args...)
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("if any flags in the group [vm snapshot pvc] are set none of the others can be; [pvc snapshot vm] were all set"))
		})

		DescribeTable("Invalid arguments/flags", func(errString string, args ...string) {
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
			Entry("Using 'port-forward' with invalid port", fmt.Sprintf(virtctlvmexport.ErrInvalidValue, virtctlvmexport.LOCAL_PORT_FLAG, "valid port numbers"), virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.PORT_FORWARD_FLAG, setflag(virtctlvmexport.LOCAL_PORT_FLAG, "test")),
			Entry("Using 'format' with invalid download format", fmt.Sprintf(virtctlvmexport.ErrInvalidValue, virtctlvmexport.FORMAT_FLAG, "gzip/raw"), virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.FORMAT_FLAG, "test")),
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
			utils.HandleVMExportCreate(vmExportClient, nil)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.CREATE, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test-pvc"))
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
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DELETE, vmexportName)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		// Download tests
		It("Succesfully download from an already existing VirtualMachineExport", func() {
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.OUTPUT_FLAG, "test-pvc"), virtctlvmexport.INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create and download a VirtualMachineExport", func() {
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test-pvc"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.OUTPUT_FLAG, "test-pvc"), virtctlvmexport.INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create and download a VirtualMachineExport with raw format", func() {
			virtctlvmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader([]byte{
						0x1f, 0x8b, 0x08, 0x08, 0xc8, 0x58, 0x13, 0x4a,
						0x00, 0x03, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x2e,
						0x74, 0x78, 0x74, 0x00, 0xcb, 0x48, 0xcd, 0xc9,
						0xc9, 0x57, 0x28, 0xcf, 0x2f, 0xca, 0x49, 0xe1,
						0x02, 0x00, 0x2d, 0x3b, 0x08, 0xaf, 0x0c, 0x00,
						0x00, 0x00,
						0x1f, 0x8b, 0x08, 0x08, 0xc8, 0x58, 0x13, 0x4a,
						0x00, 0x03, 0x68, 0x65, 0x6c, 0x6c, 0x6f, 0x2e,
						0x74, 0x78, 0x74, 0x00, 0xcb, 0x48, 0xcd, 0xc9,
						0xc9, 0x57, 0x28, 0xcf, 0x2f, 0xca, 0x49, 0xe1,
						0x02, 0x00, 0x2d, 0x3b, 0x08, 0xaf, 0x0c, 0x00,
						0x00, 0x00,
					})),
				}
				return &resp, nil
			}
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.FORMAT_FLAG, virtctlvmexport.RAW_FORMAT), setflag(virtctlvmexport.PVC_FLAG, "test-pvc"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.OUTPUT_FLAG, "test-pvc"), virtctlvmexport.INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully download a VirtualMachineExport without decompressing is url is already raw", func() {
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.FORMAT_FLAG, virtctlvmexport.RAW_FORMAT), setflag(virtctlvmexport.PVC_FLAG, "test-pvc"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.OUTPUT_FLAG, "test-pvc"), virtctlvmexport.INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully download a VirtualMachineExport with just 'raw' links", func() {
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.VOLUME_FLAG, volumeName), setflag(virtctlvmexport.OUTPUT_FLAG, "test-pvc"), virtctlvmexport.INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when the volume has a different name than expected but there's only one volume", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"), setflag(virtctlvmexport.VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when there's only one volume and no --volume has been specified", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create VirtualMachineExport with TTL", func() {
			vmexport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				Expect(*vme.Spec.TTLDuration).To(Equal(metav1.Duration{Duration: time.Minute}))

				return true, vme, nil
			})

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.CREATE, vmexportName, setflag(virtctlvmexport.PVC_FLAG, "test-pvc"), setflag(virtctlvmexport.TTL_FLAG, "1m"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			testDone()
		})
	})

	Context("getUrlFromVirtualMachineExport", func() {
		// Mocking the minimum viable VMExportInfo struct
		var vmeinfo *virtctlvmexport.VMExportInfo
		BeforeEach(func() {
			vmeinfo = &virtctlvmexport.VMExportInfo{
				Name:       vmexportName,
				VolumeName: volumeName,
			}
		})

		It("Should get compressed URL even when there's multiple URLs", func() {
			vmExport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmExport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
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
			}, secretName)
			url, err := virtctlvmexport.GetUrlFromVirtualMachineExport(vmExport, vmeinfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("compressed"))
		})

		It("Should get raw URL when there's no other option", func() {
			vmExport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmExport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat("raw", exportv1.KubeVirtRaw),
				},
			}, secretName)
			url, err := virtctlvmexport.GetUrlFromVirtualMachineExport(vmExport, vmeinfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("raw"))
		})

		It("Should not get any URL when there's no valid options", func() {
			vmExport := utils.VMExportSpecPVC(vmexportName, metav1.NamespaceDefault, "test-pvc", secretName)
			vmExport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.Dir),
				},
			}, secretName)
			url, err := virtctlvmexport.GetUrlFromVirtualMachineExport(vmExport, vmeinfo)
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
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.PORT_FORWARD_FLAG, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("Service virt-export-test-vme does not have a service port 443"))
		})

		It("VirtualMachineExport download with port-forward fails when the service doesn't have a valid pod ", func() {
			utils.HandleServiceGet(kubeClient, fmt.Sprintf("virt-export-%s", vme.Name), 443)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, virtctlvmexport.PORT_FORWARD_FLAG, setflag(virtctlvmexport.LOCAL_PORT_FLAG, "5432"), setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
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
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, virtctlvmexport.DOWNLOAD, vmexportName, setflag(virtctlvmexport.VOLUME_FLAG, volumeName), virtctlvmexport.PORT_FORWARD_FLAG, setflag(virtctlvmexport.OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
