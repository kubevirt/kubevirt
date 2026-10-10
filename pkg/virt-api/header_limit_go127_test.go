//go:build go1.27

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

package virt_api

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("identity header limit", func() {
	DescribeTable("limits forwarded group headers according to the server configuration",
		func(configure bool, groups int, wantCode int) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			if configure {
				configureIdentityHeaderLimit(server.Config)
			}
			server.Start()
			defer server.Close()

			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			Expect(err).NotTo(HaveOccurred())
			for range groups {
				request.Header.Add("X-Remote-Group", "group")
			}
			response, err := server.Client().Do(request)
			Expect(err).NotTo(HaveOccurred())
			defer response.Body.Close()
			Expect(response.StatusCode).To(Equal(wantCode))
		},
		Entry("rejects 1000 groups with the Go default", false, 1000, http.StatusRequestHeaderFieldsTooLarge),
		Entry("accepts 1000 groups with the configured limit", true, 1000, http.StatusOK),
		Entry("rejects more values than the configured limit", true, identityHeaderValueLimit+1, http.StatusRequestHeaderFieldsTooLarge),
	)
})
