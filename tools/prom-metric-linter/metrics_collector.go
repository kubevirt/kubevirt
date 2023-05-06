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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	dto "github.com/prometheus/client_model/go"
)

// Metric represents a Prometheus metric
type Metric struct {
	Name string
	Help string
	Type string
}

// excludedMetrics defines the metrics to ignore, open issue:https://github.com/kubevirt/kubevirt/issues/9714
// Do not add metrics to this list!
var excludedMetrics = map[string]string{
	"kubevirt_migrate_vmi_pending_count":                   "",
	"kubevirt_migrate_vmi_running_count":                   "",
	"kubevirt_migrate_vmi_scheduling_count":                "",
	"kubevirt_vmi_cpu_affinity":                            "",
	"kubevirt_vmi_filesystem_capacity_bytes_total":         "",
	"kubevirt_vmi_memory_domain_bytes_total":               "",
	"kubevirt_vmi_memory_pgmajfault":                       "",
	"kubevirt_vmi_memory_pgminfault":                       "",
	"kubevirt_vmi_memory_swap_in_traffic_bytes_total":      "",
	"kubevirt_vmi_memory_swap_out_traffic_bytes_total":     "",
	"kubevirt_vmi_outdated_count":                          "",
	"kubevirt_vmi_storage_flush_times_ms_total":            "",
	"kubevirt_vmi_storage_read_times_ms_total":             "",
	"kubevirt_vmi_storage_write_times_ms_total":            "",
	"kubevirt_vmi_vcpu_seconds":                            "",
	"kubevirt_vmi_vcpu_wait_seconds":                       "",
	"kubevirt_vmsnapshot_disks_restored_from_source_total": "",
}

// Extract the name, help, and type from the metrics doc file
func ExtractMetrics(metricsContent string) ([]Metric, error) {
	var metrics []Metric
	re := regexp.MustCompile(`### (.*)\n(.*Type: (Counter|Gauge|Histogram|Summary)\.\n)?`)
	matches := re.FindAllStringSubmatch(metricsContent, -1)
	for _, match := range matches {
		name := match[1]
		help := ""
		if len(match) > 2 {
			help = match[2]
		}
		help = strings.TrimSpace(help)
		typ := ""
		if len(match) > 3 {
			typ = match[3]
		}
		metrics = append(metrics, Metric{Name: name, Help: help, Type: typ})
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics found")
	}
	return metrics, nil
}

// Set the correct metric type for creating MetricFamily
func CreateMetricFamily(m Metric) *dto.MetricFamily {
	metricType := dto.MetricType_UNTYPED

	switch m.Type {
	case "Counter":
		metricType = dto.MetricType_COUNTER
	case "Gauge":
		metricType = dto.MetricType_GAUGE
	case "Histogram":
		metricType = dto.MetricType_HISTOGRAM
	case "Summary":
		metricType = dto.MetricType_SUMMARY
	}

	return &dto.MetricFamily{
		Name: &m.Name,
		Help: &m.Help,
		Type: &metricType,
	}
}

// Read the metrics and parse them to a MetricFamily
func ReadMetrics(metricsPath string) ([]*dto.MetricFamily, error) {
	metricsContent, err := os.ReadFile(metricsPath)
	if err != nil {
		return nil, fmt.Errorf("error reading metrics file: %s", err)
	}

	metrics, err := ExtractMetrics(string(metricsContent))
	if err != nil {
		return nil, fmt.Errorf("error parsing metrics file: %s", err)
	}

	var metricFamily []*dto.MetricFamily
	for _, metric := range metrics {
		// Remove ignored metrics from all rules
		if _, ok := excludedMetrics[metric.Name]; !ok {
			mf := CreateMetricFamily(metric)
			metricFamily = append(metricFamily, mf)
		}
	}
	return metricFamily, nil
}
