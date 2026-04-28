/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package virtoperator

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	leaderMetrics = []operatormetrics.Metric{
		emulationEnabled,
	}

	emulationEnabled = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_configuration_emulation_enabled",
			Help: "Indicates whether the Software Emulation is enabled in the configuration.",
		},
	)
)

func SetEmulationEnabledMetric(isEmulationEnabled bool) {
	emulationEnabled.Set(boolToFloat64(isEmulationEnabled))
}
