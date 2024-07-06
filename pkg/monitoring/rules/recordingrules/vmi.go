/*
Copyright 2023 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package recordingrules

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var vmiRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_phase_count",
			Help: "Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`].",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("sum by (node, phase, os, workload, flavor, instance_type, preference, guest_os_kernel_release, guest_os_machine, guest_os_name, guest_os_version_id) (kubevirt_vmi_info)"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_memory_used_bytes",
			Help: "Amount of `used` memory as seen by the domain.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr:       intstr.FromString("kubevirt_vmi_memory_available_bytes-kubevirt_vmi_memory_usable_bytes"),
	},
}
