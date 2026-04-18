/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package rules

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"

	"kubevirt.io/kubevirt/pkg/monitoring/rules/alerts"
	"kubevirt.io/kubevirt/pkg/monitoring/rules/recordingrules"
)

const (
	kubevirtPrometheusRuleName = "prometheus-kubevirt-rules"

	prometheusLabelKey   = "prometheus.kubevirt.io"
	prometheusLabelValue = "true"

	k8sAppLabelKey     = "k8s-app"
	kubevirtLabelValue = "kubevirt"
)

var registry = operatorrules.NewRegistry()

func SetupRules(namespace string) error {
	err := recordingrules.Register(registry, namespace)
	if err != nil {
		return err
	}

	err = alerts.Register(registry, namespace)
	if err != nil {
		return err
	}

	return nil
}

func BuildPrometheusRule(namespace string) (*promv1.PrometheusRule, error) {
	rules, err := registry.BuildPrometheusRule(
		kubevirtPrometheusRuleName,
		namespace,
		map[string]string{
			prometheusLabelKey: prometheusLabelValue,
			k8sAppLabelKey:     kubevirtLabelValue,
		},
	)
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func ListRecordingRules() []operatorrules.RecordingRule {
	return registry.ListRecordingRules()
}

func ListAlerts() []promv1.Rule {
	return registry.ListAlerts()
}
