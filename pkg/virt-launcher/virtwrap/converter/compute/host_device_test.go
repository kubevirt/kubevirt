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
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("HostDevice Domain Configurator", func() {
	It("Should handle empty variadic arguments", func() {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewHostDeviceDomainConfigurator()
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	DescribeTable("should preserve the order of input devices", func(existing, input []api.HostDevice) {
		vmi := libvmi.New()

		domain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					HostDevices: existing,
				},
			},
		}

		configurator := compute.NewHostDeviceDomainConfigurator(input)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					HostDevices: slices.Concat(existing, input),
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	},
		Entry("without existing devices",
			nil,
			[]api.HostDevice{
				newHostDevice("hostdevice-generic0", api.HostDevicePCI),
				newHostDevice("gpu-gpu0", api.HostDeviceMDev),
				newHostDevice("sriov-sriov0", api.HostDevicePCI),
			},
		),
		Entry("with existing device",
			[]api.HostDevice{
				newHostDevice("existing-device", api.HostDevicePCI),
			},
			[]api.HostDevice{
				newHostDevice("hostdevice-generic0", api.HostDevicePCI),
				newHostDevice("gpu-gpu0", api.HostDeviceMDev),
				newHostDevice("sriov-sriov0", api.HostDevicePCI),
			},
		),
	)
})

func newHostDevice(name, typeString string) api.HostDevice {
	return api.HostDevice{
		Alias: api.NewUserDefinedAlias(name),
		Type:  typeString,
	}
}
