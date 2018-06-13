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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	rest2 "kubevirt.io/kubevirt/pkg/rest"
	. "kubevirt.io/kubevirt/pkg/testutils"
	. "kubevirt.io/kubevirt/pkg/virt-api/rest"
)

const vmiResource = "virtualmachineinstances"
const vmiName = "test-vmi"

var _ = Describe("Kubeproxy", func() {
	flag.Parse()
	var apiserverMock *ghttp.Server
	var kubeproxy *httptest.Server
	vmiGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: vmiResource}
	ctx := context.Background()
	var restClient *rest.RESTClient
	var sourceVMI *v1.VirtualMachineInstance

	// Work around go-client bug
	expectedVMI := v1.NewMinimalVMI(vmiName)
	expectedVMI.TypeMeta.Kind = ""
	expectedVMI.TypeMeta.APIVersion = ""

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeSuite(func() {
		apiserverMock = ghttp.NewServer()
		flag.Lookup("master").Value.Set(apiserverMock.URL())

		ws, err := GroupVersionProxyBase(ctx, v1.GroupVersion)
		Expect(err).ToNot(HaveOccurred())
		ws, err = GenericResourceProxy(ws, ctx, vmiGVR, &v1.VirtualMachineInstance{}, v1.VirtualMachineInstanceGroupVersionKind.Kind, &v1.VirtualMachineInstanceList{})
		Expect(err).ToNot(HaveOccurred())
		restful.Add(ws)
	})

	BeforeEach(func() {
		kubeproxy = httptest.NewServer(restful.DefaultContainer)
		var err error
		virtClient, err := kubecli.GetKubevirtClientFromFlags(kubeproxy.URL, "")
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		restClient = virtClient.RestClient()
		sourceVMI = v1.NewMinimalVMI(vmiName)
		apiserverMock.Reset()
	})

	Context("To allow autodiscovery for kubectl", func() {
		It("should proxy /apis/kubevirt.io/v1alpha2/", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, "/apis/kubevirt.io/v1alpha2/"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVMI),
				),
			)
			result := restClient.Get().AbsPath("/apis/kubevirt.io/v1alpha2/").Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			body, _ := result.Raw()
			var obj v1.VirtualMachineInstance
			Expect(json.Unmarshal(body, &obj)).To(Succeed())
			Expect(&obj).To(Equal(expectedVMI))
		})
	})

	Context("HTTP Operations on an existing VirtualMachineInstance in the apiserver", func() {
		It("POST should fail with 409", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, vmiBasePath()),
					ghttp.VerifyJSONRepresenting(sourceVMI),
					ghttp.RespondWithJSONEncoded(http.StatusConflict, struct{}{}),
				),
			)
			result := restClient.Post().Resource(vmiResource).Namespace(k8sv1.NamespaceDefault).Body(sourceVMI).Do()
			Expect(result).To(HaveStatusCode(http.StatusConflict))
		})
		It("Get with invalid VirtualMachineInstance name should fail with 400", func() {
			result := restClient.Get().Resource(vmiResource).Namespace(k8sv1.NamespaceDefault).Name("UPerLetterIsInvalid").Do()
			Expect(result).To(HaveStatusCode(http.StatusBadRequest))
		})
		table.DescribeTable("PUT should succeed", func(contentType string, accept string) {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmiPath(sourceVMI)),
					ghttp.VerifyJSONRepresenting(sourceVMI),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVMI),
				),
			)
			result := restClient.Put().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).
				SetHeader("Content-Type", contentType).SetHeader("Accept", accept).Body(toBytesFromMimeType(sourceVMI, contentType)).
				Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(RepresentMimeType(accept))
			Expect(result).To(HaveBodyEqualTo(expectedVMI))
		},
			table.Entry("sending JSON and receiving JSON", rest2.MIME_JSON, rest2.MIME_JSON),
			table.Entry("sending JSON and receiving YAML", rest2.MIME_JSON, rest2.MIME_YAML),
			table.Entry("sending YAML and receiving JSON", rest2.MIME_YAML, rest2.MIME_JSON),
			table.Entry("sending YAML and receiving YAML", rest2.MIME_YAML, rest2.MIME_YAML),
		)
		It("DELETE should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, struct{}{}),
				),
			)
			result := restClient.Delete().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
		})
		It("DELETE with delete options should succeed", func() {
			policy := v12.DeletePropagationOrphan
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, struct{}{}),
					ghttp.VerifyJSONRepresenting(&v12.DeleteOptions{
						TypeMeta: v12.TypeMeta{
							Kind:       "DeleteOptions",
							APIVersion: "kubevirt.io/v1alpha2"},
						PropagationPolicy: &policy}),
				),
			)
			result := restClient.Delete().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Body(&v12.DeleteOptions{PropagationPolicy: &policy}).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
		})
		table.DescribeTable("GET a VirtualMachineInstance should succeed", func(accept string) {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVMI),
				),
			)
			result := restClient.Get().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).SetHeader("Accept", accept).Namespace(k8sv1.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(RepresentMimeType(accept))
			Expect(result).To(HaveBodyEqualTo(expectedVMI))
		},
			table.Entry("receiving JSON", rest2.MIME_JSON),
			table.Entry("receiving YAML", rest2.MIME_YAML),
		)
		It("GET a VirtualMachineInstanceList should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.VirtualMachineInstanceList{TypeMeta: v12.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceList"}, Items: []v1.VirtualMachineInstance{*expectedVMI}}),
				),
			)
			result := restClient.Get().Resource(vmiResource).Namespace(k8sv1.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusOK))
			Expect(result).To(HaveBodyEqualTo(&v1.VirtualMachineInstanceList{Items: []v1.VirtualMachineInstance{*expectedVMI}}))
		})
		It("Merge Patch should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVMI),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmiPath(sourceVMI)),
					returnReceivedBody(http.StatusOK),
				),
			)
			obj, err := restClient.Patch(types.MergePatchType).Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Body(
				[]byte("{\"spec\" : { \"nodeSelector\": {\"test/lala\": \"blub\"}}}"),
			).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(Equal(expectedVMI))
			Expect(obj.(*v1.VirtualMachineInstance).Spec.NodeSelector).Should(HaveKeyWithValue("test/lala", "blub"))
		})
		It("JSON Patch should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVMI),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmiPath(sourceVMI)),
					returnReceivedBody(http.StatusOK),
				),
			)
			obj, err := restClient.Patch(types.JSONPatchType).Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Body(
				[]byte("[{ \"op\": \"replace\", \"path\": \"/spec/nodeSelector\", \"value\": {\"test/lala\": \"blub\" }}]"),
			).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(Equal(expectedVMI))
			Expect(obj.(*v1.VirtualMachineInstance).Spec.NodeSelector).Should(HaveKeyWithValue("test/lala", "blub"))
		})
		It("JSON Patch should fail on invalid update", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVMI),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmiPath(sourceVMI)),
					returnReceivedBody(http.StatusOK),
				),
			)
			result := restClient.Patch(types.JSONPatchType).Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Body(
				[]byte("[{ \"op\": \"replace\", \"path\": \"/spec/nodeSelector\", \"value\": \"Only an object is allowed here\"}]"),
			).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", 422))
		})
	})

	Context("HTTP Operations on not existing VMIs in the apiserver,", func() {
		table.DescribeTable("POST should succeed", func(contentType string, accept string) {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, vmiBasePath()),
					ghttp.VerifyJSONRepresenting(sourceVMI),
					ghttp.RespondWithJSONEncoded(http.StatusCreated, sourceVMI),
				),
			)
			result := restClient.Post().Resource(vmiResource).Namespace(k8sv1.NamespaceDefault).
				SetHeader("Content-Type", contentType).SetHeader("Accept", accept).
				Body(toBytesFromMimeType(sourceVMI, contentType)).
				Do()
			Expect(result).To(RepresentMimeType(accept))
			Expect(result).To(HaveStatusCode(http.StatusCreated))
			Expect(result).To(HaveBodyEqualTo(expectedVMI))
		},
			table.Entry("sending JSON and receiving JSON", rest2.MIME_JSON, rest2.MIME_JSON),
			table.Entry("sending JSON and receiving YAML", rest2.MIME_JSON, rest2.MIME_YAML),
			table.Entry("sending YAML and receiving JSON", rest2.MIME_YAML, rest2.MIME_JSON),
			table.Entry("sending YAML and receiving YAML", rest2.MIME_YAML, rest2.MIME_YAML),
		)
		It("POST should fail on missing mandatory field with 400", func() {
			sourceVMI.Spec = v1.VirtualMachineInstanceSpec{}
			result := restClient.Post().Resource(vmiResource).Namespace(k8sv1.NamespaceDefault).Body(sourceVMI).Do()
			Expect(result).To(HaveStatusCode(http.StatusBadRequest))
		})
		It("PUT should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmiPath(sourceVMI)),
					ghttp.VerifyJSONRepresenting(sourceVMI),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Put().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Body(sourceVMI).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		It("DELETE should fail", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Delete().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		It("GET should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Get().Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
		It("GET a VirtualMachineInstanceList should return an empty list", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.VirtualMachineInstanceList{TypeMeta: v12.TypeMeta{APIVersion: v1.GroupVersion.String(), Kind: "VirtualMachineInstanceList"}}),
				),
			)
			obj, err := restClient.Get().Resource(vmiResource).Namespace(k8sv1.NamespaceDefault).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(&v1.VirtualMachineInstanceList{}))
		})
		It("Merge Patch should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmiPath(sourceVMI)),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, sourceVMI),
				),
			)
			result := restClient.Patch(types.MergePatchType).Resource(vmiResource).Name(sourceVMI.GetObjectMeta().GetName()).Namespace(k8sv1.NamespaceDefault).Body(
				[]byte("{\"spec\" : { \"nodeSelector\": {\"test/lala\": \"blub\"}}}"),
			).Do()
			Expect(result).To(HaveStatusCode(http.StatusNotFound))
		})
	})

	AfterEach(func() {
		kubeproxy.Close()
	})

	AfterSuite(func() {
		apiserverMock.Close()
	})
})

func vmiBasePath() string {
	return "/apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances"
}

func vmiPath(vmi *v1.VirtualMachineInstance) string {
	return vmiBasePath() + "/" + vmi.GetObjectMeta().GetName()
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
