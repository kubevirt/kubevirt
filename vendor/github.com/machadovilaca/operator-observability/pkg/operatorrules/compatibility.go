package operatorrules

import promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

// Deprecated: operatorRegistry is deprecated.
var operatorRegistry = NewRegistry()

// Deprecated: RegisterRecordingRules is deprecated.
func RegisterRecordingRules(recordingRules ...[]RecordingRule) error {
	return operatorRegistry.RegisterRecordingRules(recordingRules...)
}

// Deprecated: RegisterAlerts is deprecated.
func RegisterAlerts(alerts ...[]promv1.Rule) error {
	return operatorRegistry.RegisterAlerts(alerts...)
}

// Deprecated: ListRecordingRules is deprecated.
func ListRecordingRules() []RecordingRule {
	return operatorRegistry.ListRecordingRules()
}

// Deprecated: ListAlerts is deprecated.
func ListAlerts() []promv1.Rule {
	return operatorRegistry.ListAlerts()
}

// Deprecated: CleanRegistry is deprecated.
func CleanRegistry() error {
	operatorRegistry = NewRegistry()
	return nil
}

// Deprecated: BuildPrometheusRule is deprecated.
func BuildPrometheusRule(name, namespace string, labels map[string]string) (*promv1.PrometheusRule, error) {
	return operatorRegistry.BuildPrometheusRule(name, namespace, labels)
}
