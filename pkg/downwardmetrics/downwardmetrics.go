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

package downwardmetrics

import (
	"path/filepath"
	"strconv"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	DownwardMetricsSerialDeviceName = "org.github.vhostmd.1"
	DownwardMetricsChannelDir       = util.VirtPrivateDir + "/downwardmetrics-channel"
	DownwardMetricsChannelSocket    = DownwardMetricsChannelDir + "/downwardmetrics.sock"
)

func CreateDownwardMetricDisk(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.DownwardMetrics != nil {
			return vhostmd.NewMetricsIODisk(config.DownwardMetricDisk).Create()
		}
	}
	return nil
}

func FormatDownwardMetricPath(pid int) string {
	return filepath.Join("/proc", strconv.Itoa(pid), "/root", config.DownwardMetricDisk)
}

func HasDownwardMetricDisk(vmi *v1.VirtualMachineInstance) bool {
	for _, volume := range vmi.Spec.Volumes {
		if volume.DownwardMetrics != nil {
			return true
		}
	}
	return false
}

func HasDevice(spec *v1.VirtualMachineInstanceSpec) bool {
	return spec.Domain.Devices.DownwardMetrics != nil
}

func ChannelSocketPathOnHost(pid int) string {
	return filepath.Join("/proc", strconv.Itoa(pid), "root", DownwardMetricsChannelSocket)
}
