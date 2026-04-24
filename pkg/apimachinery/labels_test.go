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
package apimachinery_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/validation"

	"kubevirt.io/kubevirt/pkg/apimachinery"
)

var _ = Describe("TruncateWithHash", func() {
	It("should return value unchanged when within limit", func() {
		Expect(apimachinery.TruncateWithHash("short-name", 253)).To(Equal("short-name"))
	})

	It("should return value unchanged when exactly at limit", func() {
		value := rand.String(253)
		Expect(apimachinery.TruncateWithHash(value, 253)).To(Equal(value))
	})

	It("should truncate and hash when exceeding limit", func() {
		value := rand.String(300)
		result := apimachinery.TruncateWithHash(value, 253)
		Expect(result).To(HaveLen(253))
	})

	It("should be deterministic", func() {
		value := rand.String(300)
		Expect(apimachinery.TruncateWithHash(value, 253)).To(Equal(apimachinery.TruncateWithHash(value, 253)))
	})

	It("should produce different results for different inputs", func() {
		value1 := rand.String(300)
		value2 := rand.String(300)
		Expect(apimachinery.TruncateWithHash(value1, 253)).ToNot(Equal(apimachinery.TruncateWithHash(value2, 253)))
	})
})

var _ = Describe("TruncateLabelValue", func() {
	It("should return value unchanged when within 63 chars", func() {
		Expect(apimachinery.TruncateLabelValue("short")).To(Equal("short"))
	})

	It("should truncate when exceeding 63 chars", func() {
		value := rand.String(100)
		Expect(apimachinery.TruncateLabelValue(value)).To(HaveLen(validation.LabelValueMaxLength))
	})
})
