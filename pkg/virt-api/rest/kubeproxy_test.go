package rest_test

import (
	"flag"
	"net/http"
	"net/http/httptest"

	"github.com/emicklei/go-restful"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"golang.org/x/net/context"
	"io/ioutil"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	v12 "k8s.io/client-go/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	. "kubevirt.io/kubevirt/pkg/virt-api/rest"
)

var _ = Describe("Kubeproxy", func() {
	flag.Parse()
	var apiserverMock *ghttp.Server
	var kubeproxy *httptest.Server
	vmGVR := schema.GroupVersionResource{Group: v1.GroupVersion.Group, Version: v1.GroupVersion.Version, Resource: "vms"}
	ctx := context.Background()
	var restClient *rest.RESTClient
	var sourceVM *v1.VM

	// Work around go-client bug
	expectedVM := v1.NewMinimalVM("testvm")
	expectedVM.TypeMeta.Kind = ""
	expectedVM.TypeMeta.APIVersion = ""

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	BeforeSuite(func() {
		apiserverMock = ghttp.NewServer()
		flag.Lookup("master").Value.Set(apiserverMock.URL())
		ws, err := GenericResourceProxy(ctx, vmGVR, &v1.VM{}, v1.GroupVersionKind.Kind, &v1.VMList{})
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
		restful.Add(ws)

		kubeproxy = httptest.NewServer(restful.DefaultContainer)
		restClient, err = kubecli.GetRESTClientFromFlags(kubeproxy.URL, "")
		if err != nil {
			Expect(err).ToNot(HaveOccurred())
		}
	})

	BeforeEach(func() {
		sourceVM = v1.NewMinimalVM("testvm")
		apiserverMock.Reset()
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
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", http.StatusConflict))
		})
		It("PUT should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmBasePath()+"/testvm"),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			obj, err := restClient.Put().Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(expectedVM))
		})
		It("DELETE should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, struct{}{}),
				),
			)
			Expect(restClient.Delete().Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do().Error()).ToNot(HaveOccurred())
		})
		It("GET a VM should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			obj, err := restClient.Get().Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(expectedVM))
		})
		It("GET a VMList should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.VMList{TypeMeta: v12.TypeMeta{APIVersion: "kubevirt.io/v1alpha1", Kind: "VMList"}, Items: []v1.VM{*expectedVM}}),
				),
			)
			obj, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(&v1.VMList{Items: []v1.VM{*expectedVM}}))
		})
		It("Merge Patch should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmBasePath()+"/testvm"),
					returnReceivedBody(http.StatusOK),
				),
			)
			obj, err := restClient.Patch(api.MergePatchType).Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("{\"spec\" : { \"nodeSelector\": {\"test/lala\": \"blub\"}}}"),
			).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(Equal(expectedVM))
			Expect(obj.(*v1.VM).Spec.NodeSelector).Should(HaveKeyWithValue("test/lala", "blub"))
		})
		It("JSON Patch should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmBasePath()+"/testvm"),
					returnReceivedBody(http.StatusOK),
				),
			)
			obj, err := restClient.Patch(api.JSONPatchType).Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("[{ \"op\": \"replace\", \"path\": \"/spec/nodeSelector\", \"value\": {\"test/lala\": \"blub\" }}]"),
			).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).ToNot(Equal(expectedVM))
			Expect(obj.(*v1.VM).Spec.NodeSelector).Should(HaveKeyWithValue("test/lala", "blub"))
		})
		It("JSON Patch should fail on invalid update", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmBasePath()+"/testvm"),
					returnReceivedBody(http.StatusOK),
				),
			)
			result := restClient.Patch(api.JSONPatchType).Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("[{ \"op\": \"replace\", \"path\": \"/spec/nodeSelector\", \"value\": \"Only an object is allowed here\"}]"),
			).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", 422))
		})
	})

	Context("HTTP Operations on not existing VMs in the apiserver", func() {
		It("POST should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, vmBasePath()),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(Equal(expectedVM))
		})
		It("POST should fail on missing mandatory field with 400", func() {
			sourceVM.Spec = v1.VMSpec{}
			result := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", http.StatusBadRequest))
		})
		It("POST should succeed", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPost, vmBasePath()),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusOK, sourceVM),
				),
			)
			obj, err := restClient.Post().Resource("vms").Namespace(api.NamespaceDefault).Body(sourceVM).Do().Get()
			Expect(err).NotTo(HaveOccurred())
			Expect(obj).To(Equal(expectedVM))
		})
		It("PUT should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodPut, vmBasePath()+"/testvm"),
					ghttp.VerifyJSONRepresenting(sourceVM),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Put().Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(sourceVM).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", http.StatusNotFound))
		})
		It("DELETE should fail", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodDelete, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Delete().Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", http.StatusNotFound))
		})
		It("GET should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, struct{}{}),
				),
			)
			result := restClient.Get().Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", http.StatusNotFound))
		})
		It("GET a VMList should return an empty list", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()),
					ghttp.RespondWithJSONEncoded(http.StatusOK, v1.VMList{TypeMeta: v12.TypeMeta{APIVersion: "kubevirt.io/v1alpha1", Kind: "VMList"}}),
				),
			)
			obj, err := restClient.Get().Resource("vms").Namespace(api.NamespaceDefault).Do().Get()
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(&v1.VMList{}))
		})
		It("Merge Patch should fail with 404", func() {
			apiserverMock.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest(http.MethodGet, vmBasePath()+"/testvm"),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, sourceVM),
				),
			)
			result := restClient.Patch(api.MergePatchType).Resource("vms").Name(sourceVM.GetObjectMeta().GetName()).Namespace(api.NamespaceDefault).Body(
				[]byte("{\"spec\" : { \"nodeSelector\": {\"test/lala\": \"blub\"}}}"),
			).Do()
			Expect(result.Error()).To(HaveOccurred())
			Expect(result.Error().(*errors.StatusError).Status().Code).To(BeNumerically("==", http.StatusNotFound))
		})
	})

	AfterSuite(func() {
		apiserverMock.Close()
		kubeproxy.Close()
	})
})

func vmBasePath() string {
	return "/apis/kubevirt.io/v1alpha1/namespaces/default/vms"
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
