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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Panic Device Domain Configurator", func() {
	It("Should not configure panic devices when none are specified in VMI", func() {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.PanicDevicesDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should configure multiple panic devices when multiple are specified in VMI", func() {
		hypervModel := v1.Hyperv
		isaModel := v1.Isa
		pvpanicModel := v1.Pvpanic
		vmi := libvmi.New(
			libvmi.WithPanicDevice(hypervModel),
			libvmi.WithPanicDevice(isaModel),
			libvmi.WithPanicDevice(pvpanicModel),
		)
		var domain api.Domain

		Expect(compute.PanicDevicesDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					PanicDevices: []api.PanicDevice{
						{Model: &hypervModel},
						{Model: &isaModel},
						{Model: &pvpanicModel},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})
