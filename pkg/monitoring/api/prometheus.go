package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	namespaceAndVMILabels    = []string{"namespace", "vmi"}
	activePortForwardTunnels = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_portforward_active_tunnels",
			Help: "Amount of active portforward tunnels, broken down by namespace and vmi name",
		},
		namespaceAndVMILabels,
	)
	activeVNCConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_vnc_active_connections",
			Help: "Amount of active VNC connections, broken down by namespace and vmi name",
		},
		namespaceAndVMILabels,
	)
	activeConsoleConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_console_active_connections",
			Help: "Amount of active Console connections, broken down by namespace and vmi name",
		},
		namespaceAndVMILabels,
	)
	activeUSBRedirConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubevirt_usbredir_active_connections",
			Help: "Amount of active USB redirection connections, broken down by namespace and vmi name",
		},
		namespaceAndVMILabels,
	)
)

func init() {
	prometheus.MustRegister(activePortForwardTunnels)
	prometheus.MustRegister(activeVNCConnections)
	prometheus.MustRegister(activeConsoleConnections)
	prometheus.MustRegister(activeUSBRedirConnections)
}

type Decrementer interface {
	Dec()
}

// NewActivePortForwardTunnel increments the metric for active portforward tunnels by one for namespace and name
// and returns a recorder for decrementing it once the tunnel is closed
func NewActivePortForwardTunnel(namespace, name string) Decrementer {
	recorder := activePortForwardTunnels.WithLabelValues(namespace, name)
	recorder.Inc()
	return recorder
}

// NewActiveVNCConnection increments the metric for active VNC connections by one for namespace and name
// and returns a recorder for decrementing it once the connection is closed
func NewActiveVNCConnection(namespace, name string) Decrementer {
	recorder := activeVNCConnections.WithLabelValues(namespace, name)
	recorder.Inc()
	return recorder
}

// NewActiveConsoleConnection increments the metric for active console sessions by one for namespace and name
// and returns a recorder for decrementing it once the connection is closed
func NewActiveConsoleConnection(namespace, name string) Decrementer {
	recorder := activeConsoleConnections.WithLabelValues(namespace, name)
	recorder.Inc()
	return recorder
}

// NewActiveUSBRedirConnection increments the metric for active USB redirection connections by one for namespace
// and name and returns a recorder for decrementing it once the connection is closed
func NewActiveUSBRedirConnection(namespace, name string) Decrementer {
	recorder := activeUSBRedirConnections.WithLabelValues(namespace, name)
	recorder.Inc()
	return recorder
}
