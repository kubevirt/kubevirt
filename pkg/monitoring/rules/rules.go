/*
Copyright 2023 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rules

import (
	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/rules/recordingrules"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

const (
	kubevirtPrometheusRuleName = "prometheus-kubevirt-rules"

	prometheusLabelKey   = "prometheus.kubevirt.io"
	prometheusLabelValue = "true"

	k8sAppLabelKey         = "k8s-app"
	partOfAlertLabelKey    = "kubernetes_operator_part_of"
	componentAlertLabelKey = "kubernetes_operator_component"
	kubevirtLabelValue     = "kubevirt"
)

func SetupRules(namespace string) error {
	err := recordingrules.Register(namespace)
	if err != nil {
		return err
	}

	return nil
}

func BuildPrometheusRule(namespace string) (*promv1.PrometheusRule, error) {
	rules, err := operatorrules.BuildPrometheusRule(
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

	// Add alerts to the PrometheusRule
	alerts := components.GetPrometheusAlerts(namespace)
	for _, alert := range alerts {
		alert.Labels[partOfAlertLabelKey] = kubevirtLabelValue
		alert.Labels[componentAlertLabelKey] = kubevirtLabelValue
	}
	rules.Spec.Groups = append(rules.Spec.Groups, promv1.RuleGroup{
		Name:  "alerts.rules",
		Rules: alerts,
	})

	return rules, nil
}

func ListRecordingRules() []operatorrules.RecordingRule {
	return operatorrules.ListRecordingRules()
}
