package operatorrules

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

// RecordingRule is a struct that represents a Prometheus recording rule.
type RecordingRule struct {
	MetricsOpts operatormetrics.MetricOpts
	MetricType  operatormetrics.MetricType
	Expr        intstr.IntOrString
}

// GetOpts returns the metric options of the recording rule.
func (c RecordingRule) GetOpts() operatormetrics.MetricOpts {
	return c.MetricsOpts
}

// GetType returns the metric type of the recording rule.
func (c RecordingRule) GetType() operatormetrics.MetricType {
	return c.MetricType
}
