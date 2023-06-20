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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package main

import (
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
	metricsparser "github.com/kubevirt/monitoring/test/metrics/prom-metrics-linter/metrics-parser"

	dto "github.com/prometheus/client_model/go"
)

// excludedMetrics defines the metrics to ignore, open issue:https://github.com/kubevirt/kubevirt/issues/9714
// Do not add metrics to this list!
var excludedMetrics = map[string]struct{}{
	"kubevirt_hyperconverged_operator_health_status": struct{}{},
	"kubevirt_hco_out_of_band_modifications_count":   struct{}{},
	"kubevirt_hco_unsafe_modification_count":         struct{}{},
}

// Read the metrics and parse them to a MetricFamily
func ReadMetrics() []*dto.MetricFamily {
	hcoMetrics := metrics.HcoMetrics.GetMetricDesc()

	metricsList := make([]metricsparser.Metric, len(hcoMetrics))
	var metricFamily []*dto.MetricFamily
	for i, hcoMetric := range hcoMetrics {
		metricsList[i] = metricsparser.Metric{
			Name: hcoMetric.FqName,
			Help: hcoMetric.Help,
			Type: hcoMetric.Type,
		}
	}
	for _, hcoMetric := range metricsList {
		// Remove ignored metrics from all rules
		if _, isExcludedMetric := excludedMetrics[hcoMetric.Name]; !isExcludedMetric {
			mf := metricsparser.CreateMetricFamily(hcoMetric)
			metricFamily = append(metricFamily, mf)
		}
	}
	return metricFamily
}
