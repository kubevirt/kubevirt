package operatorrules

import (
	"cmp"
	"slices"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

var operatorRegistry = newRegistry()

type operatorRegisterer struct {
	registeredRecordingRules map[string]RecordingRule
	registeredAlerts         map[string]promv1.Rule
}

func newRegistry() operatorRegisterer {
	return operatorRegisterer{
		registeredRecordingRules: map[string]RecordingRule{},
		registeredAlerts:         map[string]promv1.Rule{},
	}
}

// RegisterRecordingRules registers the given recording rules.
func RegisterRecordingRules(recordingRules ...[]RecordingRule) error {
	for _, recordingRuleList := range recordingRules {
		for _, recordingRule := range recordingRuleList {
			key := recordingRule.MetricsOpts.Name + ":" + recordingRule.Expr.String()
			operatorRegistry.registeredRecordingRules[key] = recordingRule
		}
	}

	return nil
}

// RegisterAlerts registers the given alerts.
func RegisterAlerts(alerts ...[]promv1.Rule) error {
	for _, alertList := range alerts {
		for _, alert := range alertList {
			operatorRegistry.registeredAlerts[alert.Alert] = alert
		}
	}

	return nil
}

// ListRecordingRules returns the registered recording rules.
func ListRecordingRules() []RecordingRule {
	var rules []RecordingRule
	for _, rule := range operatorRegistry.registeredRecordingRules {
		rules = append(rules, rule)
	}

	slices.SortFunc(rules, func(a, b RecordingRule) int {
		aKey := a.GetOpts().Name + ":" + a.Expr.String()
		bKey := b.GetOpts().Name + ":" + b.Expr.String()

		return cmp.Compare(aKey, bKey)
	})

	return rules
}

// ListAlerts returns the registered alerts.
func ListAlerts() []promv1.Rule {
	var alerts []promv1.Rule
	for _, alert := range operatorRegistry.registeredAlerts {
		alerts = append(alerts, alert)
	}

	slices.SortFunc(alerts, func(a, b promv1.Rule) int {
		return cmp.Compare(a.Alert, b.Alert)
	})

	return alerts
}

// CleanRegistry removes all registered rules and alerts.
func CleanRegistry() error {
	operatorRegistry = newRegistry()
	return nil
}
