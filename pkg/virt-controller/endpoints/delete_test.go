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

func newValidDeleteRequest() *http.Request {
	request, _ := http.NewRequest("DELETE", "/api/v1/delete/test", nil)
	return request
}

func testDeleteEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	Expect(request.(string)).To(Equal("test"))
	return payload{Name: request.(string)}, nil
}

var _ = Describe("Delete", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {
		handler = NewHandlerBuilder().Delete().Endpoint(testDeleteEndpoint).Build(ctx)
		router := mux.NewRouter()
		router.Methods("DELETE").Path("/api/v1/delete/{name:[a-zA-Z0-9]+}").Handler(handler)

		handler = http.Handler(router)
		request = newValidDeleteRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/v1/delete/?")
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
				Expect(recorder.Header().Get("Content-Type")).To(Equal("application/json"))
				Expect(responseBody).To(Equal(payload{Name: "test"}))
			})
		})
	})
})
