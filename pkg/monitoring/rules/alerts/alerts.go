/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package alerts

import (
	"errors"
	"fmt"
	"os"
	"strings"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

const (
	prometheusRunbookAnnotationKey = "runbook_url"
	defaultRunbookURLTemplate      = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv          = "RUNBOOK_URL_TEMPLATE"

	severityAlertLabelKey        = "severity"
	operatorHealthImpactLabelKey = "operator_health_impact"
	summaryAnnotationKey         = "summary"
	descriptionAnnotationKey     = "description"
	partOfAlertLabelKey          = "kubernetes_operator_part_of"
	componentAlertLabelKey       = "kubernetes_operator_component"
	namespaceAlertLabelKey       = "namespace"
	kubevirtLabelValue           = "kubevirt"

	eightyPercent = 80
	fiveMinutes   = 5
)

func Register(registry *operatorrules.Registry, namespace string) error {
	componentAlerts := [][]promv1.Rule{
		systemAlerts(namespace),
		virtAPIAlerts(namespace),
		virtControllerAlerts(namespace),
		virtHandlerAlerts(namespace),
		virtOperatorAlerts(namespace),
	}

	allAlerts := make([][]promv1.Rule, 0, len(componentAlerts)+1)
	allAlerts = append(allAlerts, componentAlerts...)
	allAlerts = append(allAlerts, vmsAlerts)

	runbookURLTemplate := getRunbookURLTemplate()
	for _, alertGroup := range allAlerts {
		for _, alert := range alertGroup {
			alert.Labels[partOfAlertLabelKey] = kubevirtLabelValue
			alert.Labels[componentAlertLabelKey] = kubevirtLabelValue

			alert.Annotations[prometheusRunbookAnnotationKey] = fmt.Sprintf(runbookURLTemplate, alert.Alert)
		}
	}

	// Component and system alerts operate in the KubeVirt install namespace.
	// VM workload alerts derive namespace from their PromQL expression instead.
	for _, alertGroup := range componentAlerts {
		for _, alert := range alertGroup {
			alert.Labels[namespaceAlertLabelKey] = namespace
		}
	}

	return registry.RegisterAlerts(allAlerts...)
}

func componentDownDescription(component, extra string) string {
	return "{{ if $labels.pod }}" +
		"Pod {{ $labels.pod }}" + extra +
		" is unhealthy (reason: {{ $labels.reason }})." +
		"{{ else }}" +
		"No running " + component + " pods detected " +
		"and no container waiting reasons reported." +
		"{{ end }}"
}

func getRunbookURLTemplate() string {
	runbookURLTemplate, exists := os.LookupEnv(runbookURLTemplateEnv)
	if !exists {
		runbookURLTemplate = defaultRunbookURLTemplate
	}

	if strings.Count(runbookURLTemplate, "%s") != 1 {
		panic(errors.New("runbook URL template must have exactly 1 %s substring"))
	}

	return runbookURLTemplate
}
