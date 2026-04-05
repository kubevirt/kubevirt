/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
	kubevirtLabelValue           = "kubevirt"

	eightyPercent = 80
	fiveMinutes   = 5
)

func Register(registry *operatorrules.Registry, namespace string) error {
	alerts := [][]promv1.Rule{
		systemAlerts(namespace),
		virtAPIAlerts(namespace),
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

	return registry.RegisterAlerts(alerts...)
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

func getErrorRatio(ns, podName, errorCodeRegex string, durationInMinutes int) string {
	errorRatioQuery := "sum ( rate ( kubevirt_rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  / " +
		" sum ( rate ( kubevirt_rest_client_requests_total{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
	return fmt.Sprintf(errorRatioQuery, ns, podName, errorCodeRegex, durationInMinutes, ns, podName, durationInMinutes)
}

func getRestCallsFailedWarning(failingCallsPercentage int, component string, durationInMinutes int) string {
	duration := fmt.Sprintf("%d minutes", durationInMinutes)

	const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
	return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
}
