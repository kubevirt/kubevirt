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

package testing

import (
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

func GomegaContainsCollectorResultMatcher(metric operatormetrics.Metric, expectedValue float64) types.GomegaMatcher {
	return &metricMatcher{
		Metric:        metric,
		ExpectedValue: expectedValue,
	}
}

type metricMatcher struct {
	Metric        operatormetrics.Metric
	ExpectedValue float64
}

func (matcher *metricMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to contain metric", matcher.Metric.GetOpts().Name, "with value", matcher.ExpectedValue)
}

func (matcher *metricMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to contain metric", matcher.Metric.GetOpts().Name, "with value", matcher.ExpectedValue)
}

func (matcher *metricMatcher) Match(actual interface{}) (success bool, err error) {
	cr := actual.(operatormetrics.CollectorResult)
	if cr.Metric.GetOpts().Name == matcher.Metric.GetOpts().Name {
		if cr.Value == matcher.ExpectedValue {
			return true, nil
		}
	}
	return false, nil
}
