/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package domainstats

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

var dirtyRateBytesPerSecond = operatormetrics.NewGauge(
	operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_dirty_rate_bytes_per_second",
		Help: "Guest dirty-rate in bytes per second.",
	},
)

type dirtyRateMetrics struct{}

func (dirtyRateMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		dirtyRateBytesPerSecond,
	}
}

func (dirtyRateMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.DomainStats == nil || vmiReport.vmiStats.DomainStats.DirtyRate == nil {
		return crs
	}

	dirtyRate := vmiReport.vmiStats.DomainStats.DirtyRate

	if !dirtyRate.MegabytesPerSecondSet {
		log.Log.Warningf("No domain dirty rate stats is set for VMI %s/%s", vmiReport.vmi.Namespace, vmiReport.vmi.Name)
		return crs
	}

	dirtyRateInBytesPerSecond := dirtyRate.MegabytesPerSecond * 1024 * 1024
	crs = append(crs, vmiReport.newCollectorResult(dirtyRateBytesPerSecond, float64(dirtyRateInBytesPerSecond)))

	return crs
}
