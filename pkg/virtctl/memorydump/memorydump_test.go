package memorydump_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1alpha1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/virtctl/memorydump"
	"kubevirt.io/kubevirt/pkg/virtctl/utils"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	createClaimFlag   = "--create-claim"
	claimNameFlag     = "--claim-name=testpvc"
	claimName         = "testpvc"
	configName        = "config"
	vmName            = "testvm"
	defaultFSOverhead = "0.055"
)

var (
	cdiClient       *cdifake.Clientset
	coreClient      *fake.Clientset
	pvcCreateCalled = &utils.AtomicBool{Lock: &sync.Mutex{}}
)

var _ = Describe("MemoryDump", func() {
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)

		pvcCreateCalled.False()
		coreClient = fake.NewSimpleClientset()
		cdiConfig := cdiConfigInit()
		cdiClient = cdifake.NewSimpleClientset(cdiConfig)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1()).AnyTimes()
	})

	handleGetCDIConfig := func() {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
	}

	updateCDIConfig := func() {
		config, err := cdiClient.CdiV1beta1().CDIConfigs().Get(context.Background(), configName, k8smetav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		config.Status.FilesystemOverhead.StorageClass = make(map[string]v1beta1.Percent)
		config.Status.FilesystemOverhead.StorageClass["fakeSC"] = v1beta1.Percent("0.1")
		_, err = cdiClient.CdiV1beta1().CDIConfigs().Update(context.Background(), config, k8smetav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
	}

	expectVMEndpointMemoryDump := func(vmName, claimName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().MemoryDump(context.Background(), vmName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			Expect(arg0.(string)).To(Equal(vmName))
			Expect(arg1.(*v1.VirtualMachineMemoryDumpRequest).ClaimName).To(Equal(claimName))
			return nil
		})
	}

	expectVMEndpointRemoveMemoryDump := func(vmName string) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().RemoveMemoryDump(context.Background(), vmName).DoAndReturn(func(ctx context.Context, arg0 interface{}) interface{} {
			Expect(arg0.(string)).To(Equal(vmName))
			return nil
		})
	}

	expectGetVM := func(withAssociatedMemoryDump bool) {
		vm := &v1.VirtualMachine{
			Spec:   v1.VirtualMachineSpec{},
			Status: v1.VirtualMachineStatus{},
		}
		if withAssociatedMemoryDump {
			vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{}
		}
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(k8smetav1.NamespaceDefault).Return(vmInterface).Times(1)
		vmInterface.EXPECT().Get(context.Background(), vmName, gomock.Any()).Return(vm, nil).Times(1)
	}

	expectGetVMNoAssociatedMemoryDump := func() {
		expectGetVM(false)
	}

	expectGetVMWithAssociatedMemoryDump := func() {
		expectGetVM(true)
	}

	expectGetVMI := func() {
		quantity, _ := resource.ParseQuantity("256Mi")
		vmi := &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Resources: v1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceMemory: quantity,
						},
					},
				},
			},
		}
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)

		vmiInterface.EXPECT().Get(context.Background(), vmName, gomock.Any()).Return(vmi, nil).Times(1)
	}

	pvcSpec := func() *k8sv1.PersistentVolumeClaim {
		return &k8sv1.PersistentVolumeClaim{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: claimName,
			},
		}
	}

	expectPVCCreate := func(claimName, storageclass, accessMode string) {
		coreClient.Fake.PrependReactor("create", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())

			pvc, ok := create.GetObject().(*k8sv1.PersistentVolumeClaim)
			Expect(ok).To(BeTrue())
			Expect(pvc.Name).To(Equal(claimName))

			if storageclass != "" {
				Expect(*pvc.Spec.StorageClassName).To(Equal(storageclass))
				// 392Mi = (256Mi(vmi memory size) + 100Mi (memory dump overhead)) * 10%fsoverhead for fake storage class rounded to MiB
				quantity, _ := resource.ParseQuantity("376Mi")
				Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(quantity))
			} else {
				// 376Mi = (256Mi(vmi memory size) + 100Mi (memory dump overhead)) * 5.5%fsoverhead rounded to MiB
				quantity, _ := resource.ParseQuantity("376Mi")
				Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(quantity))
			}
			if accessMode != "" {
				Expect(pvc.Spec.AccessModes[0]).To(Equal(k8sv1.PersistentVolumeAccessMode(accessMode)))
			}

			Expect(pvcCreateCalled.IsTrue()).To(BeFalse())
			pvcCreateCalled.True()

			return false, nil, nil
		})
	}

	handlePVCGet := func(pvc *k8sv1.PersistentVolumeClaim) {
		coreClient.Fake.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
			get, ok := action.(testing.GetAction)
			Expect(ok).To(BeTrue())
			Expect(get.GetNamespace()).To(Equal(k8smetav1.NamespaceDefault))
			Expect(get.GetName()).To(Equal(claimName))
			if pvc == nil {
				return true, nil, errors.NewNotFound(v1.Resource("persistentvolumeclaim"), claimName)
			}
			return true, pvc, nil
		})
	}

	DescribeTable("should fail with missing required or invalid parameters", func(errorString string, args ...string) {
		commandAndArgs := []string{"memory-dump"}
		commandAndArgs = append(commandAndArgs, args...)
		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		res := cmdAdd()
		Expect(res).To(HaveOccurred())
		Expect(res.Error()).To(ContainSubstring(errorString))
	},
		Entry("memorydump no args", "argument validation failed"),
		Entry("memorydump missing action arg", "argument validation failed", "testvm"),
		Entry("memorydump missing vm name arg", "argument validation failed", "get"),
		Entry("memorydump wrong action arg", "invalid action type create", "create", "testvm"),
		Entry("memorydump name, invalid extra parameter", "unknown flag", "testvm", "--claim-name=blah", "--invalid=test"),
		Entry("memorydump download missing outputFile", "missing outputFile", "download", "testvm", "--claim-name=pvc"),
	)

	It("should call memory dump subresource", func() {
		expectVMEndpointMemoryDump("testvm", claimName)
		commandAndArgs := []string{"memory-dump", "get", "testvm", claimNameFlag}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		Expect(cmd.Execute()).To(Succeed())
	})

	It("should call memory dump subresource without claim-name no create", func() {
		expectVMEndpointMemoryDump("testvm", "")
		commandAndArgs := []string{"memory-dump", "get", "testvm"}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		Expect(cmd.Execute()).To(Succeed())
	})

	It("should fail call memory dump subresource without claim-name with create-claim", func() {
		commandAndArgs := []string{"memory-dump", "get", "testvm", createClaimFlag}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		res := cmd.Execute()
		Expect(res).To(HaveOccurred())
		Expect(res.Error()).To(ContainSubstring("missing claim name"))
	})

	It("should fail call memory dump subresource with create-claim with already associated memory dump pvc", func() {
		expectGetVMWithAssociatedMemoryDump()
		commandAndArgs := []string{"memory-dump", "get", "testvm", claimNameFlag, createClaimFlag}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		res := cmd.Execute()
		Expect(res).To(HaveOccurred())
		Expect(res.Error()).To(ContainSubstring("please remove current memory dump"))
	})

	It("should fail call memory dump subresource with create-claim and existing pvc", func() {
		expectGetVMNoAssociatedMemoryDump()
		handlePVCGet(pvcSpec())
		commandAndArgs := []string{"memory-dump", "get", "testvm", claimNameFlag, createClaimFlag}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		res := cmd.Execute()
		Expect(res).To(HaveOccurred())
		Expect(res.Error()).To(ContainSubstring("already exists"))
	})

	It("should fail call memory dump subresource with create-claim no vmi", func() {
		expectGetVMNoAssociatedMemoryDump()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(k8smetav1.NamespaceDefault).Return(vmiInterface).Times(1)
		vmiInterface.EXPECT().Get(context.Background(), vmName, &k8smetav1.GetOptions{}).Return(nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), vmName))
		commandAndArgs := []string{"memory-dump", "get", "testvm", claimNameFlag, createClaimFlag}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		res := cmd.Execute()
		Expect(res).To(HaveOccurred())
		Expect(res.Error()).To(ContainSubstring("not found"))
	})

	DescribeTable("should fail call memory dump subresource with invalid access mode", func(accessMode, expectedErr string) {
		handleGetCDIConfig()
		expectGetVMNoAssociatedMemoryDump()
		expectGetVMI()
		commandAndArgs := []string{"memory-dump", "get", "testvm", claimNameFlag, createClaimFlag, fmt.Sprintf("--access-mode=%s", accessMode)}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		res := cmd.Execute()
		Expect(res).To(HaveOccurred())
		Expect(res.Error()).To(ContainSubstring(expectedErr))
	},
		Entry("readonly accessMode", "ReadOnlyMany", "cannot dump memory to a readonly pvc"),
		Entry("invalid accessMode", "RWX", "invalid access mode"),
	)

	DescribeTable("should create pvc for memory dump and call subresource", func(storageclass, accessMode string) {
		handleGetCDIConfig()
		expectGetVMNoAssociatedMemoryDump()
		expectGetVMI()
		expectPVCCreate(claimName, storageclass, accessMode)
		expectVMEndpointMemoryDump("testvm", claimName)
		commandAndArgs := []string{"memory-dump", "get", "testvm", claimNameFlag, createClaimFlag}
		if storageclass != "" {
			updateCDIConfig()
			commandAndArgs = append(commandAndArgs, fmt.Sprintf("--storage-class=%s", storageclass))
		}
		if accessMode != "" {
			commandAndArgs = append(commandAndArgs, fmt.Sprintf("--access-mode=%s", accessMode))
		}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		Expect(cmd.Execute()).To(Succeed())
		Expect(pvcCreateCalled.IsTrue()).To(BeTrue())
	},
		Entry("no other flags", "", ""),
		Entry("with storageclass flag", "local", ""),
		Entry("with access mode flag", "", "ReadWriteOnce"),
	)

	It("should call remove memory dump subresource", func() {
		expectVMEndpointRemoveMemoryDump("testvm")
		commandAndArgs := []string{"memory-dump", "remove", "testvm"}
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
		Expect(cmd.Execute()).To(Succeed())
	})

	Context("Download of memory dump", func() {
		var (
			vmExportClient *kubevirtfake.Clientset
			server         *httptest.Server
		)
		const (
			secretName     = "secret-test-vme"
			vmexportName   = "export-testvm-testpvc"
			outputFileFlag = "--output=out.dump.gz"
		)

		waitForMemoryDumpDefault := func(kubecli.KubevirtClient, string, string, time.Duration, time.Duration) (string, error) {
			return claimName, nil
		}

		waitForMemoryDumpErr := func(kubecli.KubevirtClient, string, string, time.Duration, time.Duration) (string, error) {
			return claimName, fmt.Errorf("memory dump failed: test err")
		}

		addDefaultReactors := func() {
			vmExportClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				vmExport, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				return true, vmExport, nil
			})

			coreClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				secret, ok := create.GetObject().(*k8sv1.Secret)
				Expect(ok).To(BeTrue())
				return true, secret, nil
			})
		}

		BeforeEach(func() {
			vmExportClient = kubevirtfake.NewSimpleClientset()

			kubecli.MockKubevirtClientInstance.EXPECT().StorageV1().Return(coreClient.StorageV1()).AnyTimes()
			kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineExport(k8smetav1.NamespaceDefault).Return(vmExportClient.ExportV1alpha1().VirtualMachineExports(k8smetav1.NamespaceDefault)).AnyTimes()
			addDefaultReactors()

			server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			vmexport.ExportProcessingComplete = utils.WaitExportCompleteDefault
			vmexport.SetHTTPClientCreator(func(*http.Transport, bool) *http.Client {
				return server.Client()
			})
			vmexport.SetPortForwarder(func(client kubecli.KubevirtClient, pod k8sv1.Pod, namespace string, ports []string, stopChan, readyChan chan struct{}, portChan chan uint16) error {
				readyChan <- struct{}{}
				portChan <- uint16(5432)
				return nil
			})
		})

		AfterEach(func() {
			vmexport.SetDefaultPortForwarder()
			vmexport.SetDefaultHTTPClientCreator()
		})

		It("should get memory dump and call download memory dump", func() {
			expectVMEndpointMemoryDump("testvm", "")
			memorydump.WaitMemoryDumpComplete = waitForMemoryDumpDefault

			vmexport := utils.VMExportSpecPVC(vmexportName, k8smetav1.NamespaceDefault, claimName, secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    claimName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(coreClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			commandAndArgs := []string{"memory-dump", "get", "testvm", outputFileFlag}
			cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
			Expect(cmd.Execute()).To(Succeed())
		})

		It("should call download memory dump", func() {
			memorydump.WaitMemoryDumpComplete = waitForMemoryDumpDefault
			vmexport := utils.VMExportSpecPVC(vmexportName, k8smetav1.NamespaceDefault, claimName, secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    claimName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(coreClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			commandAndArgs := []string{"memory-dump", "download", "testvm", outputFileFlag}
			cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
			Expect(cmd.Execute()).To(Succeed())
		})

		It("should call download memory dump and decompress succesfully", func() {
			vmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
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
			memorydump.WaitMemoryDumpComplete = waitForMemoryDumpDefault
			vmexport := utils.VMExportSpecPVC(vmexportName, k8smetav1.NamespaceDefault, claimName, secretName)
			vmexport.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    claimName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(coreClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vmexport)

			commandAndArgs := []string{"memory-dump", "download", "testvm", outputFileFlag, "--format", "raw"}
			cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
			Expect(cmd.Execute()).To(Succeed())
		})

		DescribeTable("should call download memory dump with port-forward", func(commandAndArgs []string) {
			vmexport.HandleHTTPRequest = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(Equal("https://127.0.0.1:5432"))
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}
				return &resp, nil
			}
			memorydump.WaitMemoryDumpComplete = waitForMemoryDumpDefault
			vme := utils.VMExportSpecPVC(vmexportName, k8smetav1.NamespaceDefault, claimName, secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    claimName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vme.Status.Links.Internal = vme.Status.Links.External
			utils.HandleSecretGet(coreClient, secretName)
			utils.HandleVMExportCreate(vmExportClient, vme)
			utils.HandleServiceGet(coreClient, fmt.Sprintf("virt-export-%s", vme.Name), 443)
			utils.HandlePodList(coreClient, fmt.Sprintf("virt-export-pod-%s", vme.Name))
			cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)
			Expect(cmd.Execute()).To(Succeed())
		},
			Entry("with default port-forward", []string{"memory-dump", "download", "testvm", outputFileFlag, "--port-forward"}),
			Entry("with port-forward specifying local port", []string{"memory-dump", "download", "testvm", outputFileFlag, "--port-forward", "--local-port", "5432"}),
			Entry("with port-forward specifying default number on local port", []string{"memory-dump", "download", "testvm", outputFileFlag, "--port-forward", "--local-port", "0"}),
		)

		It("should fail download memory dump if not completed succesfully", func() {
			memorydump.WaitMemoryDumpComplete = waitForMemoryDumpErr

			commandAndArgs := []string{"memory-dump", "download", "testvm", outputFileFlag}
			cmd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
			err := cmd()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(Equal("memory dump failed: test err"))
		})
	})
})

func cdiConfigInit() (cdiConfig *v1beta1.CDIConfig) {
	cdiConfig = &v1beta1.CDIConfig{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: configName,
		},
		Spec: v1beta1.CDIConfigSpec{
			UploadProxyURLOverride: nil,
		},
		Status: v1beta1.CDIConfigStatus{
			FilesystemOverhead: &v1beta1.FilesystemOverhead{
				Global: v1beta1.Percent(defaultFSOverhead),
			},
		},
	}
	return
}
