package operatorrules

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

var operatorRegistry = newRegistry()

type operatorRegisterer struct {
	registeredRecordingRules []RecordingRule
	registeredAlerts         []promv1.Rule
}

func newRegistry() operatorRegisterer {
	return operatorRegisterer{
		registeredRecordingRules: []RecordingRule{},
	}
}

// RegisterRecordingRules registers the given recording rules.
func RegisterRecordingRules(recordingRules ...[]RecordingRule) error {
	for _, recordingRuleList := range recordingRules {
		operatorRegistry.registeredRecordingRules = append(operatorRegistry.registeredRecordingRules, recordingRuleList...)
	}

	return nil
}

// RegisterAlerts registers the given alerts.
func RegisterAlerts(alerts ...[]promv1.Rule) error {
	for _, alertList := range alerts {
		operatorRegistry.registeredAlerts = append(operatorRegistry.registeredAlerts, alertList...)
	}

	return nil
}

// ListRecordingRules returns the registered recording rules.
func ListRecordingRules() []RecordingRule {
	return operatorRegistry.registeredRecordingRules
}

// ListAlerts returns the registered alerts.
func ListAlerts() []promv1.Rule {
	return operatorRegistry.registeredAlerts
}
