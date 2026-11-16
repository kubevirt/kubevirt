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

package multus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/multus"
)

var _ = Describe("Network selection annotation merge", func() {
	const podNamespace = "default"

	DescribeTable("should merge network selection annotations",
		func(current, desired string, absent []string, expected string) {
			merged, err := multus.MergeNetworkSelectionAnnotations(podNamespace, current, desired, absent...)

			Expect(err).NotTo(HaveOccurred())
			Expect(merged).To(MatchJSON(expected))
		},
		Entry("should preserve fields not owned by KubeVirt",
			`[{"name":"some-net","namespace":"default","interface":"pod1","mac":"00:00:00:00:00:00","foo":"bar","ips":["192.0.2.1"]}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1","mac":"02:00:00:00:00:02"}]`,
			[]string{},
			`[{"foo":"bar","ips":["192.0.2.1"],"interface":"pod1","mac":"02:00:00:00:00:02","name":"some-net","namespace":"default"}]`,
		),
		Entry("should merge an element that omits the namespace with its desired counterpart",
			`[{"name":"some-net","interface":"pod1","foo":"bar"}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1","mac":"02:00:00:00:00:02"}]`,
			[]string{},
			`[{"foo":"bar","interface":"pod1","mac":"02:00:00:00:00:02","name":"some-net","namespace":"default"}]`,
		),
		Entry("should remove an owned field that is absent from the desired annotation",
			`[{"name":"some-net","namespace":"default","interface":"pod1","mac":"00:00:00:00:00:00","foo":"bar"}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
			[]string{},
			`[{"foo":"bar","interface":"pod1","name":"some-net","namespace":"default"}]`,
		),
		Entry("should remove a current element with an explicit namespace via an absent element without one",
			`[
				{"name":"some-net","namespace":"default","interface":"pod1"},
				{"name":"other-net","namespace":"default","interface":"pod2"}
			]`,
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
			[]string{`[{"name":"some-net","interface":"pod1"}]`},
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
		),
		Entry("should preserve nested extra values and nested cni-args extras",
			`[{
				"name":"some-net","namespace":"default","interface":"pod1",
				"nested":{"a":[1,2,{"b":null}]},
				"cni-args":{"deviceID":"0000:01:00.0","meta":{"vf":3}}
			}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
			[]string{},
			`[{
				"cni-args":{"deviceID":"0000:01:00.0","meta":{"vf":3}},
				"interface":"pod1","name":"some-net","namespace":"default",
				"nested":{"a":[1,2,{"b":null}]}
			}]`,
		),
		Entry("should not change cni-args",
			`[{"name":"some-net","namespace":"default","interface":"pod1","cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"default"}}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
			[]string{},
			`[{"cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"default"},"interface":"pod1","name":"some-net","namespace":"default"}]`,
		),
		Entry("should support multiple binding plugins using the same NAD",
			`[
				{"name":"binding-net","namespace":"default","cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"net1"}},
				{"name":"binding-net","namespace":"default","cni-args":{"logicNetworkName":"net2"}}
			]`,
			`[
				{"name":"binding-net","namespace":"default","cni-args":{"logicNetworkName":"net1"}},
				{"name":"binding-net","namespace":"default","cni-args":{"logicNetworkName":"net2"}}
			]`,
			[]string{},
			`[
				{"cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"net1"},"name":"binding-net","namespace":"default"},
				{"cni-args":{"logicNetworkName":"net2"},"name":"binding-net","namespace":"default"}
			]`,
		),
		Entry("should remove only the absent binding plugin using the same NAD",
			`[
				{"name":"binding-net","namespace":"default","cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"net1"}},
				{"name":"binding-net","namespace":"default","cni-args":{"logicNetworkName":"net2"}},
				{"name":"binding-net","namespace":"default"}
			]`,
			`[{"name":"binding-net","namespace":"default","cni-args":{"logicNetworkName":"net1"}}]`,
			[]string{`[{"name":"binding-net","namespace":"default","cni-args":{"logicNetworkName":"net2"}}]`},
			`[
				{"cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"net1"},"name":"binding-net","namespace":"default"},
				{"name":"binding-net","namespace":"default"}
			]`,
		),
		Entry("should update the NAD of a binding plugin element when it is re-pointed",
			`[{"name":"old-nad","namespace":"default","cni-args":{"logicNetworkName":"net1","deviceID":"0000:01:00.0"}}]`,
			`[{"name":"new-nad","namespace":"default","cni-args":{"logicNetworkName":"net1"}}]`,
			[]string{},
			`[{"cni-args":{"deviceID":"0000:01:00.0","logicNetworkName":"net1"},"name":"new-nad","namespace":"default"}]`,
		),
		Entry("should remove a binding plugin element with an absent element naming another NAD",
			`[
				{"name":"old-nad","namespace":"default","cni-args":{"logicNetworkName":"net1"}},
				{"name":"other-net","namespace":"default","interface":"pod2"}
			]`,
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
			[]string{`[{"name":"net1","cni-args":{"logicNetworkName":"net1"}}]`},
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
		),
		Entry("should not identify elements that have an interface request by their cni-args logicNetworkName",
			`[
				{"name":"some-net","namespace":"default","interface":"pod1","cni-args":{"logicNetworkName":"shared"}},
				{"name":"other-net","namespace":"default","interface":"pod2","cni-args":{"logicNetworkName":"shared"}}
			]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
			[]string{`[{"name":"other-net","namespace":"default","interface":"pod2","cni-args":{"logicNetworkName":"shared"}}]`},
			`[
				{"cni-args":{"logicNetworkName":"shared"},"interface":"pod1","name":"some-net","namespace":"default"}
			]`,
		),
		Entry("should remove only absent elements and preserve external-only elements",
			`[
				{"name":"some-net","namespace":"default","interface":"pod1","foo":"bar"},
				{"name":"other-net","namespace":"default","interface":"pod2"},
				{"name":"external-net","namespace":"default"}
			]`,
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
			[]string{`[{"name":"some-net","namespace":"default","interface":"pod1"}]`},
			`[
				{"name":"other-net","namespace":"default","interface":"pod2"},
				{"name":"external-net","namespace":"default"}
			]`,
		),
		Entry("should remove elements from multiple absent annotations",
			`[
				{"name":"some-net","namespace":"default","interface":"pod1"},
				{"name":"other-net","namespace":"default","interface":"pod2"},
				{"name":"external-net","namespace":"default"}
			]`,
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
			[]string{
				`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
				`[{"name":"external-net","namespace":"default"}]`,
			},
			`[{"name":"other-net","namespace":"default","interface":"pod2"}]`,
		),
		Entry("should support multiple attachments to the same NAD",
			`[
				{"name":"some-net","namespace":"default","interface":"pod1","foo":"bar"},
				{"name":"some-net","namespace":"default","interface":"pod2","baz":"qux"}
			]`,
			`[{"name":"some-net","namespace":"default","interface":"pod2"}]`,
			[]string{`[{"name":"some-net","namespace":"default","interface":"pod1"}]`},
			`[{"baz":"qux","interface":"pod2","name":"some-net","namespace":"default"}]`,
		),
		Entry("should preserve external-only elements when all KubeVirt-owned elements are removed",
			`[
				{"name":"some-net","namespace":"default","interface":"pod1"},
				{"name":"external-net","namespace":"default"}
			]`,
			"",
			[]string{`[{"name":"some-net","namespace":"default","interface":"pod1"}]`},
			`[{"name":"external-net","namespace":"default"}]`,
		),
		Entry("should return the current element when the desired annotation matches it",
			`[{"foo":"bar","interface":"pod1","name":"some-net","namespace":"default"}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
			[]string{},
			`[{"foo":"bar","interface":"pod1","name":"some-net","namespace":"default"}]`,
		),
	)

	It("should preserve the current annotation when the desired annotation is a semantic no-op", func() {
		current := `[{"foo":"bar","interface":"pod1","name":"some-net","namespace":"default"}]`
		desired := `[{"name":"some-net","namespace":"default","interface":"pod1"}]`

		merged, err := multus.MergeNetworkSelectionAnnotations(podNamespace, current, desired)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(Equal(current))
	})

	It("should fall back to the desired annotation when the current annotation is not valid JSON", func() {
		desired := `[{"name":"some-net","namespace":"default","interface":"pod1"}]`

		merged, err := multus.MergeNetworkSelectionAnnotations(podNamespace, "not-json", desired)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(Equal(desired))
	})

	It("should fall back to the desired annotation when the current annotation has an invalid element", func() {
		desired := `[{"name":"some-net","namespace":"default","interface":"pod1"}]`

		merged, err := multus.MergeNetworkSelectionAnnotations(podNamespace, `[{"interface":"pod1"}]`, desired)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(Equal(desired))
	})

	It("should fail when the desired annotation is not valid JSON", func() {
		_, err := multus.MergeNetworkSelectionAnnotations(podNamespace, "", "not-json")

		Expect(err).To(HaveOccurred())
	})

	It("should fail when an absent annotation is not valid JSON", func() {
		_, err := multus.MergeNetworkSelectionAnnotations(podNamespace, "", "", "not-json")

		Expect(err).To(HaveOccurred())
	})

	It("should fail when a desired element has a non-object cni-args field", func() {
		_, err := multus.MergeNetworkSelectionAnnotations(podNamespace, "", `[{"name":"some-net","cni-args":"not-an-object"}]`)

		Expect(err).To(HaveOccurred())
	})

	It("should fail when a desired element is missing the name field", func() {
		_, err := multus.MergeNetworkSelectionAnnotations(podNamespace, "", `[{"interface":"pod1"}]`)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`missing the "name" field`))
	})

	It("should fail when the desired annotation has duplicate elements", func() {
		_, err := multus.MergeNetworkSelectionAnnotations(
			podNamespace,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
			`[
				{"name":"some-net","namespace":"default","interface":"pod1"},
				{"name":"some-net","namespace":"default","interface":"pod1"}
			]`,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate network selection element"))
	})

	It("should not duplicate an owned field spelled with non-canonical casing", func() {
		current := `[{"name":"some-net","namespace":"default","interface":"pod1","MAC":"00:00:00:00:00:00"}]`
		desired := `[{"name":"some-net","namespace":"default","interface":"pod1","mac":"02:00:00:00:00:02"}]`

		merged, err := multus.MergeNetworkSelectionAnnotations(podNamespace, current, desired)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(MatchJSON(`[{"interface":"pod1","mac":"02:00:00:00:00:02","name":"some-net","namespace":"default"}]`))
	})

	It("should preserve large integer values in extra fields", func() {
		current := `[{"name":"some-net","interface":"pod1","count":9007199254740993}]`

		merged, err := multus.MergeNetworkSelectionAnnotations(
			podNamespace,
			current,
			`[{"name":"some-net","namespace":"default","interface":"pod1"}]`,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(Equal(`[{"count":9007199254740993,"interface":"pod1","name":"some-net","namespace":"default"}]`))
	})
})
