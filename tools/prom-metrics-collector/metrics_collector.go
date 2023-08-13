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

	parser "github.com/kubevirt/monitoring/pkg/metrics/parser"

	dto "github.com/prometheus/client_model/go"
)

// This should be used only for very rare cases where the naming conventions that are explained in the best practices:
// https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/#metrics-guidelines
// should be ignored.
var excludedMetrics = map[string]struct{}{
	"kubevirt_vmi_phase_count": struct{}{},
}

// Extract the name, help, and type from the metrics doc file
func ExtractMetrics(metricsContent string) ([]parser.Metric, error) {
	var metrics []parser.Metric
	re := regexp.MustCompile(`### (.*)\n(.*Type: (Counter|Gauge|Histogram|Summary)\.\n)?`)
	matches := re.FindAllStringSubmatch(metricsContent, -1)
	for _, match := range matches {
		name := match[1]
		help := ""
		if len(match) > 2 {
			help = strings.TrimSpace(match[2])
		}
		typ := ""
		if len(match) > 3 {
			typ = match[3]
		}
		metrics = append(metrics, parser.Metric{Name: name, Help: help, Type: typ})
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics found")
	}
	return metrics, nil
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
		if _, isExcludedMetric := excludedMetrics[metric.Name]; !isExcludedMetric {
			mf := parser.CreateMetricFamily(metric)
			metricFamily = append(metricFamily, mf)
		}
	}
	return metricFamily, nil
}
