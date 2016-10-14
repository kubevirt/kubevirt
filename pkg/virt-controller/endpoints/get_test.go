package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	"net/http"

	"encoding/json"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"net/url"
)

func newValidGetRequest() *http.Request {
	request, _ := http.NewRequest("PUT", "/api/v1/get/test", nil)
	return request
}

func testGetEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	Expect(request.(string)).To(Equal("test"))
	return payload{Name: "test", Email: "test@test.com"}, nil
}

var _ = Describe("Get", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {
		handler = NewHandlerBuilder().Get().Endpoint(testGetEndpoint).Build(ctx)
		router := mux.NewRouter()
		router.Methods("PUT").Path("/api/v1/get/{name:[a-zA-Z0-9]+}").Handler(handler)

		handler = http.Handler(router)
		request = newValidGetRequest()
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
	})
})
