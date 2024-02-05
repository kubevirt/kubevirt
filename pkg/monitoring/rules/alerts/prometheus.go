package alerts

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/machadovilaca/operator-observability/pkg/operatorrules"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

const (
	prometheusRunbookAnnotationKey = "runbook_url"
	partOfAlertLabelKey            = "kubernetes_operator_part_of"
	partOfAlertLabelValue          = "kubevirt"
	componentAlertLabelKey         = "kubernetes_operator_component"
	componentAlertLabelValue       = "hyperconverged-cluster-operator"
	defaultRunbookURLTemplate      = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv          = "RUNBOOK_URL_TEMPLATE"
)

func Register() error {
	alerts := [][]promv1.Rule{
		operatorAlerts(),
	}

	runbookURLTemplate := getRunbookURLTemplate()
	for _, alertGroup := range alerts {
		for _, alert := range alertGroup {
			alert.Labels[partOfAlertLabelKey] = partOfAlertLabelValue
			alert.Labels[componentAlertLabelKey] = componentAlertLabelValue
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
