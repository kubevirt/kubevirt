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

var _ = Describe("PPC64LE Machine Domain Configurator", func() {
	It("should disable the large decrementer on ppc64le", func() {
		var domain api.Domain

		configurator := compute.PPC64LEMachineDomainConfigurator{}
		Expect(configurator.Configure(libvmi.New(), &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithQEMUCmd(&api.Commandline{
			QEMUArg: []api.Arg{
				{Value: "-machine"},
				{Value: "cap-large-decr=off"},
			},
		})))
	})

	It("should keep existing QEMU arguments on ppc64le", func() {
		domain := newDomainWithQEMUCmd(&api.Commandline{
			QEMUArg: []api.Arg{{Value: "-fw_cfg"}, {Value: "name=test"}},
		})

		configurator := compute.PPC64LEMachineDomainConfigurator{}
		Expect(configurator.Configure(libvmi.New(), &domain)).To(Succeed())

		Expect(domain.Spec.QEMUCmd.QEMUArg).To(Equal([]api.Arg{
			{Value: "-fw_cfg"},
			{Value: "name=test"},
			{Value: "-machine"},
			{Value: "cap-large-decr=off"},
		}))
	})
})
