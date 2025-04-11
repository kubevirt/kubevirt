package virt_handler

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"libvirt.org/go/libvirtxml"
)

var (
	machineTypeMetrics = []operatormetrics.Metric{
		supportedMachineTypeMetric,
	}

	supportedMachineTypeMetric = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_supported_machine_types",
			Help: "List of supported machine types.",
		},
		[]string{"node", "machine_type", "deprecated"},
	)
)

func ReportSupportedMachineTypes(nodeName string, supportedMachines []libvirtxml.CapsGuestMachine) {
	for _, machine := range supportedMachines {
		supportedMachineTypeMetric.WithLabelValues(nodeName, machine.Name, machine.Deprecated).Set(1)
	}
}
