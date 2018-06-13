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
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/evanphx/json-patch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"kubevirt.io/kubevirt/pkg/middleware"
	"kubevirt.io/kubevirt/pkg/rest"
)

func newValidPatchRequest() *http.Request {
	request, _ := http.NewRequest("PATCH", "/apis/kubevirt.io/v1alpha2/namespaces/default/virtualmachineinstances/test", nil)
	request.Body = toReader("[{\"op\": \"replace\", \"path\": \"/email\", \"value\":\"newmail\"}]")
	request.Header.Set("Content-Type", rest.MIME_JSON_PATCH)
	return request
}

func testPatchEndpoint(ctx context.Context, request interface{}) (interface{}, error) {
	obj := request.(*PatchObject)
	originalPayload := payload{Email: "my@email", Name: "test"}
	rawOriginalpayload, err := json.Marshal(&originalPayload)
	Expect(err).ToNot(HaveOccurred())

	b, err := json.Marshal(obj.Patch)
	Expect(err).ToNot(HaveOccurred())

	patch, err := jsonpatch.DecodePatch(b)
	Expect(err).ToNot(HaveOccurred())
	rawPatchedBody, err := patch.Apply(rawOriginalpayload)
	if err != nil {
		return middleware.NewUnprocessibleEntityError(err), nil
	}

	Expect(json.Unmarshal(rawPatchedBody, &originalPayload)).To(Succeed())
	return &originalPayload, nil
}

var _ = Describe("Patch", func() {
	var recorder *httptest.ResponseRecorder
	var request *http.Request
	var handler http.Handler
	ctx := context.Background()

	BeforeEach(func() {

		ws := new(restful.WebService)
		ws.Produces(restful.MIME_JSON, rest.MIME_YAML).Consumes(rest.MIME_JSON_PATCH)
		handler = http.Handler(restful.NewContainer().Add(ws))

		target := MakeGoRestfulWrapper(NewHandlerBuilder().Patch().Endpoint(testPatchEndpoint).Build(ctx))
		ws.Route(ws.PATCH("/apis/kubevirt.io/v1alpha2/namespaces/{namespace}/virtualmachineinstances/{name}").To(target))

		request = newValidPatchRequest()
		recorder = httptest.NewRecorder()
	})

	Describe("REST call", func() {
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
		Context("with invalid patch operation", func() {
			It("should return 422", func() {
				request.Body = toReader("[{\"op\": \"nonexistent\", \"path\": \"/unknown\", \"value\":\"newmail\"}]")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(422))
			})
		})
		Context("with invalid field", func() {
			It("should return 200 and not update the object", func() {
				request.Body = toReader("[{\"op\": \"add\", \"path\": \"/unknown\", \"value\":\"newmail\"}]")
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
				responseBody := payload{}
				json.NewDecoder(recorder.Body).Decode(&responseBody)
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_JSON))
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "my@email"}))
			})
		})
		Context("with valid JSON Patch", func() {
			It("should return 200", func() {
				handler.ServeHTTP(recorder, request)
				Expect(recorder.Code).To(Equal(http.StatusOK))
			})
			It("should return a json containing the right name and email", func() {
				handler.ServeHTTP(recorder, request)
				responseBody := payload{}
				json.NewDecoder(recorder.Body).Decode(&responseBody)
				Expect(recorder.Header().Get("Content-Type")).To(Equal(rest.MIME_JSON))
				Expect(responseBody).To(Equal(payload{Name: "test", Email: "newmail"}))
			})
		})
	})
})
