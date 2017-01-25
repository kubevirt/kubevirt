package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"

	"encoding/json"
	"github.com/emicklei/go-restful"
	"golang.org/x/net/context"
	"kubevirt.io/kubevirt/pkg/rest"
	"net/http/httptest"
	"net/url"
)

func newValidDeleteRequest() *http.Request {
	request, _ := http.NewRequest("DELETE", "/apis/kubevirt.io/v1alpha1/namespaces/default/vms/test", nil)
	return request
}

func testDeleteEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	Expect(request.(*Metadata).Name).To(Equal("test"))
	return payload{Name: request.(*Metadata).Name}, nil
}

var _ = Describe("Delete", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {

		ws := new(restful.WebService)
		handler = http.Handler(restful.NewContainer().Add(ws))

		target := MakeGoRestfulWrapper(NewHandlerBuilder().Delete().Endpoint(testDeleteEndpoint).Build(ctx))

		ws.Route(ws.DELETE("/apis/kubevirt.io/v1alpha1/namespaces/{namespace}/vms/{name}").To(target))
		request = newValidDeleteRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/rest/delete/?")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with valid request", func() {
			It("should return 200", func() {
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
			It("should return deleted entity", func() {
				handler.ServeHTTP(recorder, request)
				responseBody := payload{}
				json.NewDecoder(recorder.Body).Decode(&responseBody)
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_JSON))
				Expect(responseBody).To(Equal(payload{Name: "test"}))
			})
		})
	})
})
