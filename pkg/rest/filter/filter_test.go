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
 * Copyright The KubeVirt Authors.
 *
 */

package filter_test

import (
	"net/http"
	"net/http/httptest"

	restful "github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/rest/filter"
)

const (
	xContentTypeOptions      = "X-Content-Type-Options"
	xContentTypeOptionsValue = "nosniff"
	someOtherValue           = "some-other-value"
)

var _ = Describe("SecurityHeadersFilter", func() {
	var (
		recorder *httptest.ResponseRecorder
		req      *restful.Request
		resp     *restful.Response
	)

	BeforeEach(func() {
		recorder = httptest.NewRecorder()
		req = restful.NewRequest(&http.Request{})
		resp = restful.NewResponse(recorder)
	})

	It("should set X-Content-Type-Options to nosniff", func() {
		chain := &restful.FilterChain{
			Filters: []restful.FilterFunction{filter.SecurityHeadersFilter()},
			Target:  func(_ *restful.Request, _ *restful.Response) {},
		}
		chain.ProcessFilter(req, resp)

		Expect(recorder.Header().Get(xContentTypeOptions)).To(Equal(xContentTypeOptionsValue))
	})

	It("should allow a target handler to override X-Content-Type-Options", func() {
		// SecurityHeadersFilter sets the header before calling the chain, so a target
		// handler that explicitly sets the same header afterwards takes precedence.
		// Handlers in this codebase should not override this header.
		chain := &restful.FilterChain{
			Filters: []restful.FilterFunction{filter.SecurityHeadersFilter()},
			Target: func(_ *restful.Request, resp *restful.Response) {
				resp.Header().Set(xContentTypeOptions, someOtherValue)
			},
		}
		chain.ProcessFilter(req, resp)

		Expect(recorder.Header().Get(xContentTypeOptions)).To(Equal(someOtherValue))
	})
})
