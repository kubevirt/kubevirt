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

package hugepages_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hugepages"
	"kubevirt.io/kubevirt/pkg/pointer"
)

var _ = Describe("hugepages helpers", func() {
	DescribeTable("EffectiveMode", func(hp *v1.Hugepages, expected v1.HugepagesMode) {
		Expect(hugepages.EffectiveMode(hp)).To(Equal(expected))
	},
		Entry("nil hugepages", nil, v1.HugepagesModeStatic),
		Entry("mode unset", &v1.Hugepages{PageSize: "2Mi"}, v1.HugepagesModeStatic),
		Entry("mode empty", &v1.Hugepages{Mode: pointer.P(v1.HugepagesMode(""))}, v1.HugepagesModeStatic),
		Entry("mode static", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeStatic)}, v1.HugepagesModeStatic),
		Entry("mode transparent", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeTransparent)}, v1.HugepagesModeTransparent),
	)

	DescribeTable("EffectivePolicy", func(hp *v1.Hugepages, expected v1.HugepagesPolicy) {
		Expect(hugepages.EffectivePolicy(hp)).To(Equal(expected))
	},
		Entry("nil hugepages", nil, v1.HugepagesPolicyBestEffort),
		Entry("policy unset", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeTransparent)}, v1.HugepagesPolicyBestEffort),
		Entry("policy empty", &v1.Hugepages{Policy: pointer.P(v1.HugepagesPolicy(""))}, v1.HugepagesPolicyBestEffort),
		Entry("policy bestEffort", &v1.Hugepages{Policy: pointer.P(v1.HugepagesPolicyBestEffort)}, v1.HugepagesPolicyBestEffort),
		Entry("policy guaranteed", &v1.Hugepages{Policy: pointer.P(v1.HugepagesPolicyGuaranteed)}, v1.HugepagesPolicyGuaranteed),
	)

	DescribeTable("IsTransparent", func(hp *v1.Hugepages, expected bool) {
		Expect(hugepages.IsTransparent(hp)).To(Equal(expected))
	},
		Entry("nil", nil, false),
		Entry("unset", &v1.Hugepages{}, false),
		Entry("static", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeStatic)}, false),
		Entry("transparent", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeTransparent)}, true),
	)

	DescribeTable("ForbidsPostCopy", func(hp *v1.Hugepages, expected bool) {
		Expect(hugepages.ForbidsPostCopy(hp)).To(Equal(expected))
	},
		Entry("nil", nil, false),
		Entry("static", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeStatic)}, false),
		Entry("transparent bestEffort default", &v1.Hugepages{Mode: pointer.P(v1.HugepagesModeTransparent)}, false),
		Entry("transparent bestEffort", &v1.Hugepages{
			Mode:   pointer.P(v1.HugepagesModeTransparent),
			Policy: pointer.P(v1.HugepagesPolicyBestEffort),
		}, false),
		Entry("transparent guaranteed", &v1.Hugepages{
			Mode:   pointer.P(v1.HugepagesModeTransparent),
			Policy: pointer.P(v1.HugepagesPolicyGuaranteed),
		}, true),
		Entry("static with guaranteed policy still forbids only when transparent", &v1.Hugepages{
			Mode:   pointer.P(v1.HugepagesModeStatic),
			Policy: pointer.P(v1.HugepagesPolicyGuaranteed),
		}, false),
	)
})
