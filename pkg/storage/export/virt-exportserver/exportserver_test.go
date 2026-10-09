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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/storage/nbdclient"

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
	s, err := NewExportServer(config)
	Expect(err).ToNot(HaveOccurred())
	return s.(*exportServer)
}

type fakeNBDSource struct {
	servingErr error
	mapErr     error
	extents    []nbdclient.Extent
	data       []byte

	exportName string
	bitmapName string
	offset     uint64
	length     uint64
}

func (f *fakeNBDSource) Serving(time.Duration) error { return f.servingErr }

func (f *fakeNBDSource) Map(_ context.Context, exportName, bitmapName string, offset, length uint64) iter.Seq2[nbdclient.Extent, error] {
	f.exportName, f.bitmapName, f.offset, f.length = exportName, bitmapName, offset, length
	return func(yield func(nbdclient.Extent, error) bool) {
		if f.mapErr != nil {
			yield(nbdclient.Extent{}, f.mapErr)
			return
		}
		for _, extent := range f.extents {
			if !yield(extent, nil) {
				return
			}
		}
	}
}

func (f *fakeNBDSource) Read(_ context.Context, exportName string, offset, length uint64, w io.Writer) error {
	f.exportName, f.offset, f.length = exportName, offset, length
	_, err := w.Write(f.data)
	return err
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

		It("Should keep host port when volume URI is absolute", func() {
			testVm := getTestVm()
			getExpandedVM = func() *virtv1.VirtualMachine {
				return testVm
			}

			getInternalHost := func() (string, error) {
				return fmt.Sprintf("virt-export-test.default.svc:%d", export.ExportServerPort), nil
			}
			req, err := http.NewRequest("GET", "https://test.blah.invalid/vm_def?x-kubevirt-export-token=bar", nil)
			req.Header.Set("Accept", runtime.ContentTypeYAML)
			resp := httptest.NewRecorder()
			Expect(err).ToNot(HaveOccurred())
			handler := vmHandler([]export.VolumeInfo{
				{
					RawGzURI: "/volumes/test-dv/disk.img.gz",
				},
			}, getInternalHost, getCaConfigMap)
			handler.ServeHTTP(resp, req)
			Expect(resp.Code).To(BeEquivalentTo(http.StatusOK))
			out := strings.Split(resp.Body.String(), "---\n")
			Expect(out).To(HaveLen(3))
			resVm := &virtv1.VirtualMachine{}
			err = yaml.Unmarshal([]byte(out[1]), resVm)
			Expect(err).ToNot(HaveOccurred())
			Expect(resVm.Spec.DataVolumeTemplates).To(HaveLen(1))
			Expect(resVm.Spec.DataVolumeTemplates[0].Spec.Source.HTTP.URL).To(Equal(fmt.Sprintf("https://virt-export-test.default.svc:%d/volumes/test-dv/disk.img.gz", export.ExportServerPort)))
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

	Context("backupMapHandler", func() {
		var server *exportServer

		BeforeEach(func() {
			server = &exportServer{nbdSource: &fakeNBDSource{}}
		})

		DescribeTable("should return error on non GET", func(verb string) {
			req := httptest.NewRequest(verb, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(BeEquivalentTo(http.StatusMethodNotAllowed))
		},
			Entry("POST", http.MethodPost),
			Entry("PUT", http.MethodPut),
			Entry("PATCH", http.MethodPatch),
			Entry("DELETE", http.MethodDelete),
		)

		It("should return 503 when the NBD server is not serving", func() {
			server.nbdSource = &fakeNBDSource{servingErr: errors.New("connection refused")}

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("should return a JSON map response", func() {
			server.nbdSource = &fakeNBDSource{extents: []nbdclient.Extent{
				{Offset: 0, Length: 512, Flags: 0, Description: "data"},
				{Offset: 512, Length: 512, Flags: 1, Description: "hole"},
			}}

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp export.ExportMapResponse
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Extents).To(Equal([]export.ExportMapExtent{
				{Offset: 0, Length: 512, Type: 0, Description: "data"},
				{Offset: 512, Length: 512, Type: 1, Description: "hole"},
			}))
			Expect(resp.NextOffset).To(BeNil())
		})

		It("should set NextOffset when page_size is exceeded", func() {
			server.nbdSource = &fakeNBDSource{extents: []nbdclient.Extent{
				{Offset: 0, Length: 512},
				{Offset: 512, Length: 512},
				{Offset: 1024, Length: 512},
			}}

			req := httptest.NewRequest(http.MethodGet, "/backup/map?page_size=2", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp export.ExportMapResponse
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Extents).To(HaveLen(2))
			Expect(resp.NextOffset).To(HaveValue(Equal(uint64(1024))))
		})

		DescribeTable("should return 400 for invalid query parameters",
			func(query string) {
				req := httptest.NewRequest(http.MethodGet, "/backup/map?"+query, nil)
				rec := httptest.NewRecorder()
				server.backupMapHandler("disk0").ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusBadRequest))
			},
			Entry("non-numeric offset", "offset=notanumber"),
			Entry("non-numeric length", "length=notanumber"),
			Entry("zero page_size", "page_size=0"),
			Entry("negative page_size", "page_size=-5"),
		)

		It("should return 500 when the map fails", func() {
			server.nbdSource = &fakeNBDSource{mapErr: errors.New("map failed")}

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})

		DescribeTable("should select the bitmap from the backup type", func(backupType, expectedBitmap string) {
			source := &fakeNBDSource{}
			server.nbdSource = source
			server.BackupType = backupType
			server.BackupCheckpoint = "checkpoint-name"

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			server.backupMapHandler("disk0").ServeHTTP(httptest.NewRecorder(), req)

			Expect(source.exportName).To(Equal("disk0"))
			Expect(source.bitmapName).To(Equal(expectedBitmap))
		},
			Entry("incremental passes the checkpoint", string(backupv1.Incremental), "checkpoint-name"),
			Entry("full passes no bitmap", "Full", ""),
		)
	})

	Context("backupDataHandler", func() {
		var server *exportServer

		BeforeEach(func() {
			server = &exportServer{nbdSource: &fakeNBDSource{}}
		})

		DescribeTable("should return error on non GET", func(verb string) {
			req := httptest.NewRequest(verb, "/backup/data", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(BeEquivalentTo(http.StatusMethodNotAllowed))
		},
			Entry("POST", http.MethodPost),
			Entry("PUT", http.MethodPut),
		)

		It("should return 503 when the NBD server is not serving", func() {
			server.nbdSource = &fakeNBDSource{servingErr: errors.New("connection refused")}

			req := httptest.NewRequest(http.MethodGet, "/backup/data", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("should stream the export with the right content type", func() {
			server.nbdSource = &fakeNBDSource{data: []byte("disk contents")}

			req := httptest.NewRequest(http.MethodGet, "/backup/data", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(Equal("application/octet-stream"))
			Expect(rec.Body.String()).To(Equal("disk contents"))
		})

		It("should pass offset and length through", func() {
			source := &fakeNBDSource{}
			server.nbdSource = source

			req := httptest.NewRequest(http.MethodGet, "/backup/data?offset=512&length=1024", nil)
			server.backupDataHandler("disk0").ServeHTTP(httptest.NewRecorder(), req)

			Expect(source.exportName).To(Equal("disk0"))
			Expect(source.offset).To(Equal(uint64(512)))
			Expect(source.length).To(Equal(uint64(1024)))
		})

		DescribeTable("should return 400 for invalid query parameters", func(query string) {
			req := httptest.NewRequest(http.MethodGet, "/backup/data?"+query, nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		},
			Entry("non-numeric offset", "offset=notanumber"),
			Entry("non-numeric length", "length=notanumber"),
		)
	})

	Context("collectMapPage", func() {
		extentsOf := func(extents ...nbdclient.Extent) iter.Seq2[nbdclient.Extent, error] {
			return func(yield func(nbdclient.Extent, error) bool) {
				for _, extent := range extents {
					if !yield(extent, nil) {
						return
					}
				}
			}
		}

		It("should collect every extent when the page is not filled", func() {
			page, next, err := collectMapPage(extentsOf(
				nbdclient.Extent{Offset: 0, Length: 512, Flags: 3, Description: "hole,zero"},
			), 10)

			Expect(err).ToNot(HaveOccurred())
			Expect(next).To(BeNil())
			Expect(page).To(Equal([]export.ExportMapExtent{
				{Offset: 0, Length: 512, Type: 3, Description: "hole,zero"},
			}))
		})

		It("should stop at the page size and report the next offset", func() {
			page, next, err := collectMapPage(extentsOf(
				nbdclient.Extent{Offset: 0, Length: 512},
				nbdclient.Extent{Offset: 512, Length: 512},
			), 1)

			Expect(err).ToNot(HaveOccurred())
			Expect(page).To(HaveLen(1))
			Expect(next).To(HaveValue(Equal(uint64(512))))
		})

		It("should return the map error", func() {
			failing := func(yield func(nbdclient.Extent, error) bool) {
				yield(nbdclient.Extent{}, errors.New("map failed"))
			}

			_, _, err := collectMapPage(failing, 10)

			Expect(err).To(MatchError("map failed"))
		})
	})

	Context("dirHandler symlink safety", func() {
		It("should serve files within the mount root", func() {
			root := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0644)).To(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/volumes/pvc/dir/hello.txt", nil)
			rec := httptest.NewRecorder()
			dirHandler("/volumes/pvc/dir/", root).ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("hello"))
		})

		It("should block symlinks pointing outside the mount root", func() {
			root := GinkgoT().TempDir()
			outside := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(outside, "secret"), []byte("ESCAPED"), 0600)).To(Succeed())
			Expect(os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "escape-link"))).To(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/volumes/pvc/dir/escape-link", nil)
			rec := httptest.NewRecorder()
			dirHandler("/volumes/pvc/dir/", root).ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})

		It("should allow symlinks that stay within the mount root", func() {
			root := GinkgoT().TempDir()
			Expect(os.MkdirAll(filepath.Join(root, "subdir"), 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(root, "subdir", "data.txt"), []byte("internal"), 0644)).To(Succeed())
			Expect(os.Symlink("subdir/data.txt", filepath.Join(root, "internal-link"))).To(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/volumes/pvc/dir/internal-link", nil)
			rec := httptest.NewRecorder()
			dirHandler("/volumes/pvc/dir/", root).ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(Equal("internal"))
		})

		It("should block path traversal via dot-dot segments", func() {
			root := GinkgoT().TempDir()
			outside := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(outside, "secret"), []byte("ESCAPED"), 0600)).To(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/volumes/pvc/dir/../../"+filepath.Base(outside)+"/secret", nil)
			rec := httptest.NewRecorder()
			dirHandler("/volumes/pvc/dir/", root).ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusNotFound))
			Expect(rec.Body.String()).ToNot(ContainSubstring("ESCAPED"))
		})

		It("should block symlink to absolute path outside root", func() {
			root := GinkgoT().TempDir()
			Expect(os.Symlink("/etc/hostname", filepath.Join(root, "host-link"))).To(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/volumes/pvc/dir/host-link", nil)
			rec := httptest.NewRecorder()
			dirHandler("/volumes/pvc/dir/", root).ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusNotFound))
		})

		It("should serve the root directory listing", func() {
			root := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0644)).To(Succeed())

			req := httptest.NewRequest(http.MethodGet, "/volumes/pvc/dir/", nil)
			rec := httptest.NewRecorder()
			dirHandler("/volumes/pvc/dir/", root).ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.String()).To(ContainSubstring("file.txt"))
		})
	})

	Context("buildServer", func() {
		var (
			server      *exportServer
			baseHandler *http.ServeMux
		)

		BeforeEach(func() {
			baseHandler = http.NewServeMux()
			server = &exportServer{
				handler: baseHandler,
				ExportServerConfig: ExportServerConfig{
					ListenAddr:    fmt.Sprintf(":%d", export.ExportServerPort),
					TLSMinVersion: tls.VersionTLS12,
				},
			}
		})

		It("should return a standard TLS config and the unmodified base handler", func() {
			srv := server.buildServer(context.Background())

			Expect(srv.Addr).To(Equal(fmt.Sprintf(":%d", export.ExportServerPort)))
			Expect(srv.Handler).To(BeIdenticalTo(baseHandler))
			Expect(srv.TLSConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
			Expect(srv.TLSConfig.ClientCAs).To(BeNil())
			Expect(srv.TLSConfig.ClientAuth).To(Equal(tls.NoClientCert))
		})
	})

	Context("readiness with backup paths", func() {
		It("should return 503 when backup paths exist but the NBD server is not serving", func() {
			es := newTestServer("token")
			es.Paths = &export.ServerPaths{
				Backups: []export.BackupInfo{
					{Path: "vol1", DataURI: "/exports/vol1/data", MapURI: "/exports/vol1/map"},
				},
			}
			es.nbdSource = &fakeNBDSource{servingErr: errors.New("connection refused")}
			es.initHandler()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, export.ReadinessPath, http.NoBody)
			es.handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("should return 200 when backup paths exist and the NBD server is serving", func() {
			es := newTestServer("token")
			es.Paths = &export.ServerPaths{
				Backups: []export.BackupInfo{
					{Path: "vol1", DataURI: "/exports/vol1/data", MapURI: "/exports/vol1/map"},
				},
			}
			es.nbdSource = &fakeNBDSource{}
			es.initHandler()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, export.ReadinessPath, http.NoBody)
			es.handler.ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})
})
