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

var _ = Describe("Watchdog Domain Configurator", func() {
	DescribeTable("Should not configure watchdog when watchdog is unspecified", func(architecture string) {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewWatchdogDomainConfigurator(architecture)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	},
		Entry("amd64", "amd64"),
		Entry("arm64", "arm64"),
		Entry("s390x", "s390x"),
	)

	DescribeTable("should successfully convert watchdog for supported architectures",
		func(architecture string, input v1.Watchdog, expectedDevice api.Watchdog) {
			vmi := libvmi.New(withWatchdog(input))
			var domain api.Domain

			configurator := compute.NewWatchdogDomainConfigurator(architecture)
			Expect(configurator.Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Watchdogs: []api.Watchdog{
							expectedDevice,
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("amd64 with I6300ESB",
			"amd64",
			v1.Watchdog{
				Name: "mywatchdog",
				WatchdogDevice: v1.WatchdogDevice{
					I6300ESB: &v1.I6300ESBWatchdog{
						Action: v1.WatchdogActionPoweroff,
					},
				},
			},
			api.Watchdog{
				Alias:  api.NewUserDefinedAlias("mywatchdog"),
				Model:  "i6300esb",
				Action: "poweroff",
			},
		),
		Entry("s390x with Diag288",
			"s390x",
			v1.Watchdog{
				Name: "diagwatchdog",
				WatchdogDevice: v1.WatchdogDevice{
					Diag288: &v1.Diag288Watchdog{
						Action: v1.WatchdogActionReset,
					},
				},
			},
			api.Watchdog{
				Alias:  api.NewUserDefinedAlias("diagwatchdog"),
				Model:  "diag288",
				Action: "reset",
			},
		),
	)

	DescribeTable("should fail to convert watchdog for unsupported or invalid architectures",
		func(architecture string, input v1.Watchdog, expectedErrMsg string) {
			vmi := libvmi.New(withWatchdog(input))
			var domain api.Domain

			configurator := compute.NewWatchdogDomainConfigurator(architecture)
			err := configurator.Configure(vmi, &domain)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedErrMsg))
		},
		Entry("arm64 not supported",
			"arm64",
			v1.Watchdog{Name: "unsupportedwatchdog"},
			"not supported on architecture",
		),
		Entry("amd64 with no watchdog type",
			"amd64",
			v1.Watchdog{Name: "emptywatchdog"},
			"can't be mapped",
		),
		Entry("s390x with nil Diag288",
			"s390x",
			v1.Watchdog{Name: "diagwatchdog"},
			"can't be mapped",
		),
	)
})

func withWatchdog(watchdog v1.Watchdog) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.Watchdog = &watchdog
	}
}
