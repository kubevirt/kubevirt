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

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Clock Domain Configurator", func() {
	It("Should not set clock domain attribute when clock is unspecified on the VMI", func() {
		vmi := libvmi.New()

		var domain api.Domain

		Expect(compute.ClockDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should set timezone attribute when timezone is specified on the VMI", func() {
		const expectedTimezone = "America/New_York"
		clock := v1.Clock{
			ClockOffset: v1.ClockOffset{
				Timezone: pointer.P(v1.ClockOffsetTimezone(expectedTimezone)),
			},
			Timer: &v1.Timer{},
		}
		vmi := libvmi.New(libvmi.WithClock(clock))

		var domain api.Domain

		Expect(compute.ClockDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Clock: &api.Clock{
					Offset:     "timezone",
					Timezone:   expectedTimezone,
					Adjustment: "",
					Timer:      nil,
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})
