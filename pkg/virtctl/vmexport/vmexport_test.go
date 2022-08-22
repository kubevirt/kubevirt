package vmexport_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	. "kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	commandName      = "vmexport"
	defaultNamespace = "default"
	vmexportName     = "test-vme"
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
		vmexport := &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: secretName,
				Source: v1.TypedLocalObjectReference{
					APIGroup: &v1.SchemeGroupVersion.Group,
					Kind:     kind,
					Name:     resourceName,
				},
			},
		}

		return vmexport
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

		vmexport.SetHTTPClientCreator(func(*http.Transport) *http.Client {
			return server.Client()
		})
	}

	testDone := func() {
		vmexport.SetDefaultHTTPClientCreator()
		server.Close()
	}

	Context("VMExport fails", func() {
		BeforeEach(func() {
			testInit(http.StatusOK)
		})

		It("VirtualMachineExport already exists when using 'create'", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			expectVMExportGet(vmExportClient, vmexport)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, CREATE, vmexportName, setflag(PVC_FLAG, "test-pvc"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("VirtualMachineExport 'default/test-vme' already exists"))
		})

		It("VirtualMachineExport doesn't exist when using 'delete'", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DELETE, vmexportName)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("VirtualMachineExport 'default/test-vme' does not exist"))
		})

		It("VirtualMachineExport doesn't exist when using 'download' without the --create flag", func() {
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, "testVolume"), setflag(OUTPUT_FLAG, "output.img"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("Unable to get 'default/test-vme' VirtualMachineExport"))
		})

		It("VirtualMachineExport already exists when using 'download' with the --create flag", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			expectVMExportGet(vmExportClient, vmexport)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(PVC_FLAG, "test-pvc"), CREATE_FLAG, setflag(OUTPUT_FLAG, "disk.img"), setflag(VOLUME_FLAG, "test-volume"))
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("VirtualMachineExport 'default/test-vme' already exists"))
		})

		It("Bad flag combination", func() {
			args := []string{CREATE, vmexportName, setflag(PVC_FLAG, "test"), setflag(VM_FLAG, "test2"), setflag(SNAPSHOT_FLAG, "test3")}
			args = append([]string{commandName}, args...)
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("if any flags in the group [vm snapshot pvc] are set none of the others can be; [pvc snapshot vm] were all set"))
		})

		DescribeTable("Invalid arguments/flags", func(errString string, args []string) {
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
			Entry("Using 'download' with --create and without export type", ErrRequiredExportType, []string{DOWNLOAD, vmexportName, CREATE_FLAG}),
			Entry("Using 'download' without --create flag and with export type", ErrIncompatibleExportType, []string{DOWNLOAD, vmexportName, setflag(PVC_FLAG, "test")}),
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
			expectVMExportCreate(vmExportClient, nil)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, CREATE, vmexportName, setflag(PVC_FLAG, "test-pvc"))
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		// Delete tests
		It("VirtualMachineExport is deleted succesfully", func() {
			expectVMExportDelete(vmExportClient, vmexportName)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DELETE, vmexportName)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		// Download tests
		It("Succesfully download from an already existing VirtualMachineExport", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			mockDownload(vmexport, server.URL, false, vmExportClient, kubeClient)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, setflag(VOLUME_FLAG, "test-pvc"), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create and download a VirtualMachineExport", func() {
			vmexport := vmexportSpec(vmexportName, defaultNamespace, "pvc", "test-pvc")
			mockDownload(vmexport, server.URL, true, vmExportClient, kubeClient)
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandName, DOWNLOAD, vmexportName, CREATE_FLAG, setflag(PVC_FLAG, "test-pvc"), setflag(VOLUME_FLAG, "test-pvc"), setflag(OUTPUT_FLAG, "test-pvc"), INSECURE_FLAG)
			err := cmd()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			testDone()
		})
	})
})

func expectVMExportDelete(client *kubevirtfake.Clientset, name string) {
	client.Fake.PrependReactor("delete", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		delete, ok := action.(testing.DeleteAction)
		Expect(ok).To(BeTrue())
		Expect(delete.GetName()).To(Equal(name))
		return true, nil, nil
	})
}

func expectVMExportGet(client *kubevirtfake.Clientset, vme *exportv1.VirtualMachineExport) {
	client.Fake.PrependReactor("get", "virtualmachineexports", func(action testing.Action) (bool, runtime.Object, error) {
		get, ok := action.(testing.GetAction)
		Expect(ok).To(BeTrue())
		Expect(get.GetNamespace()).To(Equal(defaultNamespace))
		Expect(get.GetName()).To(Equal(vmexportName))
		if vme == nil {
			return true, nil, errors.NewNotFound(v1.Resource("virtualmachineexport"), vmexportName)
		}
		return true, vme, nil
	})
}

func expectVMExportCreate(client *kubevirtfake.Clientset, vme *exportv1.VirtualMachineExport) {
	client.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		create, ok := action.(testing.CreateAction)
		Expect(ok).To(BeTrue())

		if vme == nil {
			vme, ok = create.GetObject().(*exportv1.VirtualMachineExport)
		} else {
			_, ok = create.GetObject().(*exportv1.VirtualMachineExport)
		}

		Expect(ok).To(BeTrue())
		expectVMExportGet(client, vme)
		return true, vme, nil
	})
}

func expectSecretGet(k8sClient *fakek8sclient.Clientset) {
	secret := &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: defaultNamespace,
		},
		Type: k8sv1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte("test"),
		},
	}

	k8sClient.Fake.PrependReactor("get", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
		get, ok := action.(testing.GetAction)
		Expect(ok).To(BeTrue())
		Expect(get.GetNamespace()).To(Equal(defaultNamespace))
		Expect(get.GetName()).To(Equal(secretName))
		if secret == nil {
			return true, nil, errors.NewNotFound(v1.Resource("Secret"), secretName)
		}
		return true, secret, nil
	})
}

func mockDownload(vme *exportv1.VirtualMachineExport, url string, create bool, client *kubevirtfake.Clientset, k8sClient *fakek8sclient.Clientset) {
	// Mock the expected vme status
	vme.Status = &exportv1.VirtualMachineExportStatus{
		Phase: exportv1.Ready,
		Links: &exportv1.VirtualMachineExportLinks{
			External: &exportv1.VirtualMachineExportLink{
				Volumes: []exportv1.VirtualMachineExportVolume{
					{
						Name: vme.Spec.Source.Name,
						Formats: []exportv1.VirtualMachineExportVolumeFormat{
							{
								Format: exportv1.KubeVirtRaw,
								Url:    url,
							},
							{
								Format: exportv1.KubeVirtGz,
								Url:    url,
							},
						},
					},
				},
				Cert: "test",
			},
		},
	}

	expectSecretGet(k8sClient)

	if create == true {
		expectVMExportCreate(client, vme)
	} else {
		expectVMExportGet(client, vme)
	}
}
