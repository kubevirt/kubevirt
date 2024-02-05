package rules

import (
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/rules/alerts"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/rules/recordingrules"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	ruleName = hcoutil.HyperConvergedName + "-prometheus-rule"
)

func SetupRules() error {
	err := recordingrules.Register()
	if err != nil {
		return err
	}

	err = alerts.Register()
	if err != nil {
		return err
	}

	return nil
}

func BuildPrometheusRule(namespace string, owner metav1.OwnerReference) (*promv1.PrometheusRule, error) {
	rules, err := operatorrules.BuildPrometheusRule(
		ruleName,
		namespace,
		hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring),
	)
	if err != nil {
		return nil, err
	}

	rules.OwnerReferences = []metav1.OwnerReference{owner}

	return rules, nil
}

func ListRecordingRules() []operatorrules.RecordingRule {
	return operatorrules.ListRecordingRules()
}

func ListAlerts() []promv1.Rule {
	return operatorrules.ListAlerts()
}
