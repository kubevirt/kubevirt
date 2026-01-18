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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("IOThreads Configurator", func() {
	DescribeTable("Should", func(totalThreads uint, expectedDomain api.Domain) {
		vmi := libvmi.New()
		var domain api.Domain
		configurator := compute.NewIOThreadsDomainConfigurator(totalThreads)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("configure IOThreads in the domain when they're specified", uint(4),
			api.Domain{
				Spec: api.DomainSpec{
					IOThreads: &api.IOThreads{IOThreads: 4},
				},
			},
		),
		Entry("not configure IOThreads in the domain when they're not specified", uint(0), api.Domain{}),
	)
})
