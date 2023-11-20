package operatorrules

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// BuildPrometheusRule builds a PrometheusRule object from the registered recording rules and alerts.
func BuildPrometheusRule(name, namespace string, labels map[string]string) (*promv1.PrometheusRule, error) {
	spec, err := buildPrometheusRuleSpec()
	if err != nil {
		return nil, err
	}

	return &promv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: promv1.SchemeGroupVersion.String(),
			Kind:       promv1.PrometheusRuleKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: *spec,
	}, nil
}

func buildPrometheusRuleSpec() (*promv1.PrometheusRuleSpec, error) {
	var groups []promv1.RuleGroup

	if len(operatorRegistry.registeredRecordingRules) != 0 {
		groups = append(groups, promv1.RuleGroup{
			Name:  "recordingRules.rules",
			Rules: buildRecordingRulesRules(),
		})
	}

	if len(operatorRegistry.registeredAlerts) != 0 {
		groups = append(groups, promv1.RuleGroup{
			Name:  "alerts.rules",
			Rules: buildAlertsRules(),
		})
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no registered recording rule or alert")
	}

	return &promv1.PrometheusRuleSpec{Groups: groups}, nil
}

func buildRecordingRulesRules() []promv1.Rule {
	var rules []promv1.Rule

	for _, recordingRule := range operatorRegistry.registeredRecordingRules {
		rules = append(rules, promv1.Rule{
			Record: recordingRule.MetricsOpts.Name,
			Expr:   recordingRule.Expr,
		})
	}

	return rules
}

func buildAlertsRules() []promv1.Rule {
	var rules []promv1.Rule
	rules = append(rules, operatorRegistry.registeredAlerts...)
	return rules
}
