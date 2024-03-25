package domainstats

import "github.com/machadovilaca/operator-observability/pkg/operatormetrics"

var (
	networkTrafficBytesDeprecated = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_traffic_bytes_total",
			Help: "[Deprecated] Total number of bytes sent and received.",
		},
	)

	networkReceiveBytes = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_receive_bytes_total",
			Help: "Total network traffic received in bytes.",
		},
	)

	networkTransmitBytes = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_transmit_bytes_total",
			Help: "Total network traffic transmitted in bytes.",
		},
	)

	networkReceivePackets = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_receive_packets_total",
			Help: "Total network traffic received packets.",
		},
	)

	networkTransmitPackets = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_transmit_packets_total",
			Help: "Total network traffic transmitted packets.",
		},
	)

	networkReceiveErrors = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_receive_errors_total",
			Help: "Total network received error packets.",
		},
	)

	networkTransmitErrors = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_transmit_errors_total",
			Help: "Total network transmitted error packets.",
		},
	)

	networkReceivePacketsDropped = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_receive_packets_dropped_total",
			Help: "The total number of rx packets dropped on vNIC interfaces.",
		},
	)

	networkTransmitPacketsDropped = operatormetrics.NewCounter(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_network_transmit_packets_dropped_total",
			Help: "The total number of tx packets dropped on vNIC interfaces.",
		},
	)
)

type networkMetrics struct{}

func (networkMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		networkTrafficBytesDeprecated,
		networkReceiveBytes,
		networkTransmitBytes,
		networkReceivePackets,
		networkTransmitPackets,
		networkReceiveErrors,
		networkTransmitErrors,
		networkReceivePacketsDropped,
		networkTransmitPacketsDropped,
	}
}

func (networkMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.Net == nil {
		return crs
	}

	for _, net := range vmiReport.vmiStats.DomainStats.Net {
		if !net.NameSet {
			continue
		}

		iface := net.Name
		if net.AliasSet {
			iface = net.Alias
		}
		netLabels := map[string]string{"interface": iface}

		if net.RxBytesSet {
			deprecatedLabels := map[string]string{"interface": iface, "type": "rx"}
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkTrafficBytesDeprecated, float64(net.RxBytes), deprecatedLabels))
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkReceiveBytes, float64(net.RxBytes), netLabels))
		}

		if net.TxBytesSet {
			deprecatedLabels := map[string]string{"interface": iface, "type": "tx"}
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkTrafficBytesDeprecated, float64(net.TxBytes), deprecatedLabels))
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkTransmitBytes, float64(net.TxBytes), netLabels))
		}

		if net.RxPktsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkReceivePackets, float64(net.RxPkts), netLabels))
		}

		if net.TxPktsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkTransmitPackets, float64(net.TxPkts), netLabels))
		}

		if net.RxErrsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkReceiveErrors, float64(net.RxErrs), netLabels))
		}

		if net.TxErrsSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkTransmitErrors, float64(net.TxErrs), netLabels))
		}

		if net.RxDropSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkReceivePacketsDropped, float64(net.RxDrop), netLabels))
		}

		if net.TxDropSet {
			crs = append(crs, vmiReport.newCollectorResultWithLabels(networkTransmitPacketsDropped, float64(net.TxDrop), netLabels))
		}
	}

	return crs
}
