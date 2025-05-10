package virt_handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"libvirt.org/go/libvirtxml"
)

var _ = Describe("supported machine types metric", func() {
	Context("ReportSupportedMachineTypes", func() {
		BeforeEach(func() {
			operatormetrics.UnregisterMetrics(machineTypeMetrics)
			Expect(operatormetrics.RegisterMetrics(machineTypeMetrics)).To(Succeed())
		})

		It("should initialize the metric with correct labels for multiple machine types", func() {
			nodeName := "testnode"
			machineTypes := []libvirtxml.CapsGuestMachine{
				{Name: "machine1", Deprecated: "false"},
				{Name: "machine2", Deprecated: "true"},
			}

			ReportSupportedMachineTypes(nodeName, machineTypes)

			Expect(machineTypeMetrics).ToNot(BeEmpty())
			Expect(supportedMachineTypeMetric).ToNot(BeNil(), "supportedMachineTypeMetric should be initialized")

			for _, machine := range machineTypes {
				labels := map[string]string{
					"node":         nodeName,
					"machine_type": machine.Name,
					"deprecated":   machine.Deprecated,
				}
				_, err := supportedMachineTypeMetric.GetMetricWith(labels)
				Expect(err).ToNot(HaveOccurred(), "should initialize metric with labels %v", labels)
			}
		})
	})
})
