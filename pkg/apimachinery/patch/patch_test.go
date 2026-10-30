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

package patch_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
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

	It("should convert to a slice of strings", func() {
		patch := patch.New(patch.WithRemove("/abcd"),
			patch.WithAdd("/abcd", "test"),
			patch.WithReplace("/abcd", "test"),
			patch.WithTest("/abcd", "test"))

		slice, err := patch.ToSlice()
		Expect(err).NotTo(HaveOccurred())

		expectedSlice := []string{
			`{"op":"remove","path":"/abcd"}`,
			`{"op":"add","path":"/abcd","value":"test"}`,
			`{"op":"replace","path":"/abcd","value":"test"}`,
			`{"op":"test","path":"/abcd","value":"test"}`,
		}

		Expect(slice).To(Equal(expectedSlice))
	})

	It("should generate an empty set of patches", func() {
		patches := patch.New()
		Expect(patches.IsEmpty()).To(BeTrue())
	})
})

var _ = Describe("GeneratePerKeyMapPatches", func() {
	It("should test nil then add entire map when old map is nil", func() {
		newMap := map[string]string{"key1": "val1", "key2": "val2"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", nil, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(2))
		Expect(patches[0].Op).To(Equal("test"))
		Expect(patches[0].Path).To(Equal("/metadata/labels"))
		Expect(patches[0].Value).To(BeNil())
		Expect(patches[1].Op).To(Equal("add"))
		Expect(patches[1].Path).To(Equal("/metadata/labels"))
	})

	It("should test empty then add entire map when old map is empty", func() {
		newMap := map[string]string{"key1": "val1"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", map[string]string{}, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(2))
		Expect(patches[0].Op).To(Equal("test"))
		Expect(patches[0].Path).To(Equal("/metadata/labels"))
		Expect(patches[1].Op).To(Equal("add"))
		Expect(patches[1].Path).To(Equal("/metadata/labels"))
	})

	It("should generate per-key remove when new map is nil", func() {
		oldMap := map[string]string{"key1": "val1"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, nil)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(1))
		Expect(patches[0].Op).To(Equal("remove"))
		Expect(patches[0].Path).To(Equal("/metadata/labels/key1"))
	})

	It("should generate per-key add for new keys", func() {
		oldMap := map[string]string{"key1": "val1"}
		newMap := map[string]string{"key1": "val1", "key2": "val2"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(1))
		Expect(patches[0].Op).To(Equal("add"))
		Expect(patches[0].Path).To(Equal("/metadata/labels/key2"))
		Expect(patches[0].Value).To(Equal("val2"))
	})

	It("should generate per-key remove for deleted keys", func() {
		oldMap := map[string]string{"key1": "val1", "key2": "val2"}
		newMap := map[string]string{"key1": "val1"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(1))
		Expect(patches[0].Op).To(Equal("remove"))
		Expect(patches[0].Path).To(Equal("/metadata/labels/key2"))
	})

	It("should generate per-key test+replace for modified keys", func() {
		oldMap := map[string]string{"key1": "oldval"}
		newMap := map[string]string{"key1": "newval"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(2))
		Expect(patches[0].Op).To(Equal("test"))
		Expect(patches[0].Path).To(Equal("/metadata/labels/key1"))
		Expect(patches[0].Value).To(Equal("oldval"))
		Expect(patches[1].Op).To(Equal("replace"))
		Expect(patches[1].Path).To(Equal("/metadata/labels/key1"))
		Expect(patches[1].Value).To(Equal("newval"))
	})

	It("should escape JSON pointer characters in keys", func() {
		oldMap := map[string]string{"existing": "val"}
		newMap := map[string]string{"existing": "val", "kubevirt.io/nodeName": "node1"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()
		Expect(patches).To(HaveLen(1))
		Expect(patches[0].Op).To(Equal("add"))
		Expect(patches[0].Path).To(Equal("/metadata/labels/kubevirt.io~1nodeName"))
	})

	It("should generate no ops when maps are equal", func() {
		oldMap := map[string]string{"key1": "val1"}
		newMap := map[string]string{"key1": "val1"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		Expect(ops).To(BeEmpty())
	})

	It("should not emit test operations when only adding keys", func() {
		oldMap := map[string]string{"key1": "val1"}
		newMap := map[string]string{"key1": "val1", "key2": "val2"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		patchSet := patch.New(ops...)
		for _, p := range patchSet.GetPatches() {
			Expect(p.Op).ToNot(Equal("test"),
				"adding a new key should not require test operations on existing keys")
		}
	})

	It("should handle mixed add, remove, and modify in sorted key order", func() {
		oldMap := map[string]string{"keep": "same", "modify": "old", "remove": "gone"}
		newMap := map[string]string{"keep": "same", "modify": "new", "added": "fresh"}
		ops := patch.GeneratePerKeyMapPatches("/metadata/labels", oldMap, newMap)
		patchSet := patch.New(ops...)
		patches := patchSet.GetPatches()

		var opPaths []string
		for _, p := range patches {
			opPaths = append(opPaths, p.Op+" "+p.Path)
		}
		Expect(opPaths).To(Equal([]string{
			"test /metadata/labels/modify",
			"replace /metadata/labels/modify",
			"remove /metadata/labels/remove",
			"add /metadata/labels/added",
		}))
	})
})
