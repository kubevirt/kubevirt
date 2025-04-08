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
 * Copyright The KubeVirt Authors
 *
 */

package testing

import (
	"fmt"

	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/onsi/gomega/format"
)

const (
	prometheusMetricNameLabel       = "__name__"
	prometheusHistogramBucketSuffix = "_bucket"
)

type PromResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type MetricMatcher struct {
	Metric operatormetrics.Metric
	Labels map[string]string
}

func (matcher *MetricMatcher) FailureMessage(actual interface{}) (message string) {
	msg := format.Message(actual, "to contain metric", matcher.Metric.GetOpts().Name)

	if matcher.Labels != nil {
		msg += fmt.Sprintf(" with labels %v", matcher.Labels)
	}

	return msg
}

func (matcher *MetricMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	msg := format.Message(actual, "not to contain metric", matcher.Metric.GetOpts().Name)

	if matcher.Labels != nil {
		msg += fmt.Sprintf(" with labels %v", matcher.Labels)
	}

	return msg
}

func (matcher *MetricMatcher) Match(actual interface{}) (success bool, err error) {
	actualMetric, ok := actual.(PromResult)
	if !ok {
		return false, fmt.Errorf("metric matcher requires a libmonitoring.PromResult")
	}

	actualName, ok := actualMetric.Metric[prometheusMetricNameLabel]
	if !ok {
		return false, fmt.Errorf("metric matcher requires a map with %s key", prometheusMetricNameLabel)
	}

	nameToMatch := matcher.Metric.GetOpts().Name
	if matcher.Metric.GetType() == operatormetrics.HistogramType || matcher.Metric.GetType() == operatormetrics.HistogramVecType {
		nameToMatch = nameToMatch + prometheusHistogramBucketSuffix
	}

	if actualName != nameToMatch {
		return false, nil
	}

	for k, v := range matcher.Labels {
		actualValue, ok := actualMetric.Metric[k]
		if !ok {
			return false, nil
		}
		if actualValue != v {
			return false, nil
		}
	}

	return true, nil
}
