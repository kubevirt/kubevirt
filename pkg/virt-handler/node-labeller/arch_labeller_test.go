/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller

import (
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Arch Node Labeller", func() {

	DescribeTable("Should create a new archLabeller for the correct architecture", func(arch string, result archLabeller) {
		ac := newArchLabeller(arch)

		Expect(ac).To(Equal(result))
		if arch == "unknown" {
			arch = runtime.GOARCH
		}
		Expect(ac.arch()).To(Equal(arch))
	},
		Entry(amd64, amd64, archLabellerAMD64{}),
		Entry(arm64, arm64, archLabellerARM64{}),
		Entry(s390x, s390x, archLabellerS390X{}),
		Entry("unknown", "unknown", defaultArchLabeller{}),
	)
})
