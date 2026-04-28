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

var _ = Describe("RebootPolicy Domain Configurator", func() {
	It("Should not set OnReboot when RebootPolicy is unspecified in VMI", func() {
		vmi := libvmi.New()
		var domain api.Domain

		Expect(compute.RebootPolicyDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	DescribeTable("Should set OnReboot when RebootPolicy is specified in VMI",
		func(policy v1.RebootPolicy, expectedOnReboot string) {
			vmi := libvmi.New(withRebootPolicy(policy))
			var domain api.Domain

			Expect(compute.RebootPolicyDomainConfigurator{}.Configure(vmi, &domain)).To(Succeed())
			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					OnReboot: expectedOnReboot,
				},
			}
			Expect(domain).To(Equal(expectedDomain))
		},
		Entry("Terminate policy maps to destroy",
			v1.RebootPolicyTerminate, api.DomainOnRebootDestroy,
		),
		Entry("Reboot policy maps to restart",
			v1.RebootPolicyReboot, api.DomainOnRebootRestart,
		),
	)
})

func withRebootPolicy(policy v1.RebootPolicy) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.RebootPolicy = &policy
	}
}
