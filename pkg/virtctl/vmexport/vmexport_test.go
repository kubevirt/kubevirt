package vmexport_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"github.com/onsi/gomega/gstruct"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/virtctl/utils"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
	"kubevirt.io/kubevirt/tests/clientcmd"
)

const (
	commandName = "vmexport"
	vmeName     = "test-vme"
	volumeName  = "test-volume"
	secretName  = "secret-test-vme"

	manifestUrl = "https://test.something.somewhere/test/all"
	secretUrl   = "https://test.something.somewhere/test/secret"
)

var _ = Describe("vmexport", func() {
	var (
		kubeClient *fakek8sclient.Clientset
		virtClient *kubevirtfake.Clientset
		server     *httptest.Server
	)

	BeforeEach(func() {
		kubeClient = fakek8sclient.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()

		kubeClient.Fake.PrependReactor("create", "secrets", func(action testing.Action) (bool, runtime.Object, error) {
			create, ok := action.(testing.CreateAction)
			Expect(ok).To(BeTrue())
			secret, ok := create.GetObject().(*v1.Secret)
			Expect(ok).To(BeTrue())
			Expect(secret.OwnerReferences).To(
				ConsistOf(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"BlockOwnerDeletion": HaveValue(BeFalse()),
				})), "owner ref BlockOwnerDeletion should be false for secret",
			)
			return true, secret, nil
		})

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().StorageV1().Return(kubeClient.StorageV1()).AnyTimes()
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineExport(metav1.NamespaceDefault).Return(virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault)).AnyTimes()

		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		vmexport.ExportProcessingComplete = utils.WaitExportCompleteDefault
		vmexport.SetHTTPClientCreator(func(*http.Transport, bool) *http.Client {
			return server.Client()
		})
		vmexport.SetPortForwarder(func(_ kubecli.KubevirtClient, _ k8sv1.Pod, _ string, _ []string, _, readyChan chan struct{}, portChan chan uint16) error {
			readyChan <- struct{}{}
			portChan <- uint16(5432)
			return nil
		})
	})

	AfterEach(func() {
		server.Close()
	})

	Context("VMExport fails", func() {
		It("VirtualMachineExport already exists when using 'create'", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			err := runCreateCmd(setFlag(vmexport.PVC_FLAG, "test-pvc"))
			Expect(err).To(MatchError(ContainSubstring("VirtualMachineExport 'default/test-vme' already exists")))
		})

		It("VirtualMachineExport doesn't exist when using 'download' without source type", func() {
			err := runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "output.img")),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get 'default/test-vme' VirtualMachineExport")))
		})

		It("VirtualMachineExport processing fails when using 'download'", func() {
			vmexport.ExportProcessingComplete = utils.WaitExportCompleteError
			err := runDownloadCmd(
				setFlag(vmexport.PVC_FLAG, "test-pvc"),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).To(MatchError("processing failed: Test error"))
		})

		It("VirtualMachineExport download fails when there's no volume available", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus(nil, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to access the volume info from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmeName)))
		})

		It("VirtualMachineExport download fails when the volumes have a different name than expected", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
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
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmeName)))
		})

		It("VirtualMachineExport download fails when there are multiple volumes and no volume name has been specified", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
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
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
			)
			Expect(err).To(MatchError(ContainSubstring("detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", metav1.NamespaceDefault, vmeName)))
		})

		It("VirtualMachineExport download fails when no format is available", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{{Name: volumeName}}, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmeName)))
		})

		It("VirtualMachineExport download fails when the only available format is incompatible with download", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.Dir),
				},
			}, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vmeName)))
		})

		It("VirtualMachineExport download fails when the secret token is not attainable", func() {
			// Add new reactor so the client returns a nil secret
			kubeClient.Fake.PrependReactor("create", "secrets", func(_ testing.Action) (bool, runtime.Object, error) { return false, nil, nil })

			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).To(MatchError(ContainSubstring("secrets \"%s\" not found", secretName)))
		})

		It("VirtualMachineExport retries until failure if the server returns a bad status", func() {
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})

			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			utils.HandleSecretGet(kubeClient, secretName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName), setFlag(vmexport.RETRY_FLAG, "2"),
			)
			Expect(err).To(MatchError("retry count reached, exiting unsuccesfully"))
		})

		It("VirtualMachineExport succeeds after retrying due to bad status", func() {
			count := 0
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if count == 0 {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				count++
			})

			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			utils.HandleSecretGet(kubeClient, secretName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName), setFlag(vmexport.RETRY_FLAG, "2"),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("Bad flag combination", func(expected string, runFn func(args ...string) error, args ...string) {
			Expect(runFn(args...)).To(MatchError(expected))
		},
			Entry("Multiple target types", "if any flags in the group [vm snapshot pvc] are set none of the others can be; [pvc snapshot vm] were all set", runCreateCmd, setFlag(vmexport.PVC_FLAG, "test"), setFlag(vmexport.VM_FLAG, "test2"), setFlag(vmexport.SNAPSHOT_FLAG, "test3")),
			Entry("Retain and delte vmexport", "if any flags in the group [keep-vme delete-vme] are set none of the others can be; [delete-vme keep-vme] were all set", runDownloadCmd, vmexport.DELETE_FLAG, vmexport.KEEP_FLAG),
		)

		DescribeTable("Invalid arguments/flags", func(expected string, runFn func(args ...string) error, args ...string) {
			Expect(runFn(args...)).To(MatchError(expected))
		},
			Entry("No arguments", "accepts 2 arg(s), received 0", runCmd),
			Entry("Missing arg", "accepts 2 arg(s), received 1", runCmd, vmexport.CREATE, setFlag(vmexport.PVC_FLAG, vmeName)),
			Entry("More arguments than expected create", "accepts 2 arg(s), received 3", runCmd, vmexport.CREATE, vmexport.DELETE, vmeName),
			Entry("More arguments than expected download and 'manifest'", "accepts 2 arg(s), received 3", runCmd, vmexport.DOWNLOAD, vmexport.DELETE, vmexport.MANIFEST_FLAG, vmeName),
			Entry("Using 'create' without export type", vmexport.ErrRequiredExportType, runCreateCmd),
			Entry("Using 'create' with invalid flag", fmt.Sprintf(vmexport.ErrIncompatibleFlag, vmexport.INSECURE_FLAG, vmexport.CREATE), runCreateCmd, setFlag(vmexport.PVC_FLAG, "test"), vmexport.INSECURE_FLAG),
			Entry("Using 'delete' with export type", vmexport.ErrIncompatibleExportType, runDeleteCmd, setFlag(vmexport.PVC_FLAG, "test")),
			Entry("Using 'delete' with invalid flag", fmt.Sprintf(vmexport.ErrIncompatibleFlag, vmexport.INSECURE_FLAG, vmexport.DELETE), runDeleteCmd, vmexport.INSECURE_FLAG),
			Entry("Using 'manifest' with pvc flag", fmt.Sprintf(vmexport.ErrIncompatibleFlag, vmexport.PVC_FLAG, vmexport.MANIFEST_FLAG), runDownloadCmd, vmexport.MANIFEST_FLAG, setFlag(vmexport.PVC_FLAG, "test")),
			Entry("Using 'manifest' with volume type", fmt.Sprintf(vmexport.ErrIncompatibleFlag, vmexport.VOLUME_FLAG, vmexport.MANIFEST_FLAG), runDownloadCmd, vmexport.MANIFEST_FLAG, setFlag(vmexport.VM_FLAG, "test"), setFlag(vmexport.VOLUME_FLAG, "volume")),
			Entry("Using 'manifest' with invalid output_format_flag", fmt.Sprintf(vmexport.ErrInvalidValue, vmexport.OUTPUT_FORMAT_FLAG, "json/yaml"), runDownloadCmd, vmexport.MANIFEST_FLAG, setFlag(vmexport.OUTPUT_FORMAT_FLAG, "invalid")),
			Entry("Using 'port-forward' with invalid port", fmt.Sprintf(vmexport.ErrInvalidValue, vmexport.LOCAL_PORT_FLAG, "valid port numbers"), runDownloadCmd, vmexport.PORT_FORWARD_FLAG, setFlag(vmexport.LOCAL_PORT_FLAG, "test")),
			Entry("Using 'format' with invalid download format", fmt.Sprintf(vmexport.ErrInvalidValue, vmexport.FORMAT_FLAG, "gzip/raw"), runDownloadCmd, setFlag(vmexport.FORMAT_FLAG, "test")),
			Entry("Downloading volume without specifying output", fmt.Sprintf("warning: Binary output can mess up your terminal. Use '%s -' to output into stdout anyway or consider '%s <FILE>' to save to a file", vmexport.OUTPUT_FLAG, vmexport.OUTPUT_FLAG), runDownloadCmd),
		)
	})

	Context("VMExport succeeds", func() {
		// Create tests
		It("VirtualMachineExport is created succesfully", func() {
			utils.HandleVMExportCreate(virtClient, nil)
			err := runCreateCmd(setFlag(vmexport.PVC_FLAG, "test-pvc"))
			Expect(err).ToNot(HaveOccurred())
		})

		// Delete tests
		It("VirtualMachineExport is deleted succesfully", func() {
			utils.HandleVMExportDelete(virtClient, vmeName)
			Expect(runDeleteCmd()).To(Succeed())
		})

		It("VirtualMachineExport doesn't exist when using 'delete'", func() {
			Expect(runDeleteCmd()).To(Succeed())
		})

		// Download tests
		It("Succesfully download from an already existing VirtualMachineExport", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "test-pvc")),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("Succesfully create and download a VirtualMachineExport in different steps", func(expected int, arg string) {
			utils.HandleVMExportCreate(virtClient, nil)
			err := runCreateCmd(setFlag(vmexport.PVC_FLAG, "test-pvc"))
			Expect(err).ToNot(HaveOccurred())

			vme, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vmeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)

			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			deletes := 0
			virtClient.Fake.PrependReactor("delete", "virtualmachineexports", func(_ testing.Action) (bool, runtime.Object, error) {
				deletes++
				return true, nil, nil
			})

			args := []string{
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "test-pvc")),
				vmexport.INSECURE_FLAG,
				"-n", metav1.NamespaceDefault,
			}
			if arg != "" {
				args = append(args, arg)
			}
			err = runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(deletes).To(Equal(expected))
		},
			Entry("and expect retained VME (default behavior)", 0, ""),
			Entry("and expect retained VME (using flag)", 0, vmexport.KEEP_FLAG),
			Entry("and expect deleted VME", 1, vmexport.DELETE_FLAG),
		)

		It("Succesfully create and download a VirtualMachineExport", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(virtClient, vme)

			err := runDownloadCmd(
				setFlag(vmexport.PVC_FLAG, "test-pvc"),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "test-pvc")),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create and download a VirtualMachineExport with raw format", func() {
			vmexport.HandleHTTPRequest = func(_ kubecli.KubevirtClient, _ *exportv1.VirtualMachineExport, _ string, _ bool, _ string, _ map[string]string) (*http.Response, error) {
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

			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(virtClient, vme)

			err := runDownloadCmd(
				setFlag(vmexport.FORMAT_FLAG, vmexport.RAW_FORMAT),
				setFlag(vmexport.PVC_FLAG, "test-pvc"),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "test-pvc")),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully download a VirtualMachineExport without decompressing is url is already raw", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportCreate(virtClient, vme)

			err := runDownloadCmd(
				setFlag(vmexport.FORMAT_FLAG, vmexport.RAW_FORMAT),
				setFlag(vmexport.PVC_FLAG, "test-pvc"),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "test-pvc")),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully download a VirtualMachineExport with just 'raw' links", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "test-pvc")),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when the volume has a different name than expected but there's only one volume", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when there's only one volume and no --volume has been specified", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)

			err := runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create VirtualMachineExport with TTL", func() {
			vme := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)

			virtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (bool, runtime.Object, error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				Expect(*vme.Spec.TTLDuration).To(Equal(metav1.Duration{Duration: time.Minute}))
				return true, vme, nil
			})

			err := runCreateCmd(
				setFlag(vmexport.PVC_FLAG, "test-pvc"),
				setFlag(vmexport.TTL_FLAG, "1m"),
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("getUrlFromVirtualMachineExport", func() {
		var vmeinfo *vmexport.VMExportInfo

		BeforeEach(func() {
			vmeinfo = &vmexport.VMExportInfo{
				Name:       vmeName,
				VolumeName: volumeName,
			}
		})

		It("Should get compressed URL even when there's multiple URLs", func() {
			vmExport := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
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
			vmExport := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
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
			vmExport := utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
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
	})

	Context("Manifest", func() {
		DescribeTable("should successfully create VirtualMachineExport if proper arg supplied", func(arg, expected string) {
			vmexport.HandleHTTPRequest = func(_ kubecli.KubevirtClient, _ *exportv1.VirtualMachineExport, downloadUrl string, _ bool, _ string, headers map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(Equal(manifestUrl))
				Expect(headers).To(ContainElements(expected))
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}
				return &resp, nil
			}

			vme := utils.VMExportSpecVM(vmeName, metav1.NamespaceDefault, "test", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vme.Status.Links.External.Manifests = append(vme.Status.Links.External.Manifests, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AllManifests,
				Url:  manifestUrl,
			})
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			utils.HandleSecretGet(kubeClient, secretName)

			virtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (bool, runtime.Object, error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				return true, vme, nil
			})

			args := []string{
				vmexport.MANIFEST_FLAG,
				setFlag(vmexport.VM_FLAG, "test"),
			}
			if arg != "" {
				args = append(args, arg)
			}
			err := runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("no output format arguments", "", vmexport.APPLICATION_YAML),
			Entry("output format json", setFlag(vmexport.OUTPUT_FORMAT_FLAG, vmexport.OUTPUT_FORMAT_JSON), vmexport.APPLICATION_JSON),
			Entry("output format yaml", setFlag(vmexport.OUTPUT_FORMAT_FLAG, vmexport.OUTPUT_FORMAT_YAML), vmexport.APPLICATION_YAML),
		)

		DescribeTable("should call both manifest and secret url if argument supplied", func(arg string, callCount int) {
			calls := 0
			vmexport.HandleHTTPRequest = func(_ kubecli.KubevirtClient, _ *exportv1.VirtualMachineExport, downloadUrl string, _ bool, _ string, _ map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(BeElementOf(manifestUrl, secretUrl))
				calls++
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}
				return &resp, nil
			}

			vme := utils.VMExportSpecVM(vmeName, metav1.NamespaceDefault, "test", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vme.Status.Links.External.Manifests = append(vme.Status.Links.External.Manifests, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AllManifests,
				Url:  manifestUrl,
			}, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AuthHeader,
				Url:  secretUrl,
			})
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			utils.HandleSecretGet(kubeClient, secretName)

			virtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (bool, runtime.Object, error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				return true, vme, nil
			})

			args := []string{
				vmexport.MANIFEST_FLAG,
				setFlag(vmexport.VM_FLAG, "test"),
			}
			if arg != "" {
				args = append(args, arg)
			}
			err := runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())
			Expect(calls).To(Equal(callCount))
		},
			Entry("without --include-secret", "", 1),
			Entry("with --include-secret", vmexport.INCLUDE_SECRET_FLAG, 2),
		)

		It("should error if http status is error", func() {
			vmexport.HandleHTTPRequest = func(_ kubecli.KubevirtClient, _ *exportv1.VirtualMachineExport, downloadUrl string, _ bool, _ string, _ map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(BeElementOf(manifestUrl, secretUrl))
				resp := http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader("")),
				}
				return &resp, nil
			}

			vme := utils.VMExportSpecVM(vmeName, metav1.NamespaceDefault, "test", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    volumeName,
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtGz),
				},
			}, secretName)
			vme.Status.Links.External.Manifests = append(vme.Status.Links.External.Manifests, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AllManifests,
				Url:  manifestUrl,
			}, exportv1.VirtualMachineExportManifest{
				Type: exportv1.AuthHeader,
				Url:  secretUrl,
			})
			utils.HandleVMExportGet(virtClient, vme, vmeName)
			utils.HandleSecretGet(kubeClient, secretName)

			virtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action testing.Action) (bool, runtime.Object, error) {
				create, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())
				vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
				Expect(ok).To(BeTrue())
				return true, vme, nil
			})

			err := runDownloadCmd(
				vmexport.MANIFEST_FLAG,
				setFlag(vmexport.VM_FLAG, "test"),
			)
			Expect(err).To(MatchError("retry count reached, exiting unsuccesfully"))
		})
	})

	Context("Port-forward", func() {
		var (
			vme *exportv1.VirtualMachineExport
		)

		BeforeEach(func() {
			vme = utils.VMExportSpecPVC(vmeName, metav1.NamespaceDefault, "test-pvc", secretName)
			vme.Status = utils.GetVMEStatus([]exportv1.VirtualMachineExportVolume{
				{
					Name:    "no-test-volume",
					Formats: utils.GetExportVolumeFormat(server.URL, exportv1.KubeVirtRaw),
				},
			}, secretName)
			utils.HandleSecretGet(kubeClient, secretName)
			utils.HandleVMExportGet(virtClient, vme, vmeName)
		})

		It("VirtualMachineExport download fails when using port-forward with an invalid port", func() {
			utils.HandleServiceGet(kubeClient, fmt.Sprintf("virt-export-%s", vme.Name), 321)
			utils.HandlePodList(kubeClient, fmt.Sprintf("virt-export-pod-%s", vme.Name))

			err := runDownloadCmd(
				vmexport.PORT_FORWARD_FLAG,
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
			)
			Expect(err).To(MatchError("Service virt-export-test-vme does not have a service port 443"))
		})

		It("VirtualMachineExport download with port-forward fails when the service doesn't have a valid pod ", func() {
			utils.HandleServiceGet(kubeClient, fmt.Sprintf("virt-export-%s", vme.Name), 443)

			err := runDownloadCmd(
				vmexport.PORT_FORWARD_FLAG,
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.LOCAL_PORT_FLAG, "5432"),
			)
			Expect(err).To(MatchError("no pods found for the service virt-export-test-vme"))
		})

		It("VirtualMachineExport download with port-forward succeeds", func() {
			vmexport.HandleHTTPRequest = func(_ kubecli.KubevirtClient, _ *exportv1.VirtualMachineExport, downloadUrl string, _ bool, _ string, _ map[string]string) (*http.Response, error) {
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

			err := runDownloadCmd(
				vmexport.PORT_FORWARD_FLAG,
				setFlag(vmexport.OUTPUT_FLAG, filepath.Join(GinkgoT().TempDir(), "disk.img")),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("%s=%s", flag, parameter)
}

func runCmd(args ...string) error {
	_args := append([]string{commandName}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}

func runCreateCmd(args ...string) error {
	_args := append([]string{commandName, vmexport.CREATE, vmeName}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}

func runDeleteCmd(args ...string) error {
	_args := append([]string{commandName, vmexport.DELETE, vmeName}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}

func runDownloadCmd(args ...string) error {
	_args := append([]string{commandName, vmexport.DOWNLOAD, vmeName}, args...)
	return clientcmd.NewRepeatableVirtctlCommand(_args...)()
}
