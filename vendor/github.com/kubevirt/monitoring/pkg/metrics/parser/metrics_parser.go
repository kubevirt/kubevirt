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

package parser

import (
	dto "github.com/prometheus/client_model/go"
)

// Metric represents a Prometheus metric
type Metric struct {
	Name string `json:"name,omitempty"`
	Help string `json:"help,omitempty"`
	Type string `json:"type,omitempty"`
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
