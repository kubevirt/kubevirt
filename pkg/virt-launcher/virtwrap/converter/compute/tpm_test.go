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

var _ = Describe("TPM Domain Configurator", func() {
	It("Should not configure a TPM device when TPM is unspecified in VMI", func() {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.TPMDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should configure a TPM device when non-persistent TPM is specified in VMI", func() {
		vmi := libvmi.New(libvmi.WithTPM(false))
		var domain api.Domain

		Expect(compute.TPMDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					TPMs: []api.TPM{
						{
							Model: "tpm-tis",
							Backend: api.TPMBackend{
								Type:    "emulator",
								Version: "2.0",
							},
						},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("Should configure a TPM device when persistent TPM is specified in VMI", func() {
		vmi := libvmi.New(libvmi.WithTPM(true))
		var domain api.Domain

		Expect(compute.TPMDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					TPMs: []api.TPM{
						{
							Model: "tpm-crb",
							Backend: api.TPMBackend{
								Type:            "emulator",
								Version:         "2.0",
								PersistentState: "yes",
							},
						},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})
