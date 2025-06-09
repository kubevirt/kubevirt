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

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	filesystemCapacityBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_filesystem_capacity_bytes",
			Help: "Total VM filesystem capacity in bytes.",
		},
	)

	filesystemUsedBytes = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_filesystem_used_bytes",
			Help: "Used VM filesystem capacity in bytes.",
		},
	)
)

type filesystemMetrics struct{}

func (filesystemMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		filesystemCapacityBytes,
		filesystemUsedBytes,
	}
}

func (filesystemMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	for _, fsStat := range vmiReport.vmiStats.FsStats.Items {
		fsLabels := map[string]string{
			"disk_name":        fsStat.DiskName,
			"mount_point":      fsStat.MountPoint,
			"file_system_type": fsStat.FileSystemType,
		}

		crs = append(crs, vmiReport.newCollectorResultWithLabels(filesystemCapacityBytes, float64(fsStat.TotalBytes), fsLabels))
		crs = append(crs, vmiReport.newCollectorResultWithLabels(filesystemUsedBytes, float64(fsStat.UsedBytes), fsLabels))
	}

	return crs
}
