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
		func(autoattach *bool, arch string, expectSerial bool) {
			vmi := libvmi.New(libvmi.WithUID(uid))
			vmi.Spec.Domain.Devices.AutoattachSerialConsole = autoattach

			var domain api.Domain
			Expect(compute.NewConsoleDomainConfigurator(false, arch).Configure(vmi, &domain)).To(Succeed())

			if arch == "s390x" {
				// On s390x: only <console> with sclp target, no <serial>
				sclpType := "sclp"
				expectedDomain := api.Domain{
					Spec: api.DomainSpec{
						Devices: api.Devices{
							Consoles: []api.Console{
								{
									Type: "unix",
									Source: &api.ConsoleSource{
										Mode: "bind",
										Path: socketPath,
									},
									Target: &api.ConsoleTarget{
										Type: &sclpType,
										Port: &serialPort,
									},
								},
							},
						},
					},
				}
				Expect(domain).To(Equal(expectedDomain))
				return
			}

			// Non-s390x: <console type="pty"> + <serial type="unix">
			var consoles []api.Console
			var serials []api.Serial
			if expectSerial {
				consoles = []api.Console{
					{
						Type: "pty",
						Target: &api.ConsoleTarget{
							Type: &serialType,
							Port: &serialPort,
						},
					},
				}
				serials = []api.Serial{
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
				}
			}

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					Devices: api.Devices{
						Consoles: consoles,
						Serials:  serials,
					},
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("when AutoattachSerialConsole is nil on amd64", nil, "amd64", true),
		Entry("when AutoattachSerialConsole is true on amd64", pointer.P(true), "amd64", true),
		Entry("when AutoattachSerialConsole is nil on arm64", nil, "arm64", true),
		Entry("when AutoattachSerialConsole is nil on s390x", nil, "s390x", true),
		Entry("when AutoattachSerialConsole is true on s390x", pointer.P(true), "s390x", true),
	)

	It("should NOT configure serial console when AutoattachSerialConsole is explicitly false", func() {
		vmi := libvmi.New(libvmi.WithAutoattachSerialConsole(false))
		var domain api.Domain

		Expect(compute.NewConsoleDomainConfigurator(false, "amd64").Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("should configure console with serial log", func() {
		vmi := libvmi.New(libvmi.WithUID(uid))

		var domain api.Domain
		configurator := compute.NewConsoleDomainConfigurator(true, "amd64")
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

	It("should configure console-only with sclp target and log on s390x (no serial device)", func() {
		vmi := libvmi.New(libvmi.WithUID(uid))

		var domain api.Domain
		configurator := compute.NewConsoleDomainConfigurator(true, "s390x")
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		sclpType := "sclp"
		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					Consoles: []api.Console{
						{
							Type: "unix",
							Source: &api.ConsoleSource{
								Mode: "bind",
								Path: socketPath,
							},
							Target: &api.ConsoleTarget{
								Type: &sclpType,
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
