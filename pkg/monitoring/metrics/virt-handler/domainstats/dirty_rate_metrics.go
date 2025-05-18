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

package domainstats

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"kubevirt.io/client-go/log"
)

var (
	dirtyRateBytesPerSecond = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_dirty_rate_bytes_per_second",
			Help: "Guest dirty-rate in bytes per second.",
		},
	)
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
