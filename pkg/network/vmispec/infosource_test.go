/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

const (
	emptySourceInfo = ""
)

var _ = Describe("infoSource", func() {
	DescribeTable("can add an infoSource entry",
		func(infoSourceData, infoSourceItem, expectedInfoSourceData string) {
			Expect(vmispec.AddInfoSource(infoSourceData, infoSourceItem)).To(Equal(expectedInfoSourceData))
		},
		Entry("given no infoSource entries",
			emptySourceInfo, vmispec.InfoSourceDomain, vmispec.InfoSourceDomain),
		Entry("given one infoSource entry",
			"domain", "guest-agent", "domain, guest-agent"),
		Entry("given two infoSource entries",
			"domain, guest-agent", "multus-status", "domain, guest-agent, multus-status"),
		Entry("given an already existing infoSource entry",
			"domain, guest-agent, multus-status", "multus-status", "domain, guest-agent, multus-status"),
	)

	DescribeTable("can remove an infoSource entry",
		func(infoSourceData, infoSourceItem, expectedInfoSourceData string) {
			Expect(vmispec.RemoveInfoSource(infoSourceData, infoSourceItem)).To(Equal(expectedInfoSourceData))
		},
		Entry("given no infoSource entries",
			emptySourceInfo, vmispec.InfoSourceDomain, emptySourceInfo),
		Entry("given one infoSource entry",
			vmispec.InfoSourceDomain, vmispec.InfoSourceDomain, emptySourceInfo),
		Entry("given two infoSource entries",
			"domain, guest-agent", "domain", "guest-agent"),
		Entry("given different infoSource entries",
			"domain, guest-agent, multus-status", "foo", "domain, guest-agent, multus-status"),
	)
})
