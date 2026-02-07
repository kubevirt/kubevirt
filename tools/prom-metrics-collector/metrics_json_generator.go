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
	"encoding/json"
	"fmt"
	"strings"

	dto "github.com/prometheus/client_model/go"

	"github.com/kubevirt/monitoring/pkg/metrics/parser"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"kubevirt.io/kubevirt/pkg/monitoring/rules"
	"kubevirt.io/kubevirt/tests/libmonitoring"
)

// This should be used only for very rare cases where the naming conventions that are explained in the best practices:
// https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/ should be ignored.
var excludedMetrics = map[string]struct{}{
	"kubevirt_vmi_migration_data_total_bytes": {},
}

type RecordingRule struct {
	Record string `json:"record,omitempty"`
	Expr   string `json:"expr,omitempty"`
	Type   string `json:"type,omitempty"`
}

type Output struct {
	MetricFamilies []*dto.MetricFamily `json:"metricFamilies,omitempty"`
	RecordingRules []RecordingRule     `json:"recordingRules,omitempty"`
}

func main() {
	err := libmonitoring.RegisterAllMetrics()
	if err != nil {
		panic(err)
	}

	metricsList := operatormetrics.ListMetrics()

	rulesList := rules.ListRecordingRules()

	var metricFamilies []*dto.MetricFamily
	for _, m := range metricsList {
		if _, isExcludedMetric := excludedMetrics[m.GetOpts().Name]; !isExcludedMetric {
			pm := parser.Metric{
				Name: m.GetOpts().Name,
				Help: m.GetOpts().Help,
				Type: strings.ToUpper(string(m.GetBaseType())),
			}
			metricFamilies = append(metricFamilies, parser.CreateMetricFamily(pm))
		}
	}

	// Build recording rules JSON and record names for filtering
	excludedRecordRuleNames := make(map[string]struct{})
	var recRules []RecordingRule
	for _, r := range rulesList {
		name := r.GetOpts().Name
		if _, isExcludedMetric := excludedMetrics[name]; isExcludedMetric {
			continue
		}
		excludedRecordRuleNames[name] = struct{}{}
		recRules = append(recRules, RecordingRule{
			Record: name,
			Expr:   r.Expr.String(),
			Type:   strings.ToUpper(string(r.GetType())),
		})
	}

	var filteredFamilies []*dto.MetricFamily
	for _, mf := range metricFamilies {
		if _, isRec := excludedRecordRuleNames[*mf.Name]; isRec {
			continue
		}
		filteredFamilies = append(filteredFamilies, mf)
	}

	out := Output{MetricFamilies: filteredFamilies, RecordingRules: recRules}
	jsonBytes, err := json.Marshal(out)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(jsonBytes)) // Write the JSON string to standard output
}
