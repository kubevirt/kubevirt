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
	defaultRunbookURLTemplate = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv     = "RUNBOOK_URL_TEMPLATE"
)

func Register(operatorRegistry *operatorrules.Registry) error {
	alerts := [][]promv1.Rule{
		clusterAlerts(),
	}

	runbookURLTemplate, err := getRunbookURLTemplate()
	if err != nil {
		return err
	}

	for _, alertGroup := range alerts {
		for _, alert := range alertGroup {
			alert.Labels["kubernetes_operator_part_of"] = "kubevirt"
			alert.Labels["kubernetes_operator_component"] = "cnv-observability"
			alert.Annotations["runbook_url"] = fmt.Sprintf(runbookURLTemplate, alert.Alert)
		}

	}

	return operatorRegistry.RegisterAlerts(alerts...)
}

func getRunbookURLTemplate() (string, error) {
	runbookURLTemplate, exists := os.LookupEnv(runbookURLTemplateEnv)
	if !exists {
		runbookURLTemplate = defaultRunbookURLTemplate
	}

	if strings.Count(runbookURLTemplate, "%s") != 1 {
		return "", errors.New("runbook URL template must have exactly 1 %s substring")
	}

	return runbookURLTemplate, nil
}
