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

package virt_api

import (
	"time"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
)

var (
	connectionMetrics = []operatormetrics.Metric{
		activePortForwardTunnels,
		activeVNCConnections,
		activeConsoleConnections,
		activeUSBRedirConnections,
		vmiLastConnectionTimestamp,
	}

	namespaceAndVMILabels = []string{"namespace", "vmi"}

	activePortForwardTunnels = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_portforward_active_tunnels",
			Help: "Amount of active portforward tunnels, broken down by namespace and vmi name.",
		},
		namespaceAndVMILabels,
	)

	activeVNCConnections = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vnc_active_connections",
			Help: "Amount of active VNC connections, broken down by namespace and vmi name.",
		},
		namespaceAndVMILabels,
	)

	activeConsoleConnections = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_console_active_connections",
			Help: "Amount of active Console connections, broken down by namespace and vmi name.",
		},
		namespaceAndVMILabels,
	)

	activeUSBRedirConnections = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_usbredir_active_connections",
			Help: "Amount of active USB redirection connections, broken down by namespace and vmi name.",
		},
		namespaceAndVMILabels,
	)

	vmiLastConnectionTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_last_api_connection_timestamp_seconds",
			Help: "Virtual Machine Instance last API connection timestamp. Including VNC, console, portforward, SSH and usbredir connections.",
		},
		namespaceAndVMILabels,
	)
)

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

func SetVMILastConnectionTimestamp(namespace, name string) {
	vmiLastConnectionTimestamp.WithLabelValues(namespace, name).Set(float64(time.Now().Unix()))
}
