/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virthandler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"libvirt.org/go/libvirtxml"
)

var _ = Describe("deprecated machine types metric", func() {
	Context("ReportDeprecatedMachineTypes", func() {
		BeforeEach(func() {
			Expect(operatormetrics.UnregisterMetrics(machineTypeMetrics)).To(Succeed())
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
