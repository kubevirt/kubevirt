package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kithttp "github.com/go-kit/kit/transport/http"
	"net/http"

	"encoding/json"
	"fmt"
	"github.com/emicklei/go-restful"
	"golang.org/x/net/context"
	"gopkg.in/ini.v1"
	"net/http/httptest"
	"net/url"
)

func newValidGetRequest() *http.Request {
	request, _ := http.NewRequest("GET", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/test", nil)
	return request
}

func testGetEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	metadata := request.(*Metadata)
	Expect(metadata.Name).To(Equal("test"))
	return &payload{Name: "test", Email: "test@test.com"}, nil
}

var _ = Describe("Get", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {

		ws := new(restful.WebService)
		ws.Produces(restful.MIME_JSON)
		handler = http.Handler(restful.NewContainer().Add(ws))

		target := MakeGoRestfulWrapper(NewHandlerBuilder().Get().Endpoint(testGetEndpoint).
			Encoder(NewMimeTypeAwareEncoder(EncodeGetResponse, map[string]kithttp.EncodeResponseFunc{
				"text/plain": EncodeINIGetResponse,
			})).Build(ctx))
		ws.Route(ws.GET("/apis/kubevirt.io/v1alpha1/namespaces/{namespace}/vms/{name}").Produces("applicatoin/json", "text/plain").To(target))

		request = newValidGetRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/v1/put/")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with valid request", func() {
			It("should return 200", func() {
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
			It("should return a json containing the right name and email", func() {
				handler.ServeHTTP(recorder, request)
				responseBody := payload{}
				json.NewDecoder(recorder.Body).Decode(&responseBody)
				Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "test@test.com"}))
			})
		})
		Context("with Accept header text/plain", func() {
			It("should return ini file", func() {
				request.Header.Add("Accept", "text/plain")
				handler.ServeHTTP(recorder, request)
				responseBody := payload{}
				f, err := ini.Load(recorder.Body.Bytes())
				Expect(err).To(BeNil())
				Expect(recorder.Header().Get("Content-Type")).To(Equal("text/plain"))
				Expect(f.MapTo(&responseBody)).To(BeNil())
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "test@test.com"}))
			})
		})
	})
})
