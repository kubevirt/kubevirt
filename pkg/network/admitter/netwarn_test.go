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

var _ = Describe("WarnPortsOnNonPodNetwork", func() {
	It("should not warn when no ports or portRanges are set", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default"}}
		spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

		Expect(admitter.WarnPortsOnNonPodNetwork(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})

	It("should not warn when ports are set on a pod network interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:  "default",
			Ports: []v1.Port{{Port: 80}},
		}}
		spec.Networks = []v1.Network{{Name: "default", NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}}

		Expect(admitter.WarnPortsOnNonPodNetwork(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})

	It("should warn when ports are set on a non-pod network interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:  "default",
			Ports: []v1.Port{{Port: 80}},
		}}
		spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test-nad"}},
		}}

		Expect(admitter.WarnPortsOnNonPodNetwork(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.domain.devices.interfaces[0]"),
		))
	})

	It("should warn when portRanges are set on a non-pod network interface", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:       "default",
			PortRanges: []v1.PortRange{{Protocol: "TCP", Start: 80, End: 90}},
		}}
		spec.Networks = []v1.Network{{
			Name:          "default",
			NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: "test-nad"}},
		}}

		Expect(admitter.WarnPortsOnNonPodNetwork(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.domain.devices.interfaces[0]"),
		))
	})

	It("should warn when ports are set on an interface with no matching network", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:  "orphan",
			Ports: []v1.Port{{Port: 80}},
		}}

		Expect(admitter.WarnPortsOnNonPodNetwork(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.domain.devices.interfaces[0]"),
		))
	})
})

var _ = Describe("WarnTooManyPorts", func() {
	It("should not warn when the number of ports is within the limit", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:  "default",
			Ports: make([]v1.Port, 32),
		}}

		Expect(admitter.WarnTooManyPorts(k8sfield.NewPath("spec"), spec)).To(BeEmpty())
	})

	It("should warn when the number of ports exceeds the limit", func() {
		spec := &v1.VirtualMachineInstanceSpec{}
		spec.Domain.Devices.Interfaces = []v1.Interface{{
			Name:  "default",
			Ports: make([]v1.Port, 33),
		}}

		Expect(admitter.WarnTooManyPorts(k8sfield.NewPath("spec"), spec)).To(ConsistOf(
			ContainSubstring("spec.domain.devices.interfaces[0]"),
		))
	})
})
