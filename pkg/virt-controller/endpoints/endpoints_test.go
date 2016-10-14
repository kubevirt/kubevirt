package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	"net/http"

	"encoding/json"
	"github.com/go-kit/kit/log"
	"io/ioutil"
	"kubevirt/core/pkg/api"
	"kubevirt/core/pkg/api/v1"
	"kubevirt/core/pkg/middleware"
	"kubevirt/core/pkg/precond"
	"kubevirt/core/pkg/virt-controller/rest"
	"net/http/httptest"
	"net/url"
	"strings"
)

const validXML = `
<domain>
    <name>testvm</name>
    <uuid>0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f</uuid>
</domain>"
`

const missingNameXML = `
<domain>
    <uuid>0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f</uuid>
</domain>"
`

const invalidUUIDXML = `
<domain>
    <name>testvm</name>
    <uuid>0a81f5b2-8403-7b236-21cc</uuid>
</domain>"
`

const wrongRootElementXML = `
<test>
    <name>testvm</name>
    <uuid>0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f</uuid>
</test>"
`

func newValidRawDomainRequest() *http.Request {
	request, _ := http.NewRequest("POST", "/api/v1/domain/raw", nil)
	request.Body = ioutil.NopCloser(strings.NewReader(validXML))
	request.Header.Set("Content-Type", "application/xml")
	return request

}

type mockVMService struct {
	rawXML       []byte
	PrecondPanic bool
	Panic        bool
	VM           *api.VM
}

func (m *mockVMService) StartVMRaw(vm *api.VM, rawXML []byte) error {
	if m.PrecondPanic {
		panic(precond.MustNotBeEmpty(""))
	}
	if m.Panic {
		panic("panic")
	}
	m.VM = vm
	m.rawXML = rawXML
	return nil
}

var _ = Describe("Endpoints", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()
	var svc mockVMService

	BeforeEach(func() {
		svc = mockVMService{}
		endpoints := rest.Handlers{
			RawDomainHandler: MakeRawDomainHandler(ctx, middleware.InternalErrorMiddleware(log.NewLogfmtLogger(GinkgoWriter))(MakeRawDomainEndpoint(&svc))),
		}
		handler = http.Handler(rest.DefineRoutes(&endpoints))
		request = newValidRawDomainRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/v1/wrong/url")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with missing Content-Type header", func() {
			It("should return 404", func() {
				request.Header.Del("Content-Type")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with invalid XML", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader("test"))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with unexpected root XML element", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader(wrongRootElementXML))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with missing VM name", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader(missingNameXML))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with invalid UUID", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader(invalidUUIDXML))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with valid XMl", func() {
			It("should return 201", func() {
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusCreated))
			})
			It("should call vmService with the right arguments", func() {
				handler.ServeHTTP(recorder, request)
				Expect(svc.VM.UUID.String()).To(Equal("0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f"))
				Expect(svc.VM.Name).To(Equal("testvm"))
				Expect(string(svc.rawXML)).To(Equal(validXML))
			})
			It("should return a json containing the VM uuid", func() {
				handler.ServeHTTP(recorder, request)
				var responseDTO v1.VM
				json.NewDecoder(recorder.Body).Decode(&responseDTO)
				Expect(responseDTO).To(Equal(v1.VM{UUID: "0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f", Name: "testvm"}))
				Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))
			})
			Context("with Node-Selector header", func() {
				It("should pass on the node selector values to the service", func() {
					request.Header.Set("Node-Selector", "kubernetes.io/hostname=master")
					handler.ServeHTTP(recorder, request)
					var responseDTO v1.VM
					json.NewDecoder(recorder.Body).Decode(&responseDTO)
					Expect(svc.VM.NodeSelector).Should(HaveLen(1))
					Expect(svc.VM.NodeSelector).Should(HaveKeyWithValue("kubernetes.io/hostname", "master"))
				})
			})
			Context("with invalid Node-Selector header", func() {
				It("should return 400", func() {
					request.Header.Set("Node-Selector", "kubernetes.io/hostname")
					handler.ServeHTTP(recorder, request)
					Expect(recorder.Code).To(Equal(http.StatusBadRequest))
				})
			})
		})

		Context("with precondition panic", func() {
			It("should return 500", func() {
				svc.PrecondPanic = true
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
		Context("with panic", func() {
			It("should return 500", func() {
				svc.Panic = true
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
			})
		})
	})

})
