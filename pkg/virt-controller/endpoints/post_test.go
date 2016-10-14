package endpoints

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	"net/http"

	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"strings"
)

func newValidPostRequest() *http.Request {
	request, _ := http.NewRequest("POST", "/api/v1/post", nil)
	request.Body = marshal(payload{Name: "test", Email: "test@test.com"})
	request.Header.Set("Content-Type", "application/json")
	return request
}

func testPostEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	return request, nil
}

var _ = Describe("Post", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {
		handler = NewHandlerBuilder().Post((*payload)(nil)).Endpoint(testPostEndpoint).Build(ctx)
		router := mux.NewRouter()
		router.Methods("POST").Path("/api/v1/post").Headers("Content-Type", "application/json").Handler(handler)

		handler = http.Handler(router)
		request = newValidPostRequest()
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
			It("should return 201", func() {
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusCreated))
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
