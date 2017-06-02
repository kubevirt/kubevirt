/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package rest_test

import (
	"flag"
	"net/http"
	"net/http/httptest"

	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	v12 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	rest2 "kubevirt.io/kubevirt/pkg/rest"
	. "kubevirt.io/kubevirt/pkg/virt-api/rest"
	. "kubevirt.io/kubevirt/test"
)

const vmResource = "vms"
const migrationResource = "migrations"
const vmName = "test-vm"

var _ = Describe("Kubeproxy", func() {
	flag.Parse()
	var apiserverMock *ghttp.Server
	var kubeproxy *httptest.Server
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: vmResource}
	migrationGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: migrationResource}
	ctx := context.Background()
	var restClient *rest.RESTClient
	var sourceVM *v1.VM
	var sourceMigration *v1.Migration

	// Work around go-client bug
	expectedVM := v1.NewMinimalVM(vmName)
	expectedVM.TypeMeta.Kind = ""
	expectedVM.TypeMeta.APIVersion = ""

	expectedMigration := v1.NewMinimalMigration("testmigration", vmName)
	expectedMigration.TypeMeta.Kind = ""
	expectedMigration.TypeMeta.APIVersion = ""

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeSuite(func() {
		apiserverMock = ghttp.NewServer()
		flag.Lookup("master").Value.Set(apiserverMock.URL())

		ws, err := GroupVersionProxyBase(ctx, v1.GroupVersion)
		Expect(err).ToNot(HaveOccurred())
		ws, err = GenericResourceProxy(ws, ctx, vmGVR, &v1.VM{}, v1.GroupVersionKind.Kind, &v1.VMList{})
		Expect(err).ToNot(HaveOccurred())
		ws, err = GenericResourceProxy(ws, ctx, migrationGVR, &v1.Migration{}, "Migration", &v1.MigrationList{})
		Expect(err).ToNot(HaveOccurred())
		restful.Add(ws)

	})

	BeforeEach(func() {
		kubeproxy = httptest.NewServer(restful.DefaultContainer)
		var err error
		restClient, err = kubecli.GetRESTClientFromFlags(kubeproxy.URL, "")
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		sourceVM = v1.NewMinimalVM(vmName)
		sourceMigration = v1.NewMinimalMigration("testmigration", vmName)
		apiserverMock.Reset()
	})

	Context("To allow autodiscovery for kubectl", func() {
		It("should proxy /apis/kubevirt.io/v1alpha1/", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/apis/kubevirt.io/v1alpha1/"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			result := restClient.Get().AbsPath("/apis/kubevirt.io/v1alpha1/").Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			body, _ := result.Raw()
			var obj v1.VM
			Expect(json.Unmarshal(body, &obj)).To(Succeed())
			Expect(&obj).To(Equal(expectedVM))
		})
	})

	Context("HTTP Operations on an existing VM in the apiserver", func() {
		It("POST should fail with 409", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, vmBasePath()),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusConflict, struct{}{}),
				),
			)
			result := restClient.Post().Resource(vmResource).Namespace(api.NamespaceDefault).Body(sourceVM).Do()
			Expect(result).To(HaveStatusCode(http.StatusConflict))
		})
		It("Get with invalid VM name should fail with 400", func() {
			result := restClient.Get().Resource(vmResource).Namespace(api.NamespaceDefault).Name("UPerLetterIsInvalid").Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		table.DescribeTable("PUT should succeed", func(contentType string, accept string) {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmPath(sourceVM)),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			result := restClient.Put().Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).
				SetHeader("Content-Type", contentType).SetHeader("Accept", accept).Body(toBytesFromMimeType(sourceVM, contentType)).
				Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(RepresentMimeType(accept))
			Expect(result).To(HaveBodyEqualTo(expectedVM))
		},
			table.Entry("sending JSON and receiving JSON", rest2.MIME_JSON, rest2.MIME_JSON),
			table.Entry("sending JSON and receiving YAML", rest2.MIME_JSON, rest2.MIME_YAML),
			table.Entry("sending YAML and receiving JSON", rest2.MIME_YAML, rest2.MIME_JSON),
			table.Entry("sending YAML and receiving YAML", rest2.MIME_YAML, rest2.MIME_YAML),
		)
		It("DELETE should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, struct{}{}),
				),
			)
			result := restClient.Delete().Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
		})
		table.DescribeTable("GET a VM should succeed", func(accept string) {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			result := restClient.Get().Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).SetHeader("Accept", accept).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(RepresentMimeType(accept))
			Expect(result).To(HaveBodyEqualTo(expectedVM))
		},
			table.Entry("receiving JSON", rest2.MIME_JSON),
			table.Entry("receiving YAML", rest2.MIME_YAML),
		)
		It("GET a VMList should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.VMList{TypeMeta: v12.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VMList"}, Items: []v1.VM{*expectedVM}}),
				),
			)
			result := restClient.Get().Resource(vmResource).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(HaveBodyEqualTo(&v1.VMList{Items: []v1.VM{*expectedVM}}))
		})
		It("Merge Patch should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmPath(sourceVM)),
					returnReceivedBody(http.StatusOK),
				),
			)
			obj, err := restClient.Patch(api.MergePatchType).Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("{\"spec\" : { \"nodeSelector\": {\"test/lala\": \"blub\"}}}"),
			).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(Equal(expectedVM))
			Expect(obj.(*v1.VM).Spec.NodeSelector).Should(HaveKeyWithValue("test/lala", "blub"))
		})
		It("JSON Patch should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmPath(sourceVM)),
					returnReceivedBody(http.StatusOK),
				),
			)
			obj, err := restClient.Patch(api.JSONPatchType).Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("[{ \"op\": \"replace\", \"path\": \"/spec/nodeSelector\", \"value\": {\"test/lala\": \"blub\" }}]"),
			).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(Equal(expectedVM))
			Expect(obj.(*v1.VM).Spec.NodeSelector).Should(HaveKeyWithValue("test/lala", "blub"))
		})
		It("JSON Patch should fail on invalid update", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmPath(sourceVM)),
					returnReceivedBody(http.StatusOK),
				),
			)
			result := restClient.Patch(api.JSONPatchType).Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("[{ \"op\": \"replace\", \"path\": \"/spec/nodeSelector\", \"value\": \"Only an object is allowed here\"}]"),
			).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", 422))
		})
	})

	Context("HTTP Operations on not existing VMs in the apiserver,", func() {
		table.DescribeTable("POST should succeed", func(contentType string, accept string) {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, vmBasePath()),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusCreated, sourceVM),
				),
			)
			result := restClient.Post().Resource(vmResource).Namespace(api.NamespaceDefault).
				SetHeader("Content-Type", contentType).SetHeader("Accept", accept).
				Body(toBytesFromMimeType(sourceVM, contentType)).
				Do()
			Expect(result).To(RepresentMimeType(accept))
			Expect(result).To(HaveStatusCode(http.StatusCreated))
			Expect(result).To(HaveBodyEqualTo(expectedVM))
		},
			table.Entry("sending JSON and receiving JSON", rest2.MIME_JSON, rest2.MIME_JSON),
			table.Entry("sending JSON and receiving YAML", rest2.MIME_JSON, rest2.MIME_YAML),
			table.Entry("sending YAML and receiving JSON", rest2.MIME_YAML, rest2.MIME_JSON),
			table.Entry("sending YAML and receiving YAML", rest2.MIME_YAML, rest2.MIME_YAML),
		)
		It("POST should fail on missing mandatory field with 400", func() {
			sourceVM.Spec = v1.VMSpec{}
			result := restClient.Post().Resource(vmResource).Namespace(api.NamespaceDefault).Body(sourceVM).Do()
			Expect(result).To(HaveStatusCode(http.StatusBadRequest))
		})
		It("PUT should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmPath(sourceVM)),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Put().Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(sourceVM).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		It("DELETE should fail", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Delete().Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		It("GET should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Get().Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		It("GET a VMList should return an empty list", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.VMList{TypeMeta: v12.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VMList"}}),
				),
			)
			obj, err := restClient.Get().Resource(vmResource).Namespace(api.NamespaceDefault).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(&v1.VMList{}))
		})
		It("Merge Patch should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmPath(sourceVM)),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, sourceVM),
				),
			)
			result := restClient.Patch(api.MergePatchType).Resource(vmResource).Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("{\"spec\" : { \"nodeSelector\": {\"test/lala\": \"blub\"}}}"),
			).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
	})

	Context("HTTP Operation on a given migration in the apiserver", func() {
		It("DELETE should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, migrationPath(sourceMigration)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, struct{}{}),
				),
			)
			result := restClient.Delete().Resource(migrationResource).Name(sourceMigration.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
		})
		It("GET should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, migrationPath(sourceMigration)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceMigration),
				),
			)
			result := restClient.Get().Resource(migrationResource).Name(sourceMigration.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(HaveBodyEqualTo(expectedMigration))
		})
		It("PUT should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, migrationPath(sourceMigration)),
					ghttp.VerifyJSONRepresenting(sourceMigration),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceMigration),
				),
			)
			result := restClient.Put().Resource(migrationResource).Name(sourceMigration.GetObjectMeta().GetName()).Body(sourceMigration).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(HaveBodyEqualTo(expectedMigration))
		})
		It("GET a MigrationList should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, migrationBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.MigrationList{TypeMeta: v12.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "MigrationList"}, Items: []v1.Migration{*expectedMigration}}),
				),
			)
			result := restClient.Get().Resource(migrationResource).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(HaveBodyEqualTo(&v1.MigrationList{Items: []v1.Migration{*expectedMigration}}))
		})
	})
	Context("HTTP Operation on a non existent migration in the apiserver", func() {
		It("POST should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, migrationBasePath()),
					ghttp.VerifyJSONRepresenting(sourceMigration),
					ghttp.RespondWithJSONEncoded(http.StatusCreated, sourceMigration),
				),
			)
			result := restClient.Post().Resource(migrationResource).Body(sourceMigration).Namespace(api.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusCreated))
			Expect(result).To(HaveBodyEqualTo(expectedMigration))
		})
	})

	AfterEach(func() {
		kubeproxy.Close()
	})

	AfterSuite(func() {
		apiserverMock.Close()
	})
})

func vmBasePath() string {
	return "/apis/kubevirt.io/v1alpha1/namespaces/default/vms"
}

func migrationBasePath() string {
	return "/apis/kubevirt.io/v1alpha1/namespaces/default/migrations"
}

func vmPath(vm *v1.VM) string {
	return vmBasePath() + "/" + vm.GetObjectMeta().GetName()
}

func migrationPath(migration *v1.Migration) string {
	return migrationBasePath() + "/" + migration.GetObjectMeta().GetName()
}

func returnReceivedBody(statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		data, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		Expect(err).ToNot(HaveOccurred())
		w.Write(data)
	}
}

func toBytesFromMimeType(obj interface{}, mimeType string) []byte {
	switch mimeType {

	case rest2.MIME_JSON:
		data, err := json.Marshal(obj)
		Expect(err).ToNot(HaveOccurred())
		return data
	case rest2.MIME_YAML:
		data, err := yaml.Marshal(obj)
		Expect(err).ToNot(HaveOccurred())
		return data
	default:
		panic(fmt.Errorf("Mime Type %s is not supported", mimeType))
	}
}
