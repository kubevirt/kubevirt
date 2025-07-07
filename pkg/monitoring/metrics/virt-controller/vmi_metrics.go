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

	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

var (
	vmiMetrics = []operatormetrics.Metric{
		vmiLauncherMemoryOverhead,
	}

	vmiLauncherMemoryOverhead = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_launcher_memory_overhead_bytes",
			Help: "Estimation of the memory amount required for virt-launcher's infrastructure components (e.g. libvirt, QEMU).",
		},
		[]string{"namespace", "name"},
	)
)

func SetVmiLaucherMemoryOverhead(vmi *v1.VirtualMachineInstance, memoryOverhead resource.Quantity) {
	vmiLauncherMemoryOverhead.
		WithLabelValues(vmi.Namespace, vmi.Name).
		Set(float64(memoryOverhead.Value()))
}

func GetVmiLaucherMemoryOverhead(vmi *v1.VirtualMachineInstance) (float64, error) {
	dto := &ioprometheusclient.Metric{}
	if err := vmiLauncherMemoryOverhead.WithLabelValues(vmi.Namespace, vmi.Name).Write(dto); err != nil {
		return -1, err
	}

	return *dto.Gauge.Value, nil
}
