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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/admitter"
)

var _ = Describe("WarnLongNetworkNames", func() {
	It("should not warn when all network names are within the recommended length", func() {
		spec := &v1.VirtualMachineInstanceSpec{
			Networks: []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}},
		}

		Expect(admitter.WarnLongNetworkNames(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})

	It("should warn about a network name longer than 63 characters", func() {
		longName := strings.Repeat("a", 64)
		spec := &v1.VirtualMachineInstanceSpec{
			Networks: []v1.Network{{Name: longName, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}},
		}

		Expect(admitter.WarnLongNetworkNames(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.networks[0].name"),
		))
	})
})

var _ = Describe("WarnLongInterfaceNames", func() {
	It("should not warn when all interface names are within the recommended length", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default"}}

		Expect(admitter.WarnLongInterfaceNames(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})

	It("should warn about an interface name longer than 63 characters", func() {
		longName := strings.Repeat("b", 64)
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{Name: longName}}

		Expect(admitter.WarnLongInterfaceNames(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.domain.devices.interfaces[0].name"),
		))
	})
})
