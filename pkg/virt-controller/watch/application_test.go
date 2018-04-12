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

package watch

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/rest"
)

func newValidGetRequest() *http.Request {
	request, _ := http.NewRequest("GET", "/leader", nil)
	return request
}

var _ = Describe("Application", func() {
	var app VirtControllerApp = VirtControllerApp{}

	Describe("Readiness probe", func() {
		var recorder *httptest.ResponseRecorder
		var request *http.Request
		var handler http.Handler

		BeforeEach(func() {
			app.readyChan = make(chan bool, 1)

			ws := new(restful.WebService)
			ws.Produces(restful.MIME_JSON)
			handler = http.Handler(restful.NewContainer().Add(ws))
			ws.Route(ws.GET("/leader").Produces(rest.MIME_JSON).To(app.leaderProbe))

			request = newValidGetRequest()
			recorder = httptest.NewRecorder()
		})

		Context("with closed channel", func() {
			It("should return 200 and that it is the leader", func() {

				close(app.readyChan)
				request.URL, _ = url.Parse("/leader")
				handler.ServeHTTP(recorder, request)
				var x map[string]interface{}
				Expect(json.Unmarshal(recorder.Body.Bytes(), &x)).To(Succeed())
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(x["apiserver"].(map[string]interface{})["leader"]).To(Equal("true"))
			})
		})
		Context("with opened channel", func() {
			It("should return 200 and that it is not the leader", func() {
				request.URL, _ = url.Parse("/leader")
				handler.ServeHTTP(recorder, request)
				var x map[string]interface{}
				Expect(json.Unmarshal(recorder.Body.Bytes(), &x)).To(Succeed())
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(x["apiserver"].(map[string]interface{})["leader"]).To(Equal("false"))
			})
		})
	})
})
