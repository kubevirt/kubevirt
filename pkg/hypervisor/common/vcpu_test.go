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

package common

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseCPUMask", func() {
	DescribeTable("should parse valid masks",
		func(mask string, expected map[string]MaskType) {
			result, err := ParseCPUMask(mask)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Mask).To(Equal(expected))
		},
		Entry("empty string", "", map[string]MaskType(nil)),
		Entry("single CPU", "5", map[string]MaskType{"5": Enabled}),
		Entry("CPU zero", "0", map[string]MaskType{"0": Enabled}),
		Entry("CPU range", "0-3", map[string]MaskType{
			"0": Enabled, "1": Enabled, "2": Enabled, "3": Enabled,
		}),
		Entry("range with same start and end", "3-3", map[string]MaskType{"3": Enabled}),
		Entry("negated CPU", "^2", map[string]MaskType{"2": Disabled}),
		Entry("mixed mask with range, single and negation", "0-3,^2,5", map[string]MaskType{
			"0": Enabled, "1": Enabled, "2": Disabled, "3": Enabled, "5": Enabled,
		}),
		Entry("whitespace in mask entries", "0, 1, 2", map[string]MaskType{
			"0": Enabled, "1": Enabled, "2": Enabled,
		}),
		Entry("negation before range preserves negation", "^1,0-3", map[string]MaskType{
			"0": Enabled, "1": Disabled, "2": Enabled, "3": Enabled,
		}),
		Entry("negation after range overrides range", "0-3,^1", map[string]MaskType{
			"0": Enabled, "1": Disabled, "2": Enabled, "3": Enabled,
		}),
		Entry("duplicate entries are deduplicated", "0,0,1", map[string]MaskType{
			"0": Enabled, "1": Enabled,
		}),
	)

	DescribeTable("should return error for invalid masks",
		func(mask, expectedSubstring string) {
			_, err := ParseCPUMask(mask)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedSubstring))
		},
		Entry("start > end in range", "5-2", "invalid mask range"),
		Entry("non-numeric value", "abc", "invalid mask value"),
		Entry("partially invalid mask", "0,abc,2", "invalid mask value"),
		Entry("leading comma", ",0", "invalid mask value"),
		Entry("trailing comma", "0,", "invalid mask value"),
		Entry("whitespace-only entry", "0, ,2", "invalid mask value"),
	)
})

var _ = Describe("IsEnabled", func() {
	DescribeTable("should report correct enabled state",
		func(mask string, vcpuID string, expected bool) {
			m, err := ParseCPUMask(mask)
			Expect(err).ToNot(HaveOccurred())
			Expect(m.IsEnabled(vcpuID)).To(Equal(expected))
		},
		Entry("enabled CPU in range", "0-3", "1", true),
		Entry("disabled CPU via negation", "0-3,^2", "2", false),
		Entry("CPU not in mask", "0-3", "10", false),
		Entry("single set CPU is enabled", "5", "5", true),
		Entry("unset CPU with non-empty mask", "^2", "0", false),
		Entry("negated-only CPU is disabled", "^0", "0", false),
	)

	It("should return true for all CPUs when mask is empty", func() {
		mask := CPUMask{}
		Expect(mask.IsEnabled("0")).To(BeTrue())
		Expect(mask.IsEnabled("99")).To(BeTrue())
	})
})
