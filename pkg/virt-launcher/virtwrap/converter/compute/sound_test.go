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

var _ = Describe("Sound Domain Configurator", func() {
	const deviceName = "sound-device"
	It("Should not configure a sound device when sound is unspecified in VMI", func() {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.SoundDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	DescribeTable("Should configure a sound device when sound is specified in VMI",
		func(inputDevice v1.SoundDevice, expectedDevice api.SoundCard) {
			vmi := libvmi.New(withSound(inputDevice))
			var domain api.Domain

			Expect(compute.SoundDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						SoundCards: []api.SoundCard{
							expectedDevice,
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("when only name is specified",
			v1.SoundDevice{Name: deviceName},
			api.SoundCard{Alias: api.NewUserDefinedAlias(deviceName), Model: "ich9"},
		),
		Entry("when name and ich9 model are specified",
			v1.SoundDevice{Name: deviceName, Model: "ich9"},
			api.SoundCard{Alias: api.NewUserDefinedAlias(deviceName), Model: "ich9"},
		),
		Entry("when name and arbitrary model are specified",
			v1.SoundDevice{Name: deviceName, Model: "arbitraryModelName"},
			api.SoundCard{Alias: api.NewUserDefinedAlias(deviceName), Model: "ich9"},
		),
		Entry("when name and ac97 model are specified",
			v1.SoundDevice{Name: deviceName, Model: "ac97"},
			api.SoundCard{Alias: api.NewUserDefinedAlias(deviceName), Model: "ac97"},
		),
	)
})

func withSound(sound v1.SoundDevice) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Sound = &sound
	}
}
