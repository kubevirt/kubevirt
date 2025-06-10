package operatorrules

import (
	"cmp"
	"fmt"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

// BuildPrometheusRule builds a PrometheusRule object from the registered recording rules and alerts.
func (r *Registry) BuildPrometheusRule(name, namespace string, labels map[string]string) (*promv1.PrometheusRule, error) {
	spec, err := r.buildPrometheusRuleSpec()
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

func (r *Registry) buildPrometheusRuleSpec() (*promv1.PrometheusRuleSpec, error) {
	var groups []promv1.RuleGroup

	if len(r.registeredRecordingRules) != 0 {
		groups = append(groups, promv1.RuleGroup{
			Name:  "recordingRules.rules",
			Rules: r.buildRecordingRulesRules(),
		})
	}

	if len(r.registeredAlerts) != 0 {
		groups = append(groups, promv1.RuleGroup{
			Name:  "alerts.rules",
			Rules: r.ListAlerts(),
		})
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("no registered recording rule or alert")
	}

	return &promv1.PrometheusRuleSpec{Groups: groups}, nil
}

func (r *Registry) buildRecordingRulesRules() []promv1.Rule {
	var rules []promv1.Rule

	for _, recordingRule := range r.registeredRecordingRules {
		rules = append(rules, promv1.Rule{
			Record: recordingRule.MetricsOpts.Name,
			Expr:   recordingRule.Expr,
			Labels: recordingRule.MetricsOpts.ConstLabels,
		})
	}

	slices.SortFunc(rules, func(a, b promv1.Rule) int {
		aKey := a.Record + ":" + a.Expr.String()
		bKey := b.Record + ":" + b.Expr.String()
		return cmp.Compare(aKey, bKey)
	})

	return rules
}
