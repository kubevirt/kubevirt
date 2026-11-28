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

package admitter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("WarnTooManyNTPServers", func() {
	It("should not warn when the number of NTP servers is within the limit", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
			DHCPOptions: &v1.DHCPOptions{
				NTPServers: make([]string, 32),
			},
		}}

		Expect(admitter.WarnTooManyNTPServers(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})

	It("should warn when the number of NTP servers exceeds the limit", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
			DHCPOptions: &v1.DHCPOptions{
				NTPServers: make([]string, 33),
			},
		}}

		Expect(admitter.WarnTooManyNTPServers(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.domain.devices.interfaces[0]"),
		))
	})

	It("should not warn when the interface has no DHCPOptions", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name: "default",
		}}

		Expect(admitter.WarnTooManyNTPServers(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})
})
