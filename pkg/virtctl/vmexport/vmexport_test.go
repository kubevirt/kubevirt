/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package vmexport_test

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
	"github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	fakek8sclient "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
	"kubevirt.io/kubevirt/pkg/virtctl/vmexport"
)

const vmeName = "test-vme"

var _ = Describe("vmexport", func() {
	const (
		pvcName    = "test-pvc"
		volumeName = "test-volume"
	)

	var (
		kubeClient *fakek8sclient.Clientset
		virtClient *kubevirtfake.Clientset
		server     *httptest.Server
		outputPath string

		vme    *exportv1.VirtualMachineExport
		secret *k8sv1.Secret
	)

	vmeStatusReady := func(volumes []exportv1.VirtualMachineExportVolume) *exportv1.VirtualMachineExportStatus {
		return &exportv1.VirtualMachineExportStatus{
			Phase: exportv1.Ready,
			Links: &exportv1.VirtualMachineExportLinks{
				External: &exportv1.VirtualMachineExportLink{
					Volumes: volumes,
				},
			},
			TokenSecretRef: &secret.Name,
		}
	}

	BeforeEach(func() {
		kubeClient = fakek8sclient.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()

		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.GetK8sClientFromClientConfig = kubecli.GetMockK8sClientFromClientConfig
		kubecli.MockK8sClientInstance = kubeClient
		kubecli.MockKubevirtClientInstance.EXPECT().VirtualMachineExport(metav1.NamespaceDefault).Return(virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault)).AnyTimes()

		server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		outputPath = filepath.Join(GinkgoT().TempDir(), "disk.img")

		vmexport.WaitForVirtualMachineExportFn = func(_ kubecli.KubevirtClient, _ *vmexport.VMExportInfo, _, _ time.Duration) error {
			return nil
		}
		vmexport.GetHTTPClientFn = func(_ *http.Transport, _ bool) *http.Client {
			DeferCleanup(server.Close)
			return server.Client()
		}

		secret = &k8sv1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "secret-" + vmeName,
			},
			Type: k8sv1.SecretTypeOpaque,
			Data: map[string][]byte{
				"token": []byte("test"),
			},
		}

		vme = &exportv1.VirtualMachineExport{
			ObjectMeta: metav1.ObjectMeta{
				Name: vmeName,
			},
			Spec: exportv1.VirtualMachineExportSpec{
				TokenSecretRef: &secret.Name,
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: &k8sv1.SchemeGroupVersion.Group,
					Kind:     "PersistentVolumeClaim",
					Name:     pvcName,
				},
			},
		}
	})

	AfterEach(func() {
		vmexport.WaitForVirtualMachineExportFn = vmexport.WaitForVirtualMachineExport
		vmexport.GetHTTPClientFn = vmexport.GetHTTPClient
		vmexport.HandleHTTPGetRequestFn = vmexport.HandleHTTPGetRequest
		vmexport.RunPortForwardFn = vmexport.RunPortForward
	})

	Context("VMExport fails", func() {
		It("VirtualMachineExport already exists when using 'create'", func() {
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runCreateCmd(
				setFlag(vmexport.PVC_FLAG, pvcName),
			)
			Expect(err).To(MatchError(ContainSubstring("VirtualMachineExport 'default/test-vme' already exists")))
		})

		It("VirtualMachineExport doesn't exist when using 'download' without source type", func() {
			err := runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get 'default/test-vme' VirtualMachineExport")))
		})

		It("VirtualMachineExport processing fails when using 'download'", func() {
			const errMsg = "processing failed: Test error"
			vmexport.WaitForVirtualMachineExportFn = func(_ kubecli.KubevirtClient, _ *vmexport.VMExportInfo, _, _ time.Duration) error {
				return errors.New(errMsg)
			}

			err := runDownloadCmd(
				setFlag(vmexport.PVC_FLAG, pvcName),
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(errMsg))
		})

		It("VirtualMachineExport download fails when there's no volume available", func() {
			vme.Status = vmeStatusReady(nil)
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to access the volume info from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vme.Name)))
		})

		It("VirtualMachineExport download fails when the volumes have a different name than expected", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{
				{
					Name: "no-test-volume",
					Formats: []exportv1.VirtualMachineExportVolumeFormat{{
						Format: exportv1.KubeVirtRaw,
						Url:    server.URL,
					}},
				},
				{
					Name: "no-test-volume-2",
					Formats: []exportv1.VirtualMachineExportVolumeFormat{{
						Format: exportv1.KubeVirtRaw,
						Url:    server.URL,
					}},
				},
			})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vme.Name)))
		})

		It("VirtualMachineExport download fails when there are multiple volumes and no volume name has been specified", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{
				{
					Name: volumeName,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{{
						Format: exportv1.KubeVirtRaw,
						Url:    server.URL,
					}},
				},
				{
					Name: "no-test-volume",
					Formats: []exportv1.VirtualMachineExportVolumeFormat{{
						Format: exportv1.KubeVirtRaw,
						Url:    server.URL,
					}},
				},
			})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("detected more than one downloadable volume in '%s/%s' VirtualMachineExport: Select the expected volume using the --volume flag", metav1.NamespaceDefault, vme.Name)))
		})

		It("VirtualMachineExport download fails when no format is available", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{Name: volumeName}})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vme.Name)))
		})

		It("VirtualMachineExport download fails when the only available format is incompatible with download", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.Dir,
					Url:    server.URL,
				}},
			}})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL from '%s/%s' VirtualMachineExport", metav1.NamespaceDefault, vme.Name)))
		})

		It("VirtualMachineExport download fails when the secret token is not attainable", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    server.URL,
				}},
			}})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError(ContainSubstring("secrets \"%s\" not found", secret.Name)))
		})

		It("VirtualMachineExport download fails when readiness timeout", func() {
			vme.Status = &exportv1.VirtualMachineExportStatus{
				Phase: exportv1.Pending,
			}
			vmexport.WaitForVirtualMachineExportFn = vmexport.WaitForVirtualMachineExport

			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.PVC_FLAG, pvcName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				setFlag(vmexport.READINESS_TIMEOUT_FLAG, "1ns"),
			)
			Expect(err).To(MatchError("context deadline exceeded"))
		})

		It("VirtualMachineExport retries until failure if the server returns a bad status", func() {
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			})

			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    server.URL,
				}},
			}})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				setFlag(vmexport.RETRY_FLAG, "2"),
			)
			Expect(err).To(MatchError("retry count reached, exiting unsuccesfully"))
		})

		It("VirtualMachineExport succeeds after retrying due to bad status", func() {
			first := true
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if first {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				first = false
			})

			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    server.URL,
				}},
			}})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				setFlag(vmexport.RETRY_FLAG, "2"),
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
			Entry("Missing arg", "accepts 2 arg(s), received 1", runCmd, vmexport.CREATE, setFlag(vmexport.PVC_FLAG, "test")),
			Entry("More arguments than expected create", "accepts 2 arg(s), received 3", runCmd, vmexport.CREATE, vmexport.DELETE, "test"),
			Entry("More arguments than expected download and 'manifest'", "accepts 2 arg(s), received 3", runCmd, vmexport.DOWNLOAD, vmexport.DELETE, vmexport.MANIFEST_FLAG, "test"),
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
		DescribeTable("VirtualMachineExport is created successfully", func(flag, kind string) {
			const name = "test"
			err := runCreateCmd(setFlag(flag, name))
			Expect(err).ToNot(HaveOccurred())

			vme, err = virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vme.Spec.Source.Kind).To(Equal(kind))
			Expect(vme.Spec.Source.Name).To(Equal(name))

			secret, err := kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Get(context.Background(), secret.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(secret.OwnerReferences).To(
				ConsistOf(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"BlockOwnerDeletion": HaveValue(BeFalse()),
				})), "owner ref BlockOwnerDeletion should be false for secret",
			)
		},
			Entry("using PVC source", vmexport.PVC_FLAG, "PersistentVolumeClaim"),
			Entry("using Snapshot source", vmexport.SNAPSHOT_FLAG, "VirtualMachineSnapshot"),
			Entry("using VM source", vmexport.VM_FLAG, "VirtualMachine"),
		)

		DescribeTable("Delete command runs successfully", func(exists bool) {
			if exists {
				_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(runDeleteCmd()).To(Succeed())
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"))
		},
			Entry("when VME exists", true),
			Entry("when VME does not exist", false),
		)

		It("VirtualMachineExport doesn't exist when using 'delete'", func() {
			Expect(runDeleteCmd()).To(Succeed())
		})

		It("Succesfully download from an already existing VirtualMachineExport", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtGz,
					Url:    server.URL,
				}},
			}})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("Succesfully create and download a VirtualMachineExport in different steps", func(expected bool, extraArgs ...string) {
			err := runCreateCmd(setFlag(vmexport.PVC_FLAG, pvcName))
			Expect(err).ToNot(HaveOccurred())

			vme, err = virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtGz,
					Url:    server.URL,
				}}},
			})
			vme, err = virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Update(context.Background(), vme, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			args := append([]string{
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				vmexport.INSECURE_FLAG,
			}, extraArgs...)
			err = runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())

			vme, err = virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			if expected {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"))
			}
		},
			Entry("and expect retained VME (default behavior)", true),
			Entry("and expect retained VME (using flag)", true, vmexport.KEEP_FLAG),
			Entry("and expect deleted VME", false, vmexport.DELETE_FLAG),
		)

		Context("Successfully create and download", func() {
			updateVMEStatusOnCreate := func(format exportv1.ExportVolumeFormat) {
				virtClient.Fake.PrependReactor("create", "virtualmachineexports", func(action k8stesting.Action) (bool, runtime.Object, error) {
					create, ok := action.(k8stesting.CreateAction)
					Expect(ok).To(BeTrue())
					vme, ok := create.GetObject().(*exportv1.VirtualMachineExport)
					Expect(ok).To(BeTrue())
					vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
						Name: volumeName,
						Formats: []exportv1.VirtualMachineExportVolumeFormat{{
							Format: format,
							Url:    server.URL,
						}}},
					})
					return false, vme, nil
				})
			}

			DescribeTable("a VirtualMachineExport", func(flag string) {
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

				updateVMEStatusOnCreate(exportv1.KubeVirtGz)
				err = runDownloadCmd(
					setFlag(flag, "source"),
					setFlag(vmexport.VOLUME_FLAG, volumeName),
					setFlag(vmexport.OUTPUT_FLAG, outputPath),
					vmexport.INSECURE_FLAG,
				)
				Expect(err).ToNot(HaveOccurred())

				outputData, err := os.ReadFile(outputPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(outputData).To(Equal(data))
				Expect(outputData).To(HaveLen(length))
			},
				Entry("using PVC source", vmexport.PVC_FLAG),
				Entry("using Snapshot source", vmexport.SNAPSHOT_FLAG),
				Entry("using VM source", vmexport.VM_FLAG),
			)

			It("a VirtualMachineExport with raw format", func() {
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

				updateVMEStatusOnCreate(exportv1.KubeVirtGz)
				err := runDownloadCmd(
					setFlag(vmexport.FORMAT_FLAG, vmexport.RAW_FORMAT),
					setFlag(vmexport.PVC_FLAG, pvcName),
					setFlag(vmexport.VOLUME_FLAG, volumeName),
					setFlag(vmexport.OUTPUT_FLAG, outputPath),
					vmexport.INSECURE_FLAG,
				)
				Expect(err).ToNot(HaveOccurred())
			})

			It("a VirtualMachineExport without decompressing is url is already raw", func() {
				updateVMEStatusOnCreate(exportv1.KubeVirtRaw)
				err := runDownloadCmd(
					setFlag(vmexport.FORMAT_FLAG, vmexport.RAW_FORMAT),
					setFlag(vmexport.PVC_FLAG, pvcName),
					setFlag(vmexport.VOLUME_FLAG, volumeName),
					setFlag(vmexport.OUTPUT_FLAG, outputPath),
					vmexport.INSECURE_FLAG,
				)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		It("Succesfully download a VirtualMachineExport with just 'raw' links", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    server.URL,
				}}},
			})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				vmexport.INSECURE_FLAG,
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when the volume has a different name than expected but there's only one volume", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: "no-test-volume",
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    server.URL,
				}}},
			})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download succeeds when there's only one volume and no --volume has been specified", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: "no-test-volume",
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    server.URL,
				}}},
			})
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Succesfully create VirtualMachineExport with TTL", func() {
			ttl := metav1.Duration{Duration: 2 * time.Minute}
			err := runCreateCmd(
				setFlag(vmexport.PVC_FLAG, pvcName),
				setFlag(vmexport.TTL_FLAG, ttl.Duration.String()),
			)
			Expect(err).ToNot(HaveOccurred())

			vme, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(*vme.Spec.TTLDuration).To(Equal(ttl))
		})

		It("Succesfully create VirtualMachineExport with custom labels and annotations", func() {
			const (
				labelKey        = "label-key"
				labelValue      = "label-value"
				annotationKey   = "annotation-key"
				annotationValue = "annotation-key"
			)

			err := runCreateCmd(
				setFlag(vmexport.PVC_FLAG, pvcName),
				setFlag(vmexport.LABELS_FLAG, fmt.Sprintf("%s=%s", labelKey, labelValue)),
				setFlag(vmexport.ANNOTATIONS_FLAG, fmt.Sprintf("%s=%s", annotationKey, annotationValue)),
			)
			Expect(err).ToNot(HaveOccurred())

			vme, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vme.Labels).To(HaveKeyWithValue(labelKey, labelValue))
			Expect(vme.Annotations).To(HaveKeyWithValue(annotationKey, annotationValue))
		})
	})

	Context("Manifest", func() {
		const (
			manifestUrl = "/test/all"
			secretUrl   = "/test/secret"
		)

		DescribeTable("should successfully create VirtualMachineExport if proper arg supplied", func(expected string, extraArgs ...string) {
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.String()).To(Equal(manifestUrl))
				Expect(r.Header).To(HaveKeyWithValue("Accept", ConsistOf(expected)))
				w.WriteHeader(http.StatusOK)
			})

			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtGz,
					Url:    server.URL,
				}}},
			})
			vme.Status.Links.External.Manifests = append(vme.Status.Links.External.Manifests,
				exportv1.VirtualMachineExportManifest{
					Type: exportv1.AllManifests,
					Url:  server.URL + manifestUrl,
				},
			)
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			args := append([]string{
				vmexport.MANIFEST_FLAG,
				setFlag(vmexport.VM_FLAG, "test"),
			}, extraArgs...)
			err = runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("no output format arguments", vmexport.APPLICATION_YAML),
			Entry("output format json", vmexport.APPLICATION_JSON, setFlag(vmexport.OUTPUT_FORMAT_FLAG, vmexport.OUTPUT_FORMAT_JSON)),
			Entry("output format yaml", vmexport.APPLICATION_YAML, setFlag(vmexport.OUTPUT_FORMAT_FLAG, vmexport.OUTPUT_FORMAT_YAML)),
		)

		DescribeTable("should call both manifest and secret url if argument supplied", func(withIncludeSecret bool) {
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if withIncludeSecret {
					Expect(r.URL.String()).To(BeElementOf(manifestUrl, secretUrl))
				} else {
					Expect(r.URL.String()).To(Equal(manifestUrl))
				}
				w.WriteHeader(http.StatusOK)
			})

			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtGz,
					Url:    server.URL,
				}}},
			})
			vme.Status.Links.External.Manifests = append(vme.Status.Links.External.Manifests,
				exportv1.VirtualMachineExportManifest{
					Type: exportv1.AllManifests,
					Url:  server.URL + manifestUrl,
				},
				exportv1.VirtualMachineExportManifest{
					Type: exportv1.AuthHeader,
					Url:  server.URL + secretUrl,
				},
			)
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			args := []string{
				vmexport.MANIFEST_FLAG,
				setFlag(vmexport.VM_FLAG, "test"),
			}
			if withIncludeSecret {
				args = append(args, vmexport.INCLUDE_SECRET_FLAG)
			}
			err = runDownloadCmd(args...)
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("without --include-secret", false),
			Entry("with --include-secret", true),
		)

		It("should error if http status is error", func() {
			server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.String()).To(BeElementOf(manifestUrl, secretUrl))
				w.WriteHeader(http.StatusBadRequest)
			})

			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtGz,
					Url:    server.URL,
				}}},
			})
			vme.Status.Links.External.Manifests = append(vme.Status.Links.External.Manifests,
				exportv1.VirtualMachineExportManifest{
					Type: exportv1.AllManifests,
					Url:  server.URL + manifestUrl,
				}, exportv1.VirtualMachineExportManifest{
					Type: exportv1.AuthHeader,
					Url:  server.URL + secretUrl,
				},
			)
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			_, err = kubeClient.CoreV1().Secrets(metav1.NamespaceDefault).Create(context.Background(), secret, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				vmexport.MANIFEST_FLAG,
				setFlag(vmexport.VM_FLAG, "test"),
			)
			Expect(err).To(MatchError("retry count reached, exiting unsuccesfully"))
		})
	})

	Context("Port-forward", func() {
		const (
			localPort    = uint16(5432)
			localPortStr = "5432"
		)
		var (
			service *k8sv1.Service
			pod     *k8sv1.Pod
		)

		BeforeEach(func() {
			vmexport.RunPortForwardFn = func(_ kubecli.KubevirtClient, _ kubernetes.Interface, _ k8sv1.Pod, _ string, _ []string, _, readyChan chan struct{}, portChan chan uint16) error {
				readyChan <- struct{}{}
				portChan <- localPort
				return nil
			}

			vme = &exportv1.VirtualMachineExport{
				ObjectMeta: metav1.ObjectMeta{
					Name: vmeName,
				},
				Spec: exportv1.VirtualMachineExportSpec{
					TokenSecretRef: &secret.Name,
					Source: k8sv1.TypedLocalObjectReference{
						APIGroup: &v1.SchemeGroupVersion.Group,
						Kind:     "VirtualMachine",
						Name:     "test-vm",
					},
				},
				Status: vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
					Name: volumeName,
					Formats: []exportv1.VirtualMachineExportVolumeFormat{{
						Format: exportv1.KubeVirtRaw,
						Url:    server.URL,
					}}},
				}),
			}
			_, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Create(context.Background(), vme, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

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
			_, err = kubeClient.CoreV1().Services(metav1.NamespaceDefault).Create(context.Background(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			pod = &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "virt-export-pod-" + vme.Name,
				},
			}
			_, err = kubeClient.CoreV1().Pods(metav1.NamespaceDefault).Create(context.Background(), pod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("VirtualMachineExport download fails when using port-forward with an invalid port", func() {
			service, err := kubeClient.CoreV1().Services(metav1.NamespaceDefault).Get(context.Background(), service.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			service.Spec.Ports[0].Port = 321
			_, err = kubeClient.CoreV1().Services(metav1.NamespaceDefault).Update(context.Background(), service, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				vmexport.PORT_FORWARD_FLAG,
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).To(MatchError("Service virt-export-test-vme does not have a service port 443"))
		})

		It("VirtualMachineExport download with port-forward fails when the service doesn't have a valid pod ", func() {
			err := kubeClient.CoreV1().Pods(metav1.NamespaceDefault).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				vmexport.PORT_FORWARD_FLAG,
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
				setFlag(vmexport.LOCAL_PORT_FLAG, localPortStr),
			)
			Expect(err).To(MatchError("no pods found for the service virt-export-test-vme"))
		})

		It("VirtualMachineExport download with port-forward succeeds", func() {
			vmexport.HandleHTTPGetRequestFn = func(_ kubernetes.Interface, _ *exportv1.VirtualMachineExport, downloadUrl string, _ bool, _ string, _ map[string]string) (*http.Response, error) {
				Expect(downloadUrl).To(Equal("https://127.0.0.1:" + localPortStr))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("data")),
				}, nil
			}

			vme, err := virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Get(context.Background(), vme.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			vme.Status.Links.Internal = vme.Status.Links.External
			_, err = virtClient.ExportV1beta1().VirtualMachineExports(metav1.NamespaceDefault).Update(context.Background(), vme, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = runDownloadCmd(
				vmexport.PORT_FORWARD_FLAG,
				setFlag(vmexport.VOLUME_FLAG, volumeName),
				setFlag(vmexport.OUTPUT_FLAG, outputPath),
			)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("getUrlFromVirtualMachineExport", func() {
		It("Should get compressed URL even when there's multiple URLs", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
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
				}},
			})
			vmeInfo := &vmexport.VMExportInfo{
				Name:       vme.Name,
				VolumeName: volumeName,
			}

			url, err := vmexport.GetUrlFromVirtualMachineExport(vme, vmeInfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("compressed"))
		})

		It("Should get raw URL when there's no other option", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.KubeVirtRaw,
					Url:    "raw",
				}}},
			})
			vmeInfo := &vmexport.VMExportInfo{
				Name:       vme.Name,
				VolumeName: volumeName,
			}

			url, err := vmexport.GetUrlFromVirtualMachineExport(vme, vmeInfo)
			Expect(err).ToNot(HaveOccurred())
			Expect(url).Should(Equal("raw"))
		})

		It("Should not get any URL when there's no valid options", func() {
			vme.Status = vmeStatusReady([]exportv1.VirtualMachineExportVolume{{
				Name: volumeName,
				Formats: []exportv1.VirtualMachineExportVolumeFormat{{
					Format: exportv1.Dir,
					Url:    server.URL,
				}}},
			})
			vmeInfo := &vmexport.VMExportInfo{
				Name:       vme.Name,
				VolumeName: volumeName,
			}

			url, err := vmexport.GetUrlFromVirtualMachineExport(vme, vmeInfo)
			Expect(err).To(MatchError(ContainSubstring("unable to get a valid URL")))
			Expect(url).To(Equal(""))
		})
	})
})

func setFlag(flag, parameter string) string {
	return fmt.Sprintf("%s=%s", flag, parameter)
}

func runCmd(args ...string) error {
	_args := append([]string{"vmexport"}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}

func runCreateCmd(args ...string) error {
	_args := append([]string{"vmexport", vmexport.CREATE, vmeName}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}

func runDeleteCmd(args ...string) error {
	_args := append([]string{"vmexport", vmexport.DELETE, vmeName}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}

func runDownloadCmd(args ...string) error {
	_args := append([]string{"vmexport", vmexport.DOWNLOAD, vmeName}, args...)
	return testing.NewRepeatableVirtctlCommand(_args...)()
}
