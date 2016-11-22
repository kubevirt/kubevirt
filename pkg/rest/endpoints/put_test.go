package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"

	"bytes"
	"encoding/json"
	"github.com/emicklei/go-restful"
	"golang.org/x/net/context"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"strings"
)

func marshal(payload interface{}) io.ReadCloser {
	var b []byte
	buffer := bytes.NewBuffer(b)
	json.NewEncoder(buffer).Encode(payload)
	return ioutil.NopCloser(buffer)
}

type payload struct {
	Name  string `json:"name" valid:"required"`
	Email string `json:"email" valid:"email"`
}

func newValidPutRequest() *http.Request {
	request, _ := http.NewRequest("PUT", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/test", nil)
	request.Body = marshal(payload{Name: "test", Email: "test@test.com"})
	request.Header.Set("Content-Type", "application/json")
	return request
}

func testPutEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	Expect(request.(*PutObject).Metadata.Name).To(Equal("test"))
	return request.(*PutObject).Payload, nil
}

var _ = Describe("Put", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {

		ws := new(restful.WebService)
		ws.Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON)
		handler = http.Handler(restful.NewContainer().Add(ws))

		target := MakeGoRestfulWrapper(NewHandlerBuilder().Put((*payload)(nil)).Endpoint(testPutEndpoint).Build(ctx))
		ws.Route(ws.PUT("/apis/kubevirt.io/v1alpha1/namespaces/{namespace}/vms/{name}").To(target))

		request = newValidPutRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/v1/put/?")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with missing Content-Type header", func() {
			It("should return 415", func() {
				request.Header.Del("Content-Type")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusUnsupportedMediaType))
			})
		})
		Context("with invalid JSON", func() {
			It("should return 400", func() {
				request.Body = ioutil.NopCloser(strings.NewReader("test"))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with missing name field", func() {
			It("should return 400", func() {
				request.Body = marshal(payload{Email: "test@test.com"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with invalid email", func() {
			It("should return 400", func() {
				request.Body = marshal(payload{Name: "test", Email: "wrong"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with valid JSON", func() {
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
	})
})
