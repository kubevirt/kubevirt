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
 * Copyright the KubeVirt Authors.
 */

package virt_controller

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	ioprometheusclient "github.com/prometheus/client_model/go"
)

var (
	leaderMetrics = []operatormetrics.Metric{
		outdatedVirtualMachineInstanceWorkloads,
	}

	outdatedVirtualMachineInstanceWorkloads = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_number_of_outdated",
			Help: "Indication for the total number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment.",
		},
	)
)

func SetOutdatedVirtualMachineInstanceWorkloads(value int) {
	outdatedVirtualMachineInstanceWorkloads.Set(float64(value))
}

func GetOutdatedVirtualMachineInstanceWorkloads() (int, error) {
	dto := &ioprometheusclient.Metric{}
	if err := outdatedVirtualMachineInstanceWorkloads.Write(dto); err != nil {
		return 0, err
	}

	return int(dto.GetGauge().GetValue()), nil
}
