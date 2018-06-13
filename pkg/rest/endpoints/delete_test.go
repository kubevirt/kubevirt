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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"

	"encoding/json"
	"net/http/httptest"
	"net/url"

	"github.com/emicklei/go-restful"
	"golang.org/x/net/context"

	"kubevirt.io/kubevirt/pkg/rest"
)

func newValidDeleteRequest() *http.Request {
	request, _ := http.NewRequest("DELETE", "/apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/test", nil)
	return request
}

func testDeleteEndpoint(_ context.Context, request interface{}) (interface{}, error) {
	Expect(request.(*PutObject).Metadata.Name).To(Equal("test"))
	return payload{Name: request.(*PutObject).Metadata.Name, Metadata: request.(*PutObject).Metadata}, nil
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

		ws.Route(ws.DELETE("/apis/kubevirt.io/v1alpha2/namespaces/{namespace}/virtualmachineinstances/{name}").To(target))
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
				Expect(responseBody).To(Equal(payload{Name: "test", Metadata: Metadata{Name: "test", Namespace: "default"}}))
			})
			It("should detect labelSelector", func() {
				request, _ := http.NewRequest("DELETE", "/apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/test?labelSelector=app%3Dmyapp", nil)
				handler.ServeHTTP(recorder, request)
				responseBody := payload{}
				json.NewDecoder(recorder.Body).Decode(&responseBody)
				Expect(responseBody.Metadata.Headers.LabelSelector).To(Equal("app=myapp"))
			})
		})
	})
})
