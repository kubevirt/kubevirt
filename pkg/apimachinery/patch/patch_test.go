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
 * Copyright The KubeVirt Authors
 *
 */

package patch_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("PatchSet", func() {

	It("should generate correct patch remove operation", func() {
		Expect(patch.New(patch.WithRemove("/abcd")).GeneratePayload()).To(Equal([]byte(
			`[{"op":"remove","path":"/abcd"}]`)))
	})

	DescribeTable("should generate correct patch add operation", func(v interface{}, expected string) {
		Expect(patch.New(patch.WithAdd("/abcd", v)).GeneratePayload()).To(Equal([]byte(
			fmt.Sprintf(`[{"op":"add","path":"/abcd","value":%s}]`, expected),
		)))
	},
		Entry("with value", "test", `"test"`),
		Entry("with nil value", nil, `null`),
	)

	DescribeTable("should generate correct patch replace operation", func(v interface{}, expected string) {
		Expect(patch.New(patch.WithReplace("/abcd", v)).GeneratePayload()).To(Equal([]byte(
			fmt.Sprintf(`[{"op":"replace","path":"/abcd","value":%s}]`, expected),
		)))
	},
		Entry("with value", "test", `"test"`),
		Entry("with nil value", nil, `null`),
	)

	DescribeTable("should generate correct patch test operation", func(v interface{}, expected string) {
		Expect(patch.New(patch.WithTest("/abcd", v)).GeneratePayload()).To(Equal([]byte(
			fmt.Sprintf(`[{"op":"test","path":"/abcd","value":%s}]`, expected),
		)))
	},
		Entry("with value", "test", `"test"`),
		Entry("with nil value", nil, `null`),
	)

	It("should generate correct patch with a mix of operations", func() {
		Expect(patch.New(patch.WithRemove("/abcd"),
			patch.WithAdd("/abcd", "test"),
			patch.WithReplace("/abcd", "test"),
			patch.WithTest("/abcd", "test")).GeneratePayload()).To(Equal([]byte(
			`[{"op":"remove","path":"/abcd"},{"op":"add","path":"/abcd","value":"test"},{"op":"replace","path":"/abcd","value":"test"},{"op":"test","path":"/abcd","value":"test"}]`,
		)))
	})

	It("should generate an empty set of patches", func() {
		patches := patch.New()
		Expect(patches.IsEmpty()).To(BeTrue())
	})

	It("should add the patch bytes with a wrong patch, it should generate an error", func() {
		Expect(patch.New().AddRawPatch([]byte(`{"something": "wrong"}`))).To(HaveOccurred())
	})
	It("should add the patch bytes with a valid patch", func() {
		Expect(patch.New().AddRawPatch([]byte(`[{"op":"replace","path":"/abcd","value":"test"}]`))).ToNot(HaveOccurred())
	})

	DescribeTable("should unmarshal the patch", func(operation *patch.PatchOp, expected string) {
		var t string
		p := []byte(`[{"op":"remove","path":"/abcd"},{"op":"add","path":"/abcd","value":"test1"},{"op":"replace","path":"/abcd","value":"test2"},{"op":"test","path":"/abcd","value":"test3"}]`)
		patchSet := patch.New()
		Expect(patchSet.AddRawPatch(p)).ToNot(HaveOccurred())
		Expect(patchSet.UnmarshalPatchValue("/abcd", operation, &t)).ToNot(HaveOccurred())
		Expect(t).To(Equal(expected))
	},
		Entry("without the operation", nil, "test1"),
		Entry("without the add operation", pointer.P(patch.PatchAddOp), "test1"),
		Entry("without the replace operation", pointer.P(patch.PatchReplaceOp), "test2"),
		Entry("without the test operation", pointer.P(patch.PatchTestOp), "test3"),
	)

	It("should fail to unmarshal with a remove operation", func() {
		var t string
		patchSet := patch.New(patch.WithRemove("/abcd"))
		Expect(patchSet.UnmarshalPatchValue("/abcd", pointer.P(patch.PatchRemoveOp), &t)).To(HaveOccurred())
	})

	It("should fail if the patch doesn't contain the path", func() {
		var t string
		patchSet := patch.New(patch.WithAdd("/abcd", "test"))
		Expect(patchSet.UnmarshalPatchValue("/wrong", pointer.P(patch.PatchAddOp), &t)).To(
			MatchError("the path or operation doesn't exist in the patch"))
	})

	It("should fail if the patch doesn't contain the operation", func() {
		var t string
		patchSet := patch.New(patch.WithAdd("/abcd", "test"))
		Expect(patchSet.UnmarshalPatchValue("/abcd", pointer.P(patch.PatchReplaceOp), &t)).To(
			MatchError("the path or operation doesn't exist in the patch"))
	})

	It("should unmarshal each single patch", func() {
		p := []byte(`[{"op":"remove","path":"/abcd"},{"op":"add","path":"/abcd","value":"test1"},{"op":"replace","path":"/abcd","value":"test2"},{"op":"test","path":"/abcd","value":"test3"}]`)
		patchSet := patch.New()
		Expect(patchSet.AddPatch(p)).ToNot(HaveOccurred())
		patches, err := patchSet.Unmarshal()
		Expect(err).ToNot(HaveOccurred())
		Expect(patches).To(HaveLen(4))
		Expect(patches[0]).To(Equal(`{"op":"remove","path":"/abcd"}`))
		Expect(patches[1]).To(Equal(`{"op":"add","path":"/abcd","value":"test1"}`))
		Expect(patches[2]).To(Equal(`{"op":"replace","path":"/abcd","value":"test2"}`))
		Expect(patches[3]).To(Equal(`{"op":"test","path":"/abcd","value":"test3"}`))

	})
})
