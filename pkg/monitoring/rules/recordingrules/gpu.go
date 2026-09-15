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

package recordingrules

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var gpuRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_gpu_mem_copy_util:sum",
			Help: "Memory utilization (ratio, 0-1) of the GPU passed through to a virtual machine instance.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"sum by(namespace, name, node, uuid, resource) (" +
				"label_replace(max by (UUID) (DCGM_FI_DEV_MEM_COPY_UTIL), 'uuid', '$1', 'UUID', '(.*)') * " +
				"on(uuid) group_left(namespace, name, node, resource) kubevirt_vmi_gpu_info) / 100"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_gpu_fb_free:sum",
			Help: "Framebuffer memory free (in bytes) of the GPU passed through to a virtual machine instance.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"sum by(namespace, name, node, uuid, resource) (" +
				"label_replace(max by (UUID) (DCGM_FI_DEV_FB_FREE), 'uuid', '$1', 'UUID', '(.*)') * " +
				"on(uuid) group_left(namespace, name, node, resource) kubevirt_vmi_gpu_info) * 1024 * 1024"),
	},
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name: "vmi:kubevirt_vmi_gpu_fb_used:sum",
			Help: "Framebuffer memory used (in bytes) of the GPU passed through to a virtual machine instance.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"sum by(namespace, name, node, uuid, resource) (" +
				"label_replace(max by (UUID) (DCGM_FI_DEV_FB_USED), 'uuid', '$1', 'UUID', '(.*)') * " +
				"on(uuid) group_left(namespace, name, node, resource) kubevirt_vmi_gpu_info) * 1024 * 1024"),
	},
}
