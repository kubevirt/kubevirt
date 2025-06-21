package memorydump_test

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	kvtesting "kubevirt.io/client-go/testing"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virtctl/memorydump"
	"kubevirt.io/kubevirt/pkg/virtctl/testing"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
)

const vmName = "test-vm"

var _ = Describe("MemoryDump", func() {
	const (
		pvcName = "test-pvc"
		scName  = "test-sc"

		vmiMemory         = "256Mi"
		defaultFSOverhead = "0.055"
		scOverhead        = "0.1"

		// 376Mi = (256Mi(vmi memory size) + 100Mi (memory dump overhead)) * 5.5%fsoverhead rounded to MiB
		defaultFSOverheadSize = "376Mi"

		// 392Mi = (256Mi(vmi memory size) + 100Mi (memory dump overhead)) * 10%fsoverhead for fake storage class rounded to MiB
		scOverheadSize = "392Mi"
	)

	var (
		cdiClient  *cdifake.Clientset
		kubeClient *fake.Clientset
		virtClient *kubevirtfake.Clientset

		vm  *v1.VirtualMachine
		vmi *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		cdiClient = cdifake.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().StorageV1().Return(kubeClient.StorageV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineExport(metav1.NamespaceDefault).Return(virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault)).AnyTimes()

		cdiConfig := &cdiv1.CDIConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "config",
			},
			Spec: cdiv1.CDIConfigSpec{
				UploadProxyURLOverride: nil,
			},
			Status: cdiv1.CDIConfigStatus{
				FilesystemOverhead: &cdiv1.FilesystemOverhead{
					Global: cdiv1.Percent(defaultFSOverhead),
					StorageClass: map[string]cdiv1.Percent{
						scName: scOverhead,
					},
				},
			},
		}
		_, err := cdiClient.CdiV1beta1().CDIConfigs().Create(context.Background(), cdiConfig, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmi = libvmi.New(
			libvmi.WithName(vmName),
			libvmi.WithMemoryRequest(vmiMemory),
		)
		vmi, err = virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		vm = libvmi.NewVirtualMachine(vmi)
		vm, err = virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	expectVMEndpointMemoryDump := func(claimName string) {
		virtClient.PrependReactor("put", "virtualmachines/memorydump", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			switch action := action.(type) {
			case kvtesting.PutAction[*v1.VirtualMachineMemoryDumpRequest]:
				Expect(action.GetName()).To(Equal(vm.Name))
				request := action.GetOptions()
				Expect(request).ToNot(BeNil())
				Expect(request.ClaimName).To(Equal(claimName))
				return true, nil, nil
			default:
				Fail("unexpected action type on memorydump")
				return false, nil, nil
			}
		})
	}

	DescribeTable("should fail with missing required or invalid parameters", func(errorString string, args ...string) {
		Expect(runCmd(args...)).To(MatchError(ContainSubstring(errorString)))
	},
		Entry("memorydump no args", "accepts 2 arg(s), received 0"),
		Entry("memorydump missing action arg", "accepts 2 arg(s), received 1", vmName),
		Entry("memorydump missing vm name arg", "accepts 2 arg(s), received 1", "get"),
		Entry("memorydump wrong action arg", "invalid action type create", "create", vmName),
		Entry("memorydump name, invalid extra parameter", "unknown flag", "testvm", setFlag(memorydump.ClaimNameFlag, pvcName), "--invalid=test"),
		Entry("memorydump download missing outputFile", "missing outputFile", "download", "testvm", setFlag(memorydump.ClaimNameFlag, pvcName)),
	)

	It("should call memory dump subresource", func() {
		expectVMEndpointMemoryDump(pvcName)
		err := runGetCmd(
			setFlag(memorydump.ClaimNameFlag, pvcName),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "memorydump")).To(HaveLen(1))
	})

	It("should call memory dump subresource without claim-name no create", func() {
		expectVMEndpointMemoryDump("")
		Expect(runGetCmd()).To(Succeed())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "memorydump")).To(HaveLen(1))
	})

	It("should fail call memory dump subresource without claim-name with create-claim", func() {
		err := runGetCmd(
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).To(MatchError(ContainSubstring("missing claim name")))
	})

	It("should fail call memory dump subresource with create-claim with already associated memory dump pvc", func() {
		vm.Status.MemoryDumpRequest = &v1.VirtualMachineMemoryDumpRequest{}
		_, err := virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault).Update(context.Background(), vm, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = runGetCmd(
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).To(MatchError(ContainSubstring("please remove current memory dump")))
	})

	It("should fail call memory dump subresource with create-claim and existing pvc", func() {
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvcName,
			},
		}
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = runGetCmd(
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).To(MatchError(ContainSubstring("already exists")))
	})

	It("should fail call memory dump subresource with create-claim no vmi", func() {
		err := virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		err = runGetCmd(
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).To(MatchError(ContainSubstring("not found")))
	})

	DescribeTable("should fail call memory dump subresource with invalid access mode", func(accessMode, expectedErr string) {
		err := runGetCmd(
			setFlag(memorydump.AccessModeFlag, accessMode),
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).To(MatchError(ContainSubstring(expectedErr)))
	},
		Entry("readonly accessMode", "ReadOnlyMany", "cannot dump memory to a readonly pvc"),
		Entry("invalid accessMode", "RWX", "invalid access mode"),
	)

	It("should create pvc for memory dump and call subresource with no other flags", func() {
		expectVMEndpointMemoryDump(pvcName)
		err := runGetCmd(
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "memorydump")).To(HaveLen(1))

		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(defaultFSOverheadSize)))
	})

	It("should create pvc for memory dump and call subresource with storageclass flag", func() {
		expectVMEndpointMemoryDump(pvcName)
		err := runGetCmd(
			setFlag(memorydump.StorageClassFlag, scName),
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "memorydump")).To(HaveLen(1))

		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(scOverheadSize)))
		Expect(*pvc.Spec.StorageClassName).To(Equal(scName))
	})

	It("should create pvc for memory dump and call subresource with access mode flag", func() {
		const accessMode = k8sv1.ReadWriteOnce
		expectVMEndpointMemoryDump(pvcName)
		err := runGetCmd(
			setFlag(memorydump.AccessModeFlag, string(accessMode)),
			setFlag(memorydump.ClaimNameFlag, pvcName),
			setFlag(memorydump.CreateClaimFlag, "true"),
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "memorydump")).To(HaveLen(1))

		pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(pvc.Spec.Resources.Requests[k8sv1.ResourceStorage]).To(Equal(resource.MustParse(defaultFSOverheadSize)))
		Expect(pvc.Spec.AccessModes[0]).To(Equal(accessMode))
	})

	It("should call remove memory dump subresource", func() {
		virtClient.PrependReactor("put", "virtualmachines/removememorydump", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			switch action := action.(type) {
			case kvtesting.PutAction[struct{}]:
				Expect(action.GetName()).To(Equal(vm.Name))
				return true, nil, nil
			default:
				Fail("unexpected action type on removememorydump")
				return false, nil, nil
			}
		})

		Expect(runRemoveCmd()).To(Succeed())
		Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "removememorydump")).To(HaveLen(1))
	})

	Context("Download of memory dump", func() {
		const (
			localPort    = uint16(5432)
			localPortStr = "5432"
		)

		var (
			server     *httptest.Server
			outputPath string

			vme     *exportv1.VirtualMachineExport
			secret  *k8sv1.Secret
			service *k8sv1.Service
			pod     *k8sv1.Pod
		)

		updateVMEStatusOnCreate := func() {
			virtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action k8stesting.Action) (bool, runtime.Object, error) {
				create, ok := action.(k8stesting.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				vme.Status = &exportv1.VirtualMachineExportStatus{
					Phase: exportv1.Ready,
					Links: &exportv1.VirtualMachineExportLinks{
						External: &exportv1.VirtualMachineExportLink{
							Volumes: []exportv1.VirtualMachineExportVolume{{
								Name: pvcName,
								Formats: []exportv1.VirtualMachineExportVolumeFormat{{
									Format: exportv1.KubeVirtGz,
									Url:    server.URL,
								}},
							}},
						},
					},
					TokenSecretRef: &secret.Name,
				}
				return false, vme, nil
			})
		}

		BeforeEach(func() {
			server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			outputPath = filepath.Join(GinkgoT().TempDir(), "out.dump.gz")

			vmexport.WaitForVirtualMachineExportFn = func(_ kubecli.KubevirtClient, _ *vmexport.VMExportInfo, _, _ time.Duration) error {
				return nil
			}
			vmexport.GetHTTPClientFn = func(*http.Transport, bool) *http.Client {
				DeferCleanup(server.Close)
				return server.Client()
			}
			vmexport.RunPortForwardFn = func(_ kubecli.KubevirtClient, _ k8sv1.Pod, _ string, _ []string, _, readyChan chan struct{}, portChan chan uint16) error {
				readyChan <- struct{}{}
				portChan <- localPort
				return nil
			}

			memorydump.WaitForMemoryDumpCompleteFn = func(_ kubecli.KubevirtClient, _, _ string, _, _ time.Duration) (string, error) {
				return pvcName, nil
			}

			secret = &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
				},
				Type: k8sv1.SecretTypeOpaque,
				Data: map[string][]byte{
					"token": []byte("test"),
				},
			}
			_, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vme = &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("export-%s-%s", vmName, pvcName),
				},
				Spec: exportv1.VirtualMachineExportSpec{
					TokenSecretRef: &secret.Name,
					Source: k8sv1.TypedLocalObjectReference{
						APIGroup: &k8sv1.SchemeGroupVersion.Group,
						Kind:     "PersistentVolumeClaim",
						Name:     pvcName,
					},
				},
				Status: &exportv1.VirtualMachineExportStatus{
					Phase: exportv1.Ready,
					Links: &exportv1.VirtualMachineExportLinks{
						External: &exportv1.VirtualMachineExportLink{
							Volumes: []exportv1.VirtualMachineExportVolume{{
								Name: pvcName,
								Formats: []exportv1.VirtualMachineExportVolumeFormat{{
									Format: exportv1.KubeVirtGz,
									Url:    server.URL,
								}},
							}},
						},
					},
					TokenSecretRef: &secret.Name,
				},
			}

			service = &k8sv1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-export-" + vme.Name,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: k8sv1.ServiceSpec{
					Ports: []k8sv1.ServicePort{{
						Name: "export",
						Port: int32(443),
					}},
				},
			}

			pod = &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "virt-export-pod-" + vme.Name,
				},
			}
		})

		AfterEach(func() {
			vmexport.WaitForVirtualMachineExportFn = vmexport.WaitForVirtualMachineExport
			vmexport.GetHTTPClientFn = vmexport.GetHTTPClient
			vmexport.HandleHTTPGetRequestFn = vmexport.HandleHTTPGetRequest
			vmexport.RunPortForwardFn = vmexport.RunPortForward
			memorydump.WaitForMemoryDumpCompleteFn = memorydump.WaitForMemoryDumpComplete
		})

		It("should get memory dump and call download memory dump", func() {
			expectVMEndpointMemoryDump("")
			updateVMEStatusOnCreate()
			err := runGetCmd(
				setFlag(memorydump.OutputFileFlag, outputPath),
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "memorydump")).To(HaveLen(1))
		})

		It("should call download memory dump", func() {
			// Create random bytes to test streaming of data works correctly
			const length = 100
			data := make([]byte, length)
			n, err := cryptorand.Read(data)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(length))

			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				n, err := w.Write(data)
				Expect(err).ToNot(HaveOccurred())
				Expect(n).To(Equal(length))
			})

			updateVMEStatusOnCreate()
			err = runDownloadCmd(
				setFlag(memorydump.OutputFileFlag, outputPath),
			)
			Expect(err).ToNot(HaveOccurred())

			outputData, err := os.ReadFile(outputPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(outputData).To(Equal(data))
			Expect(outputData).To(HaveLen(length))
		})

		It("should call download memory dump and decompress succesfully", func() {
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte{
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
				})
				Expect(err).ToNot(HaveOccurred())
			})

			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(memorydump.OutputFileFlag, outputPath),
				setFlag(memorydump.FormatFlag, vmexport.RAW_FORMAT),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("should call download memory dump with port-forward", func(extraArgs ...string) {
			vmexport.HandleHTTPGetRequestFn = func(client kubecli.KubevirtClient, vmexport *exportv1.VirtualMachineExport, downloadUrl string, insecure bool, exportURL string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(Equal("https://127.0.0.1:" + localPortStr))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}, nil
			}

			vme.Status.Links.Internal = vme.Status.Links.External
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Services(metav1.NamespaceDefault).Create(context.Background(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Pods(metav1.NamespaceDefault).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			args := append([]string{
				setFlag(memorydump.PortForwardFlag, "true"),
				setFlag(memorydump.OutputFileFlag, outputPath),
			}, extraArgs...)
			err = runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("with default port-forward"),
			Entry("with port-forward specifying local port", setFlag(memorydump.LocalPortFlag, localPortStr)),
			Entry("with port-forward specifying default number on local port", setFlag(memorydump.LocalPortFlag, "0")),
		)

		It("should fail download memory dump if not completed succesfully", func() {
			const errMsg = "memory dump failed: test err"
			memorydump.WaitForMemoryDumpCompleteFn = func(_ kubecli.KubevirtClient, _, _ string, _, _ time.Duration) (string, error) {
				return pvcName, errors.New(errMsg)
			}

			err := runDownloadCmd(
				setFlag(memorydump.OutputFileFlag, outputPath),
			)
			Expect(err).Should(MatchError(errMsg))
		})
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("--%s=%s", flag, parameter)
}

func runCmd(args ...string) error {
	_args := append([]string{"memory-dump"}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}

func runGetCmd(args ...string) error {
	_args := append([]string{"memory-dump", "get", vmName}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}

func runDownloadCmd(args ...string) error {
	_args := append([]string{"memory-dump", "download", vmName}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}

func runRemoveCmd(args ...string) error {
	_args := append([]string{"memory-dump", "remove", vmName}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}
