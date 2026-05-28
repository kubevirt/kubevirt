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

package virtexportserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"kubevirt.io/virt-template-api/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/storage/export/export"
	"kubevirt.io/kubevirt/pkg/storage/oci"
)

var _ = Describe("OCI export", func() {
	const (
		testToken         = "foo"
		testOCIURI        = "/export.oci.tar"
		exportTokenHeader = "x-kubevirt-export-token"
		testNs            = "test-ns"
	)

	It("should register OCI endpoint when enabled", func() {
		es := newTestServer(testToken)
		es.Paths = &export.ServerPaths{
			OCIURI: testOCIURI,
		}
		es.ociBuilder = &oci.Builder{}
		es.OCIHandler = func(b *oci.Builder) http.Handler {
			return http.HandlerFunc(successHandler)
		}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest(http.MethodGet, httpServer.URL+testOCIURI, http.NoBody)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set(exportTokenHeader, testToken)
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(Equal("OK"))
	})

	DescribeTable("should return 405 for non-GET requests on OCI endpoint", func(method string) {
		handler := ociHTTPHandler(&oci.Builder{})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, testOCIURI, http.NoBody)
		handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusMethodNotAllowed))
		Expect(rec.Header().Get("Allow")).To(Equal(http.MethodGet))
	},
		Entry("POST", http.MethodPost),
		Entry("PUT", http.MethodPut),
		Entry("DELETE", http.MethodDelete),
	)

	It("should not register OCI endpoint when disabled", func() {
		es := newTestServer(testToken)
		es.Paths = &export.ServerPaths{}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest(http.MethodGet, httpServer.URL+testOCIURI, http.NoBody)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set(exportTokenHeader, testToken)
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusNotFound))
		res.Body.Close()
	})

	It("should return 503 while OCI is not ready", func() {
		es := newTestServer(testToken)
		es.ociBuilder = &oci.Builder{}
		es.Paths = &export.ServerPaths{}
		es.initHandler()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, export.ReadinessPath, http.NoBody)
		es.handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
	})

	It("should return 200 from readiness when OCI is ready", func() {
		es := newTestServer(testToken)
		builder := oci.NewVMBuilder([]byte("{}"), "amd64", nil)
		Expect(builder.Prepare(context.Background())).To(Succeed())
		es.ociBuilder = builder
		es.Paths = &export.ServerPaths{}
		es.initHandler()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, export.ReadinessPath, http.NoBody)
		es.handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("should return 200 from readiness when OCI is not enabled", func() {
		es := newTestServer(testToken)
		es.Paths = &export.ServerPaths{}
		es.initHandler()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, export.ReadinessPath, http.NoBody)
		es.handler.ServeHTTP(rec, req)
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	Context("prepareVMConfig", func() {
		const (
			vmName          = "test-vm"
			dvName          = "rootdisk-dv"
			userData        = "#cloud-config"
			labelKey        = "app"
			labelValue      = "test"
			annotationKey   = "note"
			annotationValue = "value"
		)

		var vm *virtv1.VirtualMachine

		prepareAndUnmarshal := func() virtv1.VirtualMachine {
			data, err := prepareVMConfig(vm)
			Expect(err).ToNot(HaveOccurred())
			var out virtv1.VirtualMachine
			Expect(json.Unmarshal(data, &out)).To(Succeed())
			return out
		}

		BeforeEach(func() {
			vmi := libvmi.New(
				libvmi.WithName(vmName),
				libvmi.WithNamespace(testNs),
				libvmi.WithDataVolume("rootdisk", dvName),
				libvmi.WithCloudInitNoCloud(cloudinit.WithNoCloudUserData(userData)),
			)
			vm = libvmi.NewVirtualMachine(vmi,
				libvmi.WithLabels(map[string]string{labelKey: labelValue}),
				libvmi.WithAnnotations(map[string]string{annotationKey: annotationValue}),
				libvmi.WithDataVolumeTemplate(&cdiv1.DataVolume{
					ObjectMeta: metav1.ObjectMeta{Name: dvName},
				}),
			)
			vm.UID = "abc-123"
			vm.ResourceVersion = "42"
			vm.Generation = 3
			vm.CreationTimestamp = metav1.Now()
			vm.ManagedFields = []metav1.ManagedFieldsEntry{{Manager: "test"}}
			vm.OwnerReferences = []metav1.OwnerReference{{Name: "owner"}}
			vm.Finalizers = []string{"test-finalizer"}
			vm.Status = virtv1.VirtualMachineStatus{
				PrintableStatus: virtv1.VirtualMachineStatusRunning,
				Ready:           true,
			}
		})

		It("should set apiVersion and kind", func() {
			out := prepareAndUnmarshal()
			Expect(out.APIVersion).To(Equal(virtv1.GroupVersion.String()))
			Expect(out.Kind).To(Equal("VirtualMachine"))
		})

		It("should strip namespace", func() {
			out := prepareAndUnmarshal()
			Expect(out.Namespace).To(BeEmpty())
		})

		It("should preserve labels and annotations", func() {
			out := prepareAndUnmarshal()
			Expect(out.Name).To(Equal(vmName))
			Expect(out.Labels).To(HaveKeyWithValue(labelKey, labelValue))
			Expect(out.Annotations).To(HaveKeyWithValue(annotationKey, annotationValue))
		})

		It("should strip DataVolumeTemplates", func() {
			out := prepareAndUnmarshal()
			Expect(out.Spec.DataVolumeTemplates).To(BeNil())
		})

		It("should replace DataVolume sources with PVC sources", func() {
			out := prepareAndUnmarshal()
			rootVol := out.Spec.Template.Spec.Volumes[0]
			Expect(rootVol.DataVolume).To(BeNil())
			Expect(rootVol.PersistentVolumeClaim).ToNot(BeNil())
			Expect(rootVol.PersistentVolumeClaim.ClaimName).To(Equal(dvName))
		})

		It("should not touch non-DataVolume volume sources", func() {
			out := prepareAndUnmarshal()
			cloudVol := out.Spec.Template.Spec.Volumes[len(out.Spec.Template.Spec.Volumes)-1]
			Expect(cloudVol.CloudInitNoCloud).ToNot(BeNil())
			Expect(cloudVol.CloudInitNoCloud.UserData).To(Equal(userData))
		})
	})

	Context("prepareVMTemplateConfig", func() {
		const (
			tplName = "test-template"
		)

		createTemplate := func(arch string) *v1beta1.VirtualMachineTemplate {
			vm := &virtv1.VirtualMachine{
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Architecture: arch,
						},
					},
				},
			}
			vmJSON, err := json.Marshal(vm)
			Expect(err).ToNot(HaveOccurred())
			return &v1beta1.VirtualMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tplName,
					Namespace: testNs,
				},
				Spec: v1beta1.VirtualMachineTemplateSpec{
					VirtualMachine: &runtime.RawExtension{Raw: vmJSON},
					Parameters: []v1beta1.Parameter{
						{Name: "VM_NAME", Value: "my-vm"},
					},
				},
			}
		}

		It("should set APIVersion and Kind", func() {
			tpl := createTemplate("amd64")
			configJSON, err := prepareVMTemplateConfig(tpl)
			Expect(err).ToNot(HaveOccurred())

			var out v1beta1.VirtualMachineTemplate
			Expect(json.Unmarshal(configJSON, &out)).To(Succeed())
			Expect(out.APIVersion).To(Equal(v1beta1.GroupVersion.String()))
			Expect(out.Kind).To(Equal("VirtualMachineTemplate"))
		})

		It("should extract architecture from embedded VM", func() {
			tpl := createTemplate("arm64")
			Expect(extractArchitectureFromVMTemplate(tpl)).To(Equal("arm64"))
		})

		It("should resolve architecture from template parameter", func() {
			tpl := createTemplate("${ARCH}")
			tpl.Spec.Parameters = []v1beta1.Parameter{
				{Name: "ARCH", Value: "arm64"},
			}
			Expect(extractArchitectureFromVMTemplate(tpl)).To(Equal("arm64"))
		})

		It("should return empty architecture when parameter is unresolved", func() {
			tpl := createTemplate("${ARCH}")
			Expect(extractArchitectureFromVMTemplate(tpl)).To(BeEmpty())
		})

		It("should return empty architecture when embedded VM has none", func() {
			tpl := createTemplate("")
			Expect(extractArchitectureFromVMTemplate(tpl)).To(BeEmpty())
		})
	})
})
