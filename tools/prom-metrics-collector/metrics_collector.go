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

	parser "github.com/kubevirt/monitoring/pkg/metrics/parser"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
)

// This should be used only for very rare cases where the naming conventions that are explained in the best practices:
// https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/ should be ignored.
var excludedMetrics = map[string]struct{}{
	"kubevirt_vmi_migration_data_total_bytes": {},
}

// RecordingRule is an entry for the linter JSON.
type RecordingRule struct {
	Record string `json:"record,omitempty"`
	Expr   string `json:"expr,omitempty"`
	Type   string `json:"type,omitempty"`
}

func ExtractRecordingRules() ([]RecordingRule, error) {
	if err := rules.SetupRules("ci"); err != nil {
		return nil, err
	}
	var recRules []RecordingRule
	for _, rr := range rules.ListRecordingRules() {
		if _, isExcluded := excludedMetrics[rr.MetricsOpts.Name]; isExcluded {
			continue
		}
		recRules = append(recRules, RecordingRule{
			Record: rr.MetricsOpts.Name,
			Expr:   rr.Expr.String(),
			Type:   string(rr.MetricType),
		})
	}
	return recRules, nil
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
func ReadMetrics(metricsPath string, recordingRuleNames map[string]struct{}) ([]*dto.MetricFamily, error) {
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
		// Remove ignored metrics and any that are recording rules
		if _, isExcludedMetric := excludedMetrics[metric.Name]; isExcludedMetric {
			continue
		}
		if _, isRecordingRule := recordingRuleNames[metric.Name]; isRecordingRule {
			continue
		}
		mf := parser.CreateMetricFamily(metric)
		metricFamily = append(metricFamily, mf)
	}
	return metricFamily, nil
}
