package vmexport_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	commandName  = "vmexport"
	vmexportName = "test-vme"
	volumeName   = "test-volume"
	secretName   = "secret-test-vme"
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

		vmexport.ExportProcessingComplete = utils.WaitExportCompleteDefault
		vmexport.SetHTTPClientCreator(func(*http.Transport, bool) *http.Client {
			return server.Client()
		})
	}

	testDone := func() {
		vmexport.SetDefaultHTTPClientCreator()
		server.Close()
	}

	Context("VMExport fails", func() {
		It("VirtualMachineExport already exists when using 'create'", func() {
			testInit(http.StatusOK)
			vmexport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)
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
			vmexport.ExportProcessingComplete = utils.WaitExportCompleteError
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(PVC_FLAG, "test-pvc"), setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("processing failed: Test error"))
		})

		It("VirtualMachineExport download fails when there's no volume available", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus(nil, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to access the volume info from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the volumes have a different name than expected", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
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

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when there are multiple volumes and no volume name has been specified", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
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

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when no format is available", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{{Name: volumeName}}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the only available format is incompatible with download", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.Dir),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).To(HaveOccurred())
			expectedError := fmt.Sprintf("Unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmexportName)
			Expect(err.Error()).Should(ContainSubstring(expectedError))
		})

		It("VirtualMachineExport download fails when the secret token is not attainable", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)
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
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)
			utils.HandleSecretGet(kubeClient, secretName)

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
			Entry("No arguments", "argument validation failed", []string{}),
			Entry("Missing arg", "argument validation failed", []string{CREATE, setflag(PVC_FLAG, vmexportName)}),
			Entry("More arguments than expected", "argument validation failed", []string{CREATE, DELETE, vmexportName}),
			Entry("Using 'create' without export type", ErrRequiredExportType, []string{CREATE, vmexportName}),
			Entry("Using 'create' with invalid flag", fmt.Sprintf(ErrIncompatibleFlag, INSECURE_FLAG, CREATE), []string{CREATE, vmexportName, setflag(PVC_FLAG, "test"), INSECURE_FLAG}),
			Entry("Using 'delete' with export type", ErrIncompatibleExportType, []string{DELETE, vmexportName, setflag(PVC_FLAG, "test")}),
			Entry("Using 'delete' with invalid flag", fmt.Sprintf(ErrIncompatibleFlag, INSECURE_FLAG, DELETE), []string{DELETE, vmexportName, INSECURE_FLAG}),
			Entry("Using 'download' without required flags", fmt.Sprintf(ErrRequiredFlag, OUTPUT_FLAG, DOWNLOAD), []string{DOWNLOAD, vmexportName}),
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
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, CREATE, vmexportName, setflag(PVC_FLAG, "test-pvc"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		// Delete tests
		It("VirtualMachineExport is deleted succesfully", func() {
			handleVMExportDelete(vmExportClient, vmexportName)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DELETE, vmexportName)
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
			vmexport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create and download a VirtualMachineExport", func() {
			vmexport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(PVC_FLAG, "test-pvc"), setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully download a VirtualMachineExport with just 'raw' links", func() {
			vmexport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vmexport, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, volumeName), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when the volume has a different name than expected but there's only one volume", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, volumeName))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when there's only one volume and no --volume has been specified", func() {
			testInit(http.StatusOK)
			vme := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(vmExportClient, vme, vmexportName)

			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(OUTPUT_FLAG, "disk.img"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create VirtualMachineExport with TTL", func() {
			vmexport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
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
			vmExport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
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
			url, err := vmexport.GetUrlFromVirtualMachineExport(vmExport, vmeinfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("compressed"))
		})

		It("Should get raw URL when there's no other option", func() {
			vmExport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vmExport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat("raw", exportv1.KubeVirtRaw),
				},
			}, secretName)
			url, err := vmexport.GetUrlFromVirtualMachineExport(vmExport, vmeinfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("raw"))
		})

		It("Should not get any URL when there's no valid options", func() {
			vmExport := utils.VMExportSpec(vmexportName, metav1.NamespaceDefault, "pvc", "test-pvc", secretName)
			vmExport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.Dir),
				},
			}, secretName)
			url, err := vmexport.GetUrlFromVirtualMachineExport(vmExport, vmeinfo)
			Expect(err).To(HaveOccurred())
			Expect(url).To(Equal(""))
		})

		AfterEach(func() {
			testDone()
		})
	})
})

func handleVMExportDelete(client *kubevirtfake.Clientset, name string) {
	client.Fake.PrependReactor("delete", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		delete, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())
		Expect(delete.GetName()).To(Equal(name))
		return true, nil, nil
	})
}
