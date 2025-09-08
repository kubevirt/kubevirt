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

	partOfAlertLabelKey    = "kubernetes_operator_part_of"
	componentAlertLabelKey = "kubernetes_operator_component"
	kubevirtLabelValue     = "kubevirt"

	durationFiveMinutes = "5 minutes"
)

func Register(namespace string) error {
	alerts := [][]promv1.Rule{
		systemAlerts(namespace),
		virtApiAlerts(namespace),
		virtControllerAlerts(namespace),
		virtHandlerAlerts(namespace),
		virtOperatorAlerts(namespace),
		vmsAlerts,
	}

	runbookURLTemplate := getRunbookURLTemplate()
	for _, alertGroup := range alerts {
		for _, alert := range alertGroup {
			alert.Labels[partOfAlertLabelKey] = kubevirtLabelValue
			alert.Labels[componentAlertLabelKey] = kubevirtLabelValue

			alert.Annotations[prometheusRunbookAnnotationKey] = fmt.Sprintf(runbookURLTemplate, alert.Alert)
		}
	}

	return operatorrules.RegisterAlerts(alerts...)
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

func getErrorRatio(ns string, podName string, errorCodeRegex string, durationInMinutes int) string {
	errorRatioQuery := "sum ( rate ( kubevirt_rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  /  sum ( rate ( kubevirt_rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
	return fmt.Sprintf(errorRatioQuery, ns, podName, errorCodeRegex, durationInMinutes, ns, podName, durationInMinutes)
}

func getRestCallsFailedWarning(failingCallsPercentage int, component, duration string) string {
	const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
	return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
}
