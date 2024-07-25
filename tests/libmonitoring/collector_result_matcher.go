package libmonitoring

import (
	"github.com/machadovilaca/operator-observability/pkg/operatormetrics"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
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
