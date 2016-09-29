package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	"net/http"

	"encoding/json"
	"github.com/satori/go.uuid"
	"io/ioutil"
	"kubevirt/core/pkg/virt-controller/entities"
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

const missingUUIDXML = `
<domain>
    <name>testvm</name>
</domain>"
`

const wrongRootElementXML = `
<test>
    <name>testvm</name>
    <uuid>0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f</uuid>
</test>"
`

func newValidRequest() *http.Request {
	request, _ := http.NewRequest("POST", "/api/v1/domain/raw", nil)
	request.Body = ioutil.NopCloser(strings.NewReader(validXML))
	request.Header.Set("Content-Type", "application/xml")
	return request

}

type mockVMService struct {
	UUID   uuid.UUID
	VMName string
	rawXML []byte
}

func (m *mockVMService) StartVMRaw(vm *entities.VM, rawXML []byte) error {
	m.UUID = vm.UUID
	m.VMName = vm.Name
	m.rawXML = rawXML
	return nil
}

func (m *mockVMService) Clear() {
	m.UUID = uuid.Nil
	m.VMName = ""
	m.rawXML = nil
}

var _ = Describe("Endpoints", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	ctx := context.Background()
	svc := mockVMService{}
	endpoints := rest.Handlers{
		RawDomainHandler: MakeRawDomainHandler(ctx, MakeRawDomainEndpoint(&svc)),
	}
	handler := http.Handler(rest.DefineRoutes(&endpoints))

	BeforeEach(func() {
		request = newValidRequest()
		recorder = httptest.NewRecorder()
		svc.Clear()
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
		Context("with missing UUID", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader(missingUUIDXML))
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
				Expect(svc.UUID.String()).To(Equal("0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f"))
				Expect(svc.VMName).To(Equal("testvm"))
				Expect(string(svc.rawXML)).To(Equal(validXML))
			})
			It("should return a json containing the VM uuid", func() {
				handler.ServeHTTP(recorder, request)
				var responseDTO VMResponseDTO
				json.NewDecoder(recorder.Body).Decode(&responseDTO)
				Expect(responseDTO).To(Equal(VMResponseDTO{UUID: "0a81f5b2-8403-7b23-c8d6-21ccc2f80d6f"}))
				Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))
			})
		})
	})

})
