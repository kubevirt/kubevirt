package alerts

import (
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/metrics"
)

func healthAlerts() []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "OperatorConditionsUnhealthy",
			Expr:  intstr.FromString(fmt.Sprintf("kubevirt_hco_system_health_status == %f", metrics.SystemHealthStatusError)),
			Annotations: map[string]string{
				"description": "HCO and its secondary resources are in a critical state due to {{ $labels.reason }}.",
				"summary":     "HCO and its secondary resources are in a critical state.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:     "critical",
				healthImpactAlertLabelKey: "critical",
			},
		},
		{
			Alert: "OperatorConditionsUnhealthy",
			Expr:  intstr.FromString(fmt.Sprintf("kubevirt_hco_system_health_status == %f", metrics.SystemHealthStatusWarning)),
			Annotations: map[string]string{
				"description": "HCO and its secondary resources are in a warning state due to {{ $labels.reason }}.",
				"summary":     "HCO and its secondary resources are in a warning state.",
			},
			Labels: map[string]string{
				severityAlertLabelKey:     "warning",
				healthImpactAlertLabelKey: "warning",
			},
		},
	}
}
