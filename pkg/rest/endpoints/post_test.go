/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package endpoints

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/ghodss/yaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"kubevirt.io/kubevirt/pkg/rest"
)

func newValidJSONPostRequest() *http.Request {
	request, _ := http.NewRequest("POST", "/apis/kubevirt.io/v1alpha3/namespaces/default/virtualmachineinstances", nil)
	request.Body = marshalToJSON(payload{Name: "test", Email: "test@test.com"})
	request.Header.Set("Content-Type", rest.MIME_JSON)
	return request
}

func testPostEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	return request.(*PutObject).Payload, nil
}

var _ = Describe("Post", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {

		ws := new(restful.WebService)
		ws.Produces(restful.MIME_JSON, rest.MIME_YAML).Consumes(restful.MIME_JSON, rest.MIME_YAML)
		handler = http.Handler(restful.NewContainer().Add(ws))

		target := MakeGoRestfulWrapper(NewHandlerBuilder().Post((*payload)(nil)).Endpoint(testPostEndpoint).Build(ctx))
		ws.Route(ws.POST("/apis/kubevirt.io/v1alpha3/namespaces/{namespace}/virtualmachineinstances").To(target))

		request = newValidJSONPostRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
		Context("with invalid URL", func() {
			It("should return 404", func() {
				request.URL, _ = url.Parse("/api/rest/wrong/url")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusNotFound))
			})
		})
		Context("with missing Content-Type header", func() {
			It("should return 414", func() {
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
				request.Body = marshalToJSON(payload{Email: "test@test.com"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			})
		})
		Context("with invalid email", func() {
			It("should return 400", func() {
				request.Body = marshalToJSON(payload{Name: "test", Email: "wrong"})
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
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_JSON))
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "test@test.com"}))
			})
		})
		Context("with valid YAML", func() {
			It("should accept it and return it as YAML", func() {
				request.Header.Set("Content-Type", rest.MIME_YAML)
				request.Header.Set("Accept", rest.MIME_YAML)
				request.Body = marshalToYAML(&payload{Name: "test", Email: "test@test.com"})
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusCreated))
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_YAML))
				responseBody := payload{}
				yaml.Unmarshal(recorder.Body.Bytes(), &responseBody)
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "test@test.com"}))
			})
		})
	})
})
