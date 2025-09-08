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

package virt_handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"libvirt.org/go/libvirtxml"
)

var _ = Describe("deprecated machine types metric", func() {
	Context("ReportDeprecatedMachineTypes", func() {
		BeforeEach(func() {
			operatormetrics.UnregisterMetrics(machineTypeMetrics)
			Expect(operatormetrics.RegisterMetrics(machineTypeMetrics)).To(Succeed())
		})

		It("should initialize the metric correctly", func() {
			machineTypes := []libvirtxml.CapsGuestMachine{
				{Name: "machine1", Deprecated: "false"},
				{Name: "machine2", Deprecated: "true"},
			}

			ReportDeprecatedMachineTypes(machineTypes, "test-node")

			Expect(machineTypeMetrics).ToNot(BeEmpty())
			Expect(deprecatedMachineTypeMetric).ToNot(BeNil(), "deprecatedMachineTypeMetric should be initialized")

			for _, machine := range machineTypes {
				labels := map[string]string{
					"machine_type": machine.Name,
					"node":         "test-node",
				}
				_, err := deprecatedMachineTypeMetric.GetMetricWith(labels)
				Expect(err).ToNot(HaveOccurred(), "should initialize metric with labels %v", labels)
			}
		})
	})
})
