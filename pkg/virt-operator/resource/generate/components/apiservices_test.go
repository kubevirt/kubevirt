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
 */

package components

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("APIServices", func() {

	It("should load one APIService with the correct namespace", func() {
		services := NewVirtAPIAPIServices("mynamespace")
		// a subresource aggregated api endpoint should be registered for
		// each vm/vmi api version
		Expect(services).To(HaveLen(len(v1.SubresourceGroupVersions)))
		Expect(services[0].Spec.Service.Namespace).To(Equal("mynamespace"))
	})
})
