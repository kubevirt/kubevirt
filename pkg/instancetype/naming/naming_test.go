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
package naming_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/validation"

	"kubevirt.io/kubevirt/pkg/instancetype/naming"
)

var _ = Describe("TruncateWithHash", func() {
	It("should return value unchanged when within limit", func() {
		Expect(naming.TruncateWithHash("short-name", 253)).To(Equal("short-name"))
	})

	It("should return value unchanged when exactly at limit", func() {
		value := strings.Repeat("a", 253)
		Expect(naming.TruncateWithHash(value, 253)).To(Equal(value))
	})

	It("should truncate and hash when exceeding limit", func() {
		value := strings.Repeat("a", 300)
		result := naming.TruncateWithHash(value, 253)
		Expect(result).To(HaveLen(253))
	})

	It("should be deterministic", func() {
		value := strings.Repeat("x", 300)
		Expect(naming.TruncateWithHash(value, 253)).To(Equal(naming.TruncateWithHash(value, 253)))
	})

	It("should produce different results for different inputs", func() {
		value1 := strings.Repeat("a", 300)
		value2 := strings.Repeat("b", 300)
		Expect(naming.TruncateWithHash(value1, 253)).ToNot(Equal(naming.TruncateWithHash(value2, 253)))
	})
})

var _ = Describe("TruncateLabelValue", func() {
	It("should return value unchanged when within 63 chars", func() {
		Expect(naming.TruncateLabelValue("short")).To(Equal("short"))
	})

	It("should truncate when exceeding 63 chars", func() {
		value := strings.Repeat("a", 100)
		Expect(naming.TruncateLabelValue(value)).To(HaveLen(validation.LabelValueMaxLength))
	})
})
