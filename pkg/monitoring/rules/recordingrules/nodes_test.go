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
 */

package recordingrules

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Nodes recording rules", func() {
	It("should generate generic hypervisor metric with KVM resource when hypervisor is KVM", func() {
		rules := nodesRecordingRules(v1.KvmHypervisorName)

		Expect(rules).To(HaveLen(3))
		Expect(rules[0].MetricsOpts.Name).To(Equal("kubevirt_allocatable_nodes"))
		Expect(rules[1].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_hypervisor"))
		Expect(rules[1].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_kvm"))
		Expect(rules[2].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_kvm"))
		Expect(rules[2].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_kvm"))
		Expect(rules[2].MetricsOpts.ConstLabels["deprecated"]).To(Equal("true"))
	})

	It("should generate generic hypervisor metric with MSHV resource when hypervisor is HyperV", func() {
		rules := nodesRecordingRules(v1.HyperVLayeredHypervisorName)

		Expect(rules).To(HaveLen(2))
		Expect(rules[0].MetricsOpts.Name).To(Equal("kubevirt_allocatable_nodes"))
		Expect(rules[1].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_hypervisor"))
		Expect(rules[1].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_mshv"))
	})

	It("should default to KVM resource when hypervisor is empty", func() {
		rules := nodesRecordingRules("")

		Expect(rules).To(HaveLen(3))
		Expect(rules[0].MetricsOpts.Name).To(Equal("kubevirt_allocatable_nodes"))
		Expect(rules[1].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_hypervisor"))
		Expect(rules[1].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_kvm"))
		Expect(rules[2].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_kvm"))
		Expect(rules[2].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_kvm"))
		Expect(rules[2].MetricsOpts.ConstLabels["deprecated"]).To(Equal("true"))
	})

	It("should default to KVM resource when hypervisor is unknown", func() {
		rules := nodesRecordingRules("unknown-hypervisor")

		Expect(rules).To(HaveLen(3))
		Expect(rules[0].MetricsOpts.Name).To(Equal("kubevirt_allocatable_nodes"))
		Expect(rules[1].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_hypervisor"))
		Expect(rules[1].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_kvm"))
		Expect(rules[2].MetricsOpts.Name).To(Equal("kubevirt_nodes_with_kvm"))
		Expect(rules[2].Expr.StrVal).To(ContainSubstring("devices_kubevirt_io_kvm"))
		Expect(rules[2].MetricsOpts.ConstLabels["deprecated"]).To(Equal("true"))
	})
})
