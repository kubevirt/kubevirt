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
	DescribeTable("should merge network selection annotations",
		func(current, desired string, absent []string, expected string) {
			merged, err := multus.MergeNetworkSelectionAnnotations(current, desired, absent...)

			Expect(err).NotTo(HaveOccurred())
			Expect(merged).To(MatchJSON(expected))
		},
		Entry("should preserve fields not owned by KubeVirt",
			`[{"name":"some-net","namespace":"default","interface":"pod1","mac":"00:00:00:00:00:00","foo":"bar","ips":["192.0.2.1"]}]`,
			`[{"name":"some-net","namespace":"default","interface":"pod1","mac":"02:00:00:00:00:02"}]`,
			[]string{},
			`[{"foo":"bar","ips":["192.0.2.1"],"interface":"pod1","mac":"02:00:00:00:00:02","name":"some-net","namespace":"default"}]`,
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

		merged, err := multus.MergeNetworkSelectionAnnotations(current, desired)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(Equal(current))
	})

	It("should fall back to the desired annotation when the current annotation is not valid JSON", func() {
		desired := `[{"name":"some-net","namespace":"default","interface":"pod1"}]`

		merged, err := multus.MergeNetworkSelectionAnnotations("not-json", desired)

		Expect(err).NotTo(HaveOccurred())
		Expect(merged).To(Equal(desired))
	})

	It("should fail when the desired annotation is not valid JSON", func() {
		_, err := multus.MergeNetworkSelectionAnnotations("", "not-json")

		Expect(err).To(HaveOccurred())
	})

	It("should fail when an absent annotation is not valid JSON", func() {
		_, err := multus.MergeNetworkSelectionAnnotations("", "", "not-json")

		Expect(err).To(HaveOccurred())
	})
})
