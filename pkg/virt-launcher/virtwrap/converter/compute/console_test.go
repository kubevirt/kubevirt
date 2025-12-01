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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("Console Domain Configurator", func() {

	const uid = "test-uid"
	serialPort := uint(0)
	socketPath := fmt.Sprintf("%s/%s/virt-serial%d", util.VirtPrivateDir, uid, serialPort)
	serialType := "serial"

	DescribeTable("should configure serial console when AutoattachSerialConsole is not disabled",
		func(autoattach *bool) {
			vmi := libvmi.New(libvmi.WithUID(uid))
			vmi.Spec.Domain.Devices.AutoattachSerialConsole = autoattach

			var domain api.Domain
			Expect(compute.NewConsoleDomainConfigurator(false).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Consoles: []api.Console{
							{
								Type: "pty",
								Target: &api.ConsoleTarget{
									Type: &serialType,
									Port: &serialPort,
								},
							},
						},
						Serials: []api.Serial{
							{
								Type: "unix",
								Source: &api.SerialSource{
									Mode: "bind",
									Path: socketPath,
								},
								Target: &api.SerialTarget{
									Port: &serialPort,
								},
							},
						},
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("when AutoattachSerialConsole is nil", nil),
		Entry("when AutoattachSerialConsole is true", pointer.P(true)),
	)

	It("should NOT configure serial console when AutoattachSerialConsole is explicitly false", func() {
		vmi := libvmi.New(withAutoattachSerialConsole(false))
		var domain api.Domain

		Expect(compute.NewConsoleDomainConfigurator(false).Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("should configure console with serial log", func() {
		vmi := libvmi.New(libvmi.WithUID(uid))

		var domain api.Domain
		configurator := compute.NewConsoleDomainConfigurator(true)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Consoles: []api.Console{
						{
							Type: "pty",
							Target: &api.ConsoleTarget{
								Type: &serialType,
								Port: &serialPort,
							},
						},
					},
					Serials: []api.Serial{
						{
							Type: "unix",
							Source: &api.SerialSource{
								Mode: "bind",
								Path: socketPath,
							},
							Target: &api.SerialTarget{
								Port: &serialPort,
							},
							Log: &api.SerialLog{
								File:   socketPath + "-log",
								Append: "on",
							},
						},
					},
				},
			},
		}

		Expect(domain).To(Equal(expectedDomain))
	})
})

func withAutoattachSerialConsole(enabled bool) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.AutoattachSerialConsole = pointer.P(enabled)
	}
}
