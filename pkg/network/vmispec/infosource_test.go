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
 * Copyright 2021 Red Hat, Inc.
 *
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
