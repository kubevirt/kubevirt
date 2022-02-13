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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package prometheus

import (
	"net/http"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
})

var _ = Describe("URL Parsing", func() {
	Context("with resource and operation", func() {
		DescribeTable("accurately parse resource and operation", func(urlStr, method, expectedResource, expectedOperation string) {

			request := &http.Request{
				Method: method,
				URL: &url.URL{
					Path: urlStr,
				},
			}

			resource, operation := parseURLResourceOperation(request)
			Expect(resource).To(Equal(expectedResource))
			Expect(operation).To(Equal(expectedOperation))

		},
			Entry("should handle an empty URL and method", "", "", "", ""),
			Entry("should handle an empty URL", "", "GET", "", ""),
			Entry("should handle an empty Method", "/api/v1/watch/namespaces/kubevirt/pods", "", "", ""),
			Entry("should handle watching namespaced resource", "/api/v1/watch/namespaces/kubevirt/pods", "GET", "pods", "WATCH"),
			Entry("should handle watching globally scoped resource", "/api/v1/watch/pods", "GET", "pods", "WATCH"),
			Entry("should handle list of namespaced resources", "/api/v1/namespaces/kubevirt/pods", "GET", "pods", "LIST"),
			Entry("should handle get of namespaced resources", "/api/v1/namespaces/kubevirt/pods/my-pod", "GET", "pods", "GET"),
			Entry("should handle list of custom namespaced resources", "/apis/kubevirt.io/v1/namespaces/kubevirt/virtualmachineinstances", "GET", "virtualmachineinstances", "LIST"),
			Entry("should handle get of custom namespaced resources", "/apis/kubevirt.io/v1/namespaces/kubevirt/virtualmachineinstances/my-vmi", "GET", "virtualmachineinstances", "GET"),
			Entry("should handle list of custom globally scoped resources", "/apis/kubevirt.io/v1/kubevirts", "GET", "kubevirts", "LIST"),
			Entry("should handle get of custom globally scoped resources", "/apis/kubevirt.io/v1/kubevirts/my-kv", "GET", "kubevirts", "GET"),
			Entry("should handle UPDATE of namespaced resources", "/api/v1/namespaces/kubevirt/pods/my-pod", "PUT", "pods", "UPDATE"),
			Entry("should handle PATCH of namespaced resources", "/api/v1/namespaces/kubevirt/pods/my-pod", "PATCH", "pods", "PATCH"),
			Entry("should handle CREATE of namespaced resources", "/api/v1/namespaces/kubevirt/pods/my-pod", "POST", "pods", "CREATE"),
			Entry("should handle DELETE of namespaced resources", "/api/v1/namespaces/kubevirt/pods/my-pod", "DELETE", "pods", "DELETE"),
			Entry("should handle UPDATE to status subresource", "/api/v1/namespaces/kubevirt/pods/my-pod/status", "PUT", "pods", "UPDATE"),
			Entry("should handle UPDATE to custom subresource", "/apis/kubevirt.io/v1/namespaces/kubevirt/virtualmachineinstances/my-vmi/some-subresource", "PUT", "virtualmachineinstances", "UPDATE"),
		)
	})

})
