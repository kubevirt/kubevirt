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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package patch_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

var _ = Describe("PatchSet", func() {

	It("Patch remove operation", func() {
		Expect(patch.New(patch.WithRemove("/abcd")).GeneratePayload()).To(Equal([]byte(
			`[{"op":"remove","path":"/abcd"}]`)))
	})

	DescribeTable("Patch add operation", func(v interface{}, expected string) {
		Expect(patch.New(patch.WithAdd("/abcd", v)).GeneratePayload()).To(Equal([]byte(
			fmt.Sprintf(`[{"op":"add","path":"/abcd","value":%s}]`, expected),
		)))
	},
		Entry("with value", "test", `"test"`),
		Entry("with nil value", nil, `null`),
	)

	DescribeTable("Patch replace operation", func(v interface{}, expected string) {
		Expect(patch.New(patch.WithReplace("/abcd", v)).GeneratePayload()).To(Equal([]byte(
			fmt.Sprintf(`[{"op":"replace","path":"/abcd","value":%s}]`, expected),
		)))
	},
		Entry("with value", "test", `"test"`),
		Entry("with nil value", nil, `null`),
	)

	DescribeTable("Patch test operation", func(v interface{}, expected string) {
		Expect(patch.New(patch.WithTest("/abcd", v)).GeneratePayload()).To(Equal([]byte(
			fmt.Sprintf(`[{"op":"test","path":"/abcd","value":%s}]`, expected),
		)))
	},
		Entry("with value", "test", `"test"`),
		Entry("with nil value", nil, `null`),
	)

	It("Patch with a mix of operations", func() {
		Expect(patch.New(patch.WithRemove("/abcd"),
			patch.WithAdd("/abcd", "test"),
			patch.WithReplace("/abcd", "test"),
			patch.WithTest("/abcd", "test")).GeneratePayload()).To(Equal([]byte(
			`[{"op":"remove","path":"/abcd"},{"op":"add","path":"/abcd","value":"test"},{"op":"replace","path":"/abcd","value":"test"},{"op":"test","path":"/abcd","value":"test"}]`,
		)))
	})

	It("Empty set of patches", func() {
		patches := patch.New()
		Expect(patches.IsEmpty()).To(BeTrue())
	})
})
