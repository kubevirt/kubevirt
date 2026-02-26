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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	nbdv1 "kubevirt.io/kubevirt/pkg/storage/cbt/nbd/v1"

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

func newBackupServer(caCert []byte, uid string) *exportServer {
	return &exportServer{
		ExportServerConfig: ExportServerConfig{
			BackupCACert: caCert,
			BackupUID:    uid,
		},
	}
}

func generateKeypair() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).ToNot(HaveOccurred())
	return priv, &priv.PublicKey
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

	Context("backupMapHandler", func() {
		var (
			ctrl   *gomock.Controller
			server *exportServer
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			server = &exportServer{
				ExportServerConfig: ExportServerConfig{},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
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

		It("should return 503 when no NBD client is connected", func() {
			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("should return a JSON map response for a healthy client", func() {
			mapStream := nbdv1.NewMockNBD_MapClient(ctrl)
			gomock.InOrder(
				mapStream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 0, Length: 512, Flags: 0, Description: "data"}},
				}, nil),
				mapStream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 512, Length: 512, Flags: 1, Description: "hole"}},
				}, nil),
				mapStream.EXPECT().Recv().Return(nil, io.EOF),
			)
			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().
				Map(gomock.Any(), &nbdv1.MapRequest{ExportName: "disk0", Offset: 0, Length: 1024}).
				Return(mapStream, nil)
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/map?offset=0&length=1024", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp ExportMapResponse
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Extents).To(HaveLen(2))
			Expect(resp.Extents[0]).To(Equal(ExportMapExtent{Offset: 0, Length: 512, Type: 0, Description: "data"}))
			Expect(resp.Extents[1]).To(Equal(ExportMapExtent{Offset: 512, Length: 512, Type: 1, Description: "hole"}))
			Expect(resp.NextOffset).To(BeNil())
		})

		It("should set NextOffset when page_size is exceeded", func() {
			mapStream := nbdv1.NewMockNBD_MapClient(ctrl)
			gomock.InOrder(
				mapStream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 0, Length: 512}},
				}, nil),
				mapStream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 512, Length: 512}},
				}, nil),
				mapStream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 1024, Length: 512}},
				}, nil),
			)
			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().Map(gomock.Any(), gomock.Any()).Return(mapStream, nil)
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/map?page_size=2", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			var resp ExportMapResponse
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Extents).To(HaveLen(2))
			Expect(resp.NextOffset).ToNot(BeNil())
			Expect(*resp.NextOffset).To(Equal(uint64(1024)))
		})

		DescribeTable("should return 400 for invalid query parameters",
			func(query string) {
				server.nbdClient = nbdv1.NewMockNBDClient(ctrl)
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

		It("should return 500 when the gRPC Map call fails", func() {
			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().Map(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("grpc error"))
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should return 500 when the map stream errors mid-receive", func() {
			mapStream := nbdv1.NewMockNBD_MapClient(ctrl)
			mapStream.EXPECT().Recv().Return(nil, fmt.Errorf("stream error"))

			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().Map(gomock.Any(), gomock.Any()).Return(mapStream, nil)
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			rec := httptest.NewRecorder()
			server.backupMapHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should pass the bitmap name for incremental backups", func() {
			mapStream := nbdv1.NewMockNBD_MapClient(ctrl)
			mapStream.EXPECT().Recv().Return(nil, io.EOF)

			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().
				Map(gomock.Any(), &nbdv1.MapRequest{ExportName: "disk0", BitmapName: "checkpoint-name"}).
				Return(mapStream, nil)
			server.nbdClient = nbdClient
			server.ExportServerConfig.BackupType = string(backupv1.Incremental)
			server.ExportServerConfig.BackupCheckpoint = "checkpoint-name"

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			server.backupMapHandler("disk0").ServeHTTP(httptest.NewRecorder(), req)
		})

		It("should omit the bitmap name for full backups", func() {
			mapStream := nbdv1.NewMockNBD_MapClient(ctrl)
			mapStream.EXPECT().Recv().Return(nil, io.EOF)

			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().
				Map(gomock.Any(), &nbdv1.MapRequest{ExportName: "disk0"}).
				Return(mapStream, nil)
			server.nbdClient = nbdClient
			server.ExportServerConfig.BackupType = "Full"
			server.ExportServerConfig.BackupCheckpoint = "checkpoint-name"

			req := httptest.NewRequest(http.MethodGet, "/backup/map", nil)
			server.backupMapHandler("disk0").ServeHTTP(httptest.NewRecorder(), req)
		})
	})

	Context("backupDataHandler", func() {
		var (
			ctrl   *gomock.Controller
			server *exportServer
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			server = &exportServer{
				ExportServerConfig: ExportServerConfig{},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
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

		It("should return 503 when no NBD client is connected", func() {
			req := httptest.NewRequest(http.MethodGet, "/backup/data", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("should stream all chunks with correct content-type", func() {
			readStream := nbdv1.NewMockNBD_ReadClient(ctrl)
			gomock.InOrder(
				readStream.EXPECT().Recv().Return(&nbdv1.DataChunk{Data: []byte("first-chunk-")}, nil),
				readStream.EXPECT().Recv().Return(&nbdv1.DataChunk{Data: []byte("second-chunk")}, nil),
				readStream.EXPECT().Recv().Return(nil, io.EOF),
			)
			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().
				Read(gomock.Any(), &nbdv1.ReadRequest{ExportName: "disk0", Offset: 0, Length: 24}).
				Return(readStream, nil)
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/data?offset=0&length=24", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Header().Get("Content-Type")).To(Equal("application/octet-stream"))
			Expect(rec.Body.String()).To(Equal("first-chunk-second-chunk"))
		})

		It("should return 200 with an empty body for a zero-length stream", func() {
			readStream := nbdv1.NewMockNBD_ReadClient(ctrl)
			readStream.EXPECT().Recv().Return(nil, io.EOF)

			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().Read(gomock.Any(), gomock.Any()).Return(readStream, nil)
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/data", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Body.Bytes()).To(BeEmpty())
		})

		DescribeTable("should return 400 for invalid query parameters",
			func(query string) {
				server.nbdClient = nbdv1.NewMockNBDClient(ctrl)
				req := httptest.NewRequest(http.MethodGet, "/backup/data?"+query, nil)
				rec := httptest.NewRecorder()
				server.backupDataHandler("disk0").ServeHTTP(rec, req)
				Expect(rec.Code).To(Equal(http.StatusBadRequest))
			},
			Entry("non-numeric offset", "offset=bad"),
			Entry("non-numeric length", "length=bad"),
		)

		It("should return 500 when the gRPC Read call fails", func() {
			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().Read(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("read error"))
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/data", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should pass offset and length to the gRPC Read call", func() {
			readStream := nbdv1.NewMockNBD_ReadClient(ctrl)
			readStream.EXPECT().Recv().Return(nil, io.EOF)

			nbdClient := nbdv1.NewMockNBDClient(ctrl)
			nbdClient.EXPECT().
				Read(gomock.Any(), &nbdv1.ReadRequest{ExportName: "disk0", Offset: 4096, Length: 8192}).
				Return(readStream, nil)
			server.nbdClient = nbdClient

			req := httptest.NewRequest(http.MethodGet, "/backup/data?offset=4096&length=8192", nil)
			rec := httptest.NewRecorder()
			server.backupDataHandler("disk0").ServeHTTP(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
		})
	})

	Context("collectMapPage", func() {
		var ctrl *gomock.Controller

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should collect all extents when count is below page size", func() {
			stream := nbdv1.NewMockNBD_MapClient(ctrl)
			gomock.InOrder(
				stream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 0, Length: 100}},
				}, nil),
				stream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 100, Length: 200}},
				}, nil),
				stream.EXPECT().Recv().Return(nil, io.EOF),
			)

			extents, nextOff, err := collectMapPage(stream, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(extents).To(HaveLen(2))
			Expect(nextOff).To(BeNil())
		})

		It("should stop exactly at page size and returns the next extent's offset", func() {
			stream := nbdv1.NewMockNBD_MapClient(ctrl)
			gomock.InOrder(
				stream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 0, Length: 100}},
				}, nil),
				stream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 100, Length: 200}},
				}, nil),
				stream.EXPECT().Recv().Return(&nbdv1.MapResponse{
					Extents: []*nbdv1.Extent{{Offset: 300, Length: 400}},
				}, nil),
			)

			extents, nextOff, err := collectMapPage(stream, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(extents).To(HaveLen(2))
			Expect(nextOff).ToNot(BeNil())
			Expect(*nextOff).To(Equal(uint64(300)))
		})

		It("should return an error when the stream errors", func() {
			stream := nbdv1.NewMockNBD_MapClient(ctrl)
			stream.EXPECT().Recv().Return(nil, fmt.Errorf("stream broke"))

			_, _, err := collectMapPage(stream, 10)
			Expect(err).To(MatchError("stream broke"))
		})

		It("should return an empty slice and nil next offset on immediate EOF", func() {
			stream := nbdv1.NewMockNBD_MapClient(ctrl)
			stream.EXPECT().Recv().Return(nil, io.EOF)

			extents, nextOff, err := collectMapPage(stream, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(extents).To(BeEmpty())
			Expect(nextOff).To(BeNil())
		})
	})

	Context("handleTunnel", func() {
		var (
			ctrl   *gomock.Controller
			server *exportServer
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			server = &exportServer{
				ExportServerConfig: ExportServerConfig{
					BackupUID: "test-uid",
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should reject requests without a client certificate", func() {
			req := httptest.NewRequest(http.MethodConnect, "host.example.com:443", nil)
			rec := httptest.NewRecorder()

			server.handleTunnel(rec, req)

			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Body.String()).To(ContainSubstring("mTLS required"))
		})

		It("should reject requests with an invalid client CN", func() {
			req := httptest.NewRequest(http.MethodConnect, "host.example.com:443", nil)
			req.TLS = &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{Subject: pkix.Name{CommonName: "kubevirt.io:system:client:wrong-uid"}},
				},
			}
			rec := httptest.NewRecorder()

			server.handleTunnel(rec, req)

			Expect(rec.Code).To(Equal(http.StatusForbidden))
			Expect(rec.Body.String()).To(ContainSubstring("Forbidden"))
		})

		It("should reject if a tunnel is already active", func() {
			server.nbdClient = nbdv1.NewMockNBDClient(ctrl)

			req := httptest.NewRequest(http.MethodConnect, "host.example.com:443", nil)
			req.TLS = &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{Subject: pkix.Name{CommonName: "kubevirt.io:system:client:test-uid"}},
				},
			}
			rec := httptest.NewRecorder()

			server.handleTunnel(rec, req)

			Expect(rec.Code).To(Equal(http.StatusConflict))
			Expect(rec.Body.String()).To(ContainSubstring("Conflict"))
		})

		It("should accept valid connections, flush headers, and clean up on context cancel", func() {
			ctx, cancel := context.WithCancel(context.Background())
			req := httptest.NewRequest(http.MethodConnect, "host.example.com:443", http.NoBody).WithContext(ctx)
			req.TLS = &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					{Subject: pkix.Name{CommonName: "kubevirt.io:system:client:test-uid"}},
				},
			}
			rec := httptest.NewRecorder()
			cancel()

			server.handleTunnel(rec, req)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(rec.Flushed).To(BeTrue())

			server.nbdMu.Lock()
			client := server.nbdClient
			server.nbdMu.Unlock()
			Expect(client).To(BeNil())
		})
	})

	Context("h2ServerConn", func() {
		It("should bridge Read to the request body and Write to the response writer", func() {
			readData := []byte("test")
			body := io.NopCloser(bytes.NewReader(readData))
			rec := httptest.NewRecorder()

			canceled := false
			cancelFunc := func() { canceled = true }

			conn := newH2ServerConn(body, rec, cancelFunc)

			buf := make([]byte, 4)
			n, err := conn.Read(buf)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(readData)))
			Expect(buf[:n]).To(Equal(readData))

			writeData := []byte("test")
			n, err = conn.Write(writeData)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).To(Equal(len(writeData)))
			Expect(rec.Body.Bytes()).To(Equal(writeData))

			err = conn.Close()
			Expect(err).ToNot(HaveOccurred())
			Expect(canceled).To(BeTrue())
		})
	})
})
