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

var _ = Describe("HostDevice Domain Configurator", func() {
	It("Should handle empty variadic arguments", func() {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewHostDeviceDomainConfigurator()
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should preserve the order of HostDevices: generic, gpu, sriov", func() {
		vmi := libvmi.New()
		var domain api.Domain

		genericHostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("hostdevice-generic0"), Type: api.HostDevicePCI}
		gpuHostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("gpu-gpu0"), Type: api.HostDeviceMDev}
		sriovDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("sriov-sriov0"), Type: api.HostDevicePCI}

		configurator := compute.NewHostDeviceDomainConfigurator(
			[]api.HostDevice{genericHostDevice, gpuHostDevice, sriovDevice},
		)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					HostDevices: []api.HostDevice{
						genericHostDevice,
						gpuHostDevice,
						sriovDevice,
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("Should append to existing HostDevices preserving order", func() {
		vmi := libvmi.New()
		existingDevice := api.HostDevice{
			Alias: api.NewUserDefinedAlias("existing-device"),
			Type:  api.HostDevicePCI,
		}
		domain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					HostDevices: []api.HostDevice{existingDevice},
				},
			},
		}

		genericHostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("hostdevice-generic0"), Type: api.HostDevicePCI}
		gpuHostDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("gpu-gpu0"), Type: api.HostDeviceMDev}
		sriovDevice := api.HostDevice{Alias: api.NewUserDefinedAlias("sriov-sriov0"), Type: api.HostDevicePCI}

		configurator := compute.NewHostDeviceDomainConfigurator(
			[]api.HostDevice{genericHostDevice, gpuHostDevice, sriovDevice},
		)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					HostDevices: []api.HostDevice{
						existingDevice,
						genericHostDevice,
						gpuHostDevice,
						sriovDevice,
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})
