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

var _ = Describe("IOMMU Domain Configurator", func() {
	It("Should not configure IOMMU when IOMMU is nil", func() {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.NewIOMMUDomainConfigurator("arm64").Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should configure IOMMU with default model on arm64", func() {
		vmi := libvmi.New(
			libvmi.WithIOMMU(&v1.IOMMUDevice{}),
		)
		var domain api.Domain

		Expect(compute.NewIOMMUDomainConfigurator("arm64").Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					IOMMU: &api.IOMMU{
						Model: "smmuv3",
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("Should configure IOMMU with explicit model", func() {
		vmi := libvmi.New(
			libvmi.WithIOMMU(&v1.IOMMUDevice{
				Model: "smmuv3",
			}),
		)
		var domain api.Domain

		Expect(compute.NewIOMMUDomainConfigurator("arm64").Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					IOMMU: &api.IOMMU{
						Model: "smmuv3",
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("Should configure IOMMU with pciBus driver option", func() {
		vmi := libvmi.New(
			libvmi.WithIOMMU(&v1.IOMMUDevice{
				Model: "smmuv3",
				Driver: &v1.IOMMUDeviceDriver{
					PCIBus: pointer.P(uint(1)),
				},
			}),
		)
		var domain api.Domain

		Expect(compute.NewIOMMUDomainConfigurator("arm64").Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					IOMMU: &api.IOMMU{
						Model: "smmuv3",
						Driver: &api.IOMMUDriver{
							PCIBus: pointer.P(uint(1)),
						},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("Should fail with smmuv3 on non-arm64 architecture", func() {
		vmi := libvmi.New(
			libvmi.WithIOMMU(&v1.IOMMUDevice{
				Model: "smmuv3",
			}),
		)
		var domain api.Domain

		err := compute.NewIOMMUDomainConfigurator("amd64").Configure(vmi, &domain)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("smmuv3 is only supported on arm64"))
	})

	It("Should fail with default model on non-arm64 architecture", func() {
		vmi := libvmi.New(
			libvmi.WithIOMMU(&v1.IOMMUDevice{}),
		)
		var domain api.Domain

		err := compute.NewIOMMUDomainConfigurator("amd64").Configure(vmi, &domain)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("smmuv3 is only supported on arm64"))
	})
})
