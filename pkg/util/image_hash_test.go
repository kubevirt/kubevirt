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

package util

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mirrors k8s.io/apimachinery/pkg/util/validation.IsValidLabelValue
var validLabelValue = regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`)

var _ = Describe("ImageHashLabelValue", func() {
	It("should return an empty string for an empty image", func() {
		Expect(ImageHashLabelValue("")).To(Equal(""))
	})

	It("should be deterministic for the same image", func() {
		image := "registry.example.com/kubevirt/virt-handler@sha256:" + "a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abc"
		Expect(ImageHashLabelValue(image)).To(Equal(ImageHashLabelValue(image)))
	})

	It("should differ for different images", func() {
		imageA := "registry.example.com/kubevirt/virt-handler@sha256:" + "a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abc"
		imageB := "registry.example.com/kubevirt/virt-handler@sha256:" + "b0123456789abcdef0123456789abcdef0123456789abcdef0123456789abc"
		Expect(ImageHashLabelValue(imageA)).ToNot(Equal(ImageHashLabelValue(imageB)))
	})

	DescribeTable("should always produce a valid Kubernetes label value",
		func(image string) {
			value := ImageHashLabelValue(image)
			Expect(len(value)).To(BeNumerically("<=", 63))
			Expect(value).To(MatchRegexp(validLabelValue.String()))
		},
		Entry("a long image reference with a digest", "registry.example.com/some/deeply/nested/repository/virt-handler@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"),
		Entry("an image reference with a tag", "registry.example.com/kubevirt/virt-handler:v1.4.0"),
		Entry("a bare image name", "virt-handler"),
	)
})
