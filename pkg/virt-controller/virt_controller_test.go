package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	"net/http"

	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"strings"
)

func newValidRequest() *http.Request {
	request, _ := http.NewRequest("POST", "/api/v1/domain/raw", nil)
	request.Body = ioutil.NopCloser(strings.NewReader("<domain><name>testvm</name></domain>"))
	request.Header.Set("Content-Type", "application/xml")
	return request

}

type mockVmService struct{}

func (mockVmService) StartVmRaw(name string) error {
	return nil
}

var _ = Describe("VirtController", func() {
	var recorder *httptest.ResponseRecorder
	ctx := context.Background()
	svc := mockVmService{}
	endpoints := Handlers{
		RawDomainHandler: makeRawDomainHandler(ctx, makeRawDomainEndpoint(svc)),
	}
	handler := http.Handler(defineRoutes(&endpoints))

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request := newValidRequest()
				request.URL, _ = url.Parse("/api/v1/wrong/url")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(404))
			})
		})
		Context("with missing Content-Type header", func() {
			It("should return 404", func() {
				request := newValidRequest()
				request.Header.Del("Content-Type")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(404))
			})
		})
		Context("with invalid XML", func() {
			It("should return 400", func() {
				request := newValidRequest()
				request.Body = ioutil.NopCloser(strings.NewReader("test"))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(400))
			})
		})
		Context("with unexpected root XML element", func() {
			It("should return 400", func() {
				request := newValidRequest()
				request.Body = ioutil.NopCloser(strings.NewReader("<test><name>myvm</name></test>"))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(400))
			})
		})
		Context("with missing VM name", func() {
			It("should return 400", func() {
				request := newValidRequest()
				request.Body = ioutil.NopCloser(strings.NewReader("<domain><name></name></domain>"))
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(400))
			})
		})
		Context("with valid XMl", func() {
			It("should return 200", func() {
				request := newValidRequest()
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(200))
			})
		})
	})

})
