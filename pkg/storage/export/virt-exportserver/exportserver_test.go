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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/storage/export/export"
)

const (
	testNamespace = "default"
)

func successHandler(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("OK"))
}

func newTestServer(token string) *exportServer {
	config := ExportServerConfig{
		ArchiveHandler: func(string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		DirHandler: func(string, string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		FileHandler: func(string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		GzipHandler: func(string) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		VmHandler: func([]export.VolumeInfo, func() (string, error), func() (*v1.ConfigMap, error)) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		TokenSecretHandler: func(tgf TokenGetterFunc) http.Handler {
			return http.HandlerFunc(successHandler)
		},
		TokenGetter: func() (string, error) {
			return token, nil
		},
		PermissionChecker: func(string) bool {
			return true
		},
	}
	s := NewExportServer(config)
	return s.(*exportServer)
}

var _ = Describe("exportserver", func() {
	DescribeTable("should handle", func(vmURI string, vi *export.VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Paths = &export.ServerPaths{VMURI: vmURI}
		if vi != nil {
			es.Paths.Volumes = []export.VolumeInfo{*vi}
		}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri, nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("x-kubevirt-export-token", token)
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(Equal("OK"))
	},
		Entry("archive URI",
			"",
			&export.VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			"",
			&export.VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
		Entry("VM definition URI",
			"/manifest",
			nil,
			"/internal/manifest",
		),
		Entry("Token Secret URI",
			"/manifest/secret",
			nil,
			"/internal/manifest/secret",
		),
	)

	DescribeTable("should handle (query param version)", func(vmURI string, vi *export.VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Paths = &export.ServerPaths{VMURI: vmURI}
		if vi != nil {
			es.Paths.Volumes = []export.VolumeInfo{*vi}
		}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri+"?x-kubevirt-export-token="+token, nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusOK))
		defer res.Body.Close()
		out, err := io.ReadAll(res.Body)
		Expect(err).ToNot(HaveOccurred())
		Expect(string(out)).To(Equal("OK"))
	},
		Entry("archive URI",
			"",
			&export.VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			"",
			&export.VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
		Entry("VM definition URI",
			"/manifest",
			nil,
			"/internal/manifest",
		),
		Entry("Token Secret URI",
			"/manifest/secret",
			nil,
			"/internal/manifest/secret",
		),
	)

	DescribeTable("should fail bad token", func(vmURI string, vi *export.VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Paths = &export.ServerPaths{VMURI: vmURI}
		if vi != nil {
			es.Paths.Volumes = []export.VolumeInfo{*vi}
		}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri, nil)
		Expect(err).ToNot(HaveOccurred())
		req.Header.Set("x-kubevirt-export-token", "bar")
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
	},
		Entry("archive URI",
			"",
			&export.VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			"",
			&export.VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
		Entry("VM definition URI",
			"/manifest",
			nil,
			"/external/manifest",
		),
		Entry("Token Secret URI",
			"/manifest/secret",
			nil,
			"/external/manifest/secret",
		),
	)

	DescribeTable("should fail bad token (query param version)", func(vmURI string, vi *export.VolumeInfo, uri string) {
		token := "foo"
		es := newTestServer(token)
		es.Paths = &export.ServerPaths{VMURI: vmURI}
		if vi != nil {
			es.Paths.Volumes = []export.VolumeInfo{*vi}
		}
		es.initHandler()

		httpServer := httptest.NewServer(es.handler)
		defer httpServer.Close()

		client := http.Client{}
		req, err := http.NewRequest("GET", httpServer.URL+uri+"?x-kubevirt-export-token=bar", nil)
		Expect(err).ToNot(HaveOccurred())
		res, err := client.Do(req)
		Expect(err).ToNot(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusUnauthorized))
	},
		Entry("archive URI",
			"",
			&export.VolumeInfo{Path: "/tmp", ArchiveURI: "/volume/v1/disk.tar.gz"},
			"/volume/v1/disk.tar.gz",
		),
		Entry("dir URI",
			"",
			&export.VolumeInfo{Path: "/tmp", DirURI: "/volume/v1/dir/"},
			"/volume/v1/dir/",
		),
		Entry("raw URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawURI: "/volume/v1/disk.img"},
			"/volume/v1/disk.img",
		),
		Entry("raw gz URI",
			"",
			&export.VolumeInfo{Path: "/tmp", RawGzURI: "/volume/v1/disk.img.gz"},
			"/volume/v1/disk.img.gz",
		),
		Entry("VM definition URI",
			"/manifest",
			nil,
			"/external/manifest",
		),
		Entry("Token Secret URI",
			"/manifest/secret",
			nil,
			"/internal/manifest/secret",
		),
	)

	Context("Vm handler", func() {
		var (
			orgGetExportName       = getExportName
			orgGetInternalBasePath = getInternalBasePath
			orgGetExpandedVM       = getExpandedVM
			orgGetDataVolumes      = getDataVolumes
			orgGetExternalBasePath = getExternalBasePath
		)

		verifyCmYaml := func(yamlString string) {
			resCm := &v1.ConfigMap{}
			err := yaml.Unmarshal([]byte(yamlString), resCm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resCm.Name).To(Equal("test-ca-configmap"))
			Expect(resCm.Data["ca.crt"]).To(Equal("cert data"))
		}

		verifyCmJson := func(jsonBytes []byte) {
			resCm := &v1.ConfigMap{}
			err := json.Unmarshal(jsonBytes, resCm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resCm.Name).To(Equal("test-ca-configmap"))
			Expect(resCm.Data["ca.crt"]).To(Equal("cert data"))
		}

		getBasePath := func() (string, error) {
			return "base_path", nil
		}

		getErrorBasePath := func() (string, error) {
			return "", fmt.Errorf("base path error")
		}

		getCaConfigMap := func() (*v1.ConfigMap, error) {
			return &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ca-configmap",
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"ca.crt": "cert data",
				},
			}, nil
		}
		getErrorCaConfigMap := func() (*v1.ConfigMap, error) {
			return nil, fmt.Errorf("Error in reading CA")
		}

		BeforeEach(func() {
			getExportName = func() (string, error) {
				return "test-vm-export", nil
			}
			getExpandedVM = func() *virtv1.VirtualMachine {
				return &virtv1.VirtualMachine{}
			}
			getDataVolumes = func(vm *virtv1.VirtualMachine) ([]*cdiv1.DataVolume, error) {
				return nil, nil
			}
		})

		AfterEach(func() {
			getExportName = orgGetExportName
			getInternalBasePath = orgGetInternalBasePath
			getExpandedVM = orgGetExpandedVM
			getDataVolumes = orgGetDataVolumes
			getExternalBasePath = orgGetExternalBasePath
		})

		DescribeTable("Secret handler should return error on non GET", func(verb string) {
			req, err := http.NewRequest(verb, "https://test.blah.invalid/vm_def/secret?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusBadRequest))
		},
			Entry("POST", "POST"),
			Entry("PUT", "PUT"),
			Entry("PATCH", "PATCH"),
			Entry("DELETE", "DELETE"),
		)

		It("Should return 500 if export name cannot be read", func() {
			getExportName = func() (string, error) {
				return "", fmt.Errorf("Unable to read export name")
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return 500 if path returns error", func() {
			getInternalBasePath = func() (string, error) {
				return "", fmt.Errorf("Not found")
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getErrorBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return 500 if reading CAConfigMap returns error", func() {
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getBasePath, getErrorCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return 500 if path returns other error", func() {
			getExternalBasePath = func() (string, error) {
				return "", fmt.Errorf("Error")
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar&externalURI=test", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getErrorBasePath, getInternalCAConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return 500 if getExpandedVM returns nil", func() {
			getExpandedVM = func() *virtv1.VirtualMachine {
				return nil
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar&externalURI=test", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return vm definition and associated resources as bytes, yaml", func() {
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			req.Header.Set("Accept", runtime.ContentTypeYAML)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			out := strings.Split(resp.Body.String(), "---\n")
			Expect(out).To(HaveLen(3))
			verifyCmYaml(out[0])
			resVm := &virtv1.VirtualMachine{}
			err = yaml.Unmarshal([]byte(out[1]), resVm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resVm).To(Equal(&virtv1.VirtualMachine{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualMachine",
					APIVersion: virtv1.GroupVersion.String(),
				},
			}))
		})

		It("Should return vm definition and associated resources as bytes, json", func() {
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			list := &v1.List{}
			err = json.Unmarshal(resp.Body.Bytes(), list)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).To(HaveLen(2))
			verifyCmJson(list.Items[0].Raw)
			resVm := &virtv1.VirtualMachine{}
			err = yaml.Unmarshal(list.Items[1].Raw, resVm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resVm).To(Equal(&virtv1.VirtualMachine{
				TypeMeta: metav1.TypeMeta{
					Kind:       "VirtualMachine",
					APIVersion: virtv1.GroupVersion.String(),
				},
			}))
		})

		getTestVm := func() *virtv1.VirtualMachine {
			return &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: testNamespace,
				},
				Spec: virtv1.VirtualMachineSpec{
					DataVolumeTemplates: []virtv1.DataVolumeTemplateSpec{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-dv",
								Namespace: testNamespace,
							},
							Spec: cdiv1.DataVolumeSpec{
								Source: &cdiv1.DataVolumeSource{
									HTTP: &cdiv1.DataVolumeSourceHTTP{
										URL: "",
									},
								},
								Storage: &cdiv1.StorageSpec{
									AccessModes: []v1.PersistentVolumeAccessMode{
										v1.ReadWriteMany,
									},
									Resources: v1.VolumeResourceRequirements{
										Requests: v1.ResourceList{
											v1.ResourceStorage: resource.MustParse("1Gi"),
										},
									},
								},
							},
						},
					},
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "disk0",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{
											Name: "test-dv",
										},
									},
								},
							},
						},
					},
				},
			}
		}

		It("Should override DVTemplates with new source URI, yaml", func() {
			testVm := getTestVm()
			getExpandedVM = func() *virtv1.VirtualMachine {
				return testVm
			}

			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			req.Header.Set("Accept", runtime.ContentTypeYAML)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{
				{
					RawGzURI: "volume0",
				},
			}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			out := strings.Split(resp.Body.String(), "---\n")
			Expect(out).To(HaveLen(3))
			verifyCmYaml(out[0])
			resVm := &virtv1.VirtualMachine{}
			err = yaml.Unmarshal([]byte(out[1]), resVm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resVm.Name).To(Equal(testVm.Name))
			Expect(resVm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(resVm.Spec.DataVolumeTemplates[0].Name).To(Equal("test-dv"))
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source.HTTP).ToNot(BeNil())
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source.HTTP.URL).To(Equal("https://base_path/volume0"))
		})

		It("Should override DVTemplates with new source URI, json", func() {
			testVm := getTestVm()
			getExpandedVM = func() *virtv1.VirtualMachine {
				return testVm
			}

			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{
				{
					RawGzURI: "volume0",
				},
			}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			list := &v1.List{}
			err = json.Unmarshal(resp.Body.Bytes(), list)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).To(HaveLen(2))
			verifyCmJson(list.Items[0].Raw)
			resVm := &virtv1.VirtualMachine{}
			err = yaml.Unmarshal(list.Items[1].Raw, resVm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resVm.Name).To(Equal(testVm.Name))
			Expect(resVm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(resVm.Spec.DataVolumeTemplates[0].Name).To(Equal("test-dv"))
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source).ToNot(BeNil())
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source.HTTP).ToNot(BeNil())
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source.HTTP.URL).To(Equal("https://base_path/volume0"))
		})

		It("Should override datavolumes with new source URI", func() {
			testVm := &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vm",
					Namespace: testNamespace,
				},
				Spec: virtv1.VirtualMachineSpec{
					Template: &virtv1.VirtualMachineInstanceTemplateSpec{
						Spec: virtv1.VirtualMachineInstanceSpec{
							Volumes: []virtv1.Volume{
								{
									Name: "disk0",
									VolumeSource: virtv1.VolumeSource{
										DataVolume: &virtv1.DataVolumeSource{
											Name: "test-dv",
										},
									},
								},
							},
						},
					},
				},
			}
			testDvs := []*cdiv1.DataVolume{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-dv",
						Namespace: testNamespace,
					},
					Spec: cdiv1.DataVolumeSpec{
						Source: &cdiv1.DataVolumeSource{
							HTTP: &cdiv1.DataVolumeSourceHTTP{
								URL: "",
							},
						},
						Storage: &cdiv1.StorageSpec{
							AccessModes: []v1.PersistentVolumeAccessMode{
								v1.ReadWriteMany,
							},
							Resources: v1.VolumeResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceStorage: resource.MustParse("1Gi"),
								},
							},
						},
					},
				},
			}

			getExpandedVM = func() *virtv1.VirtualMachine {
				return testVm
			}
			getDataVolumes = func(vm *virtv1.VirtualMachine) ([]*cdiv1.DataVolume, error) {
				return testDvs, nil
			}

			req, err := http.NewRequest("GET", "https://test.blah.invalid/internal/manifest?x-kubevirt-export-token=bar", nil)
			req.Header.Set("Accept", runtime.ContentTypeYAML)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{
				{
					RawGzURI: "test-dv-volume0",
				},
			}, getBasePath, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			out := strings.Split(resp.Body.String(), "---\n")
			Expect(out).To(HaveLen(4))
			verifyCmYaml(out[0])
			resVm := &virtv1.VirtualMachine{}
			err = yaml.Unmarshal([]byte(out[1]), resVm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resVm.Name).To(Equal(testVm.Name))
			Expect(resVm.Spec.DataVolumeTemplates).To(BeEmpty())
			resDv := &cdiv1.DataVolume{}
			err = yaml.Unmarshal([]byte(out[2]), resDv)
			Expect(err).ToNot(HaveOccurred())
			Expect(resDv.Name).To(Equal("test-dv"))
			Expect(resDv.Spec.Source).ToNot(BeNil())
			Expect(resDv.Spec.Source.HTTP).ToNot(BeNil())
			Expect(resDv.Spec.Source.HTTP.URL).To(Equal("https://base_path/test-dv-volume0"))
		})
	})

	Context("Secret handler", func() {
		verifySecret := func(yamlString string) {
			resSecret := &v1.Secret{}
			err := yaml.Unmarshal([]byte(yamlString), resSecret)
			Expect(err).ToNot(HaveOccurred())
			Expect(resSecret.Name).To(Equal("header-secret-test-export"))
			log.DefaultLogger().Infof("%v", resSecret)
			Expect(resSecret.StringData["token"]).To(Equal("x-kubevirt-export-token:token-secret"))
		}

		tokenGetter := func() (string, error) {
			return "token-secret", nil
		}

		var (
			orgGetExportName = getExportName
		)

		BeforeEach(func() {
			getExportName = func() (string, error) {
				return "test-export", nil
			}
		})

		AfterEach(func() {
			getExportName = orgGetExportName
		})

		DescribeTable("Secret handler should return error on non GET", func(verb string) {
			req, err := http.NewRequest(verb, "https://test.blah.invalid/vm_def/secret?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := secretHandler(tokenGetter)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusBadRequest))
		},
			Entry("POST", "POST"),
			Entry("PUT", "PUT"),
			Entry("PATCH", "PATCH"),
			Entry("DELETE", "DELETE"),
		)

		It("Should return 500 if token cannot be read", func() {
			errorTokenGetter := func() (string, error) {
				return "", fmt.Errorf("Unable to read token")
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def/secret?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := secretHandler(errorTokenGetter)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return 500 if export name cannot be read", func() {
			getExportName = func() (string, error) {
				return "", fmt.Errorf("Unable to read export name")
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def/secret?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := secretHandler(tokenGetter)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusInternalServerError))
		})

		It("Should return secret token as bytes, yaml", func() {
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def/secret?x-kubevirt-export-token=bar", nil)
			req.Header.Set("Accept", runtime.ContentTypeYAML)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := secretHandler(tokenGetter)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			out := strings.Split(resp.Body.String(), "---\n")
			Expect(out).To(HaveLen(2))
			verifySecret(out[0])
		})

		It("Should return secret token as bytes, json", func() {
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def/secret?x-kubevirt-export-token=bar", nil)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := secretHandler(tokenGetter)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			list := &v1.List{}
			err = json.Unmarshal(resp.Body.Bytes(), list)
			Expect(err).ToNot(HaveOccurred())
			Expect(list.Items).To(HaveLen(1))
			verifySecret(string(list.Items[0].Raw))
		})
	})
})
