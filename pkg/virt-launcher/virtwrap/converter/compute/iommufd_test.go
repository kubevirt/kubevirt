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

var _ = Describe("IOMMUFD Domain Configurator", func() {
	It("should set IOMMUFD on domain when enabled", func() {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewIOMMUFDConfigurator(true)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain.Spec.IOMMUFD).NotTo(BeNil())
		Expect(domain.Spec.IOMMUFD.Enabled).To(Equal("yes"))
		Expect(domain.Spec.IOMMUFD.FDGroup).To(Equal("iommu"))
	})

	It("should not set IOMMUFD on domain when disabled", func() {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewIOMMUFDConfigurator(false)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain.Spec.IOMMUFD).To(BeNil())
	})
})
