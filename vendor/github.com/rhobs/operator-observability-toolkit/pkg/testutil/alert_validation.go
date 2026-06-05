package testutil

import (
	"github.com/grafana/regexp"
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type AlertValidation = func(alert *promv1.Rule) []Problem

// based on https://sdk.operatorframework.io/docs/best-practices/observability-best-practices/#alerts-style-guide
var defaultAlertValidations = []AlertValidation{
	validateAlertName,
	validateAlertHasExpression,
	validateAlertHasSeverityLabel,
	validateAlertHasSummaryAnnotation,
}

func validateAlertName(alert *promv1.Rule) []Problem {
	var result []Problem

	if alert.Alert == "" || !isPascalCase(alert.Alert) {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a name in PascalCase format",
		})
	}

	return result
}

func validateAlertHasExpression(alert *promv1.Rule) []Problem {
	var result []Problem

	if alert.Expr.StrVal == "" {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have an expression",
		})
	}

	return result
}

func validateAlertHasSeverityLabel(alert *promv1.Rule) []Problem {
	var result []Problem

	severity := alert.Labels["severity"]
	if !isValidSeverity(severity) {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a severity label with value critical, warning, or info",
		})
	}

	return result
}

func validateAlertHasSummaryAnnotation(alert *promv1.Rule) []Problem {
	var result []Problem

	summary := alert.Annotations["summary"]
	if summary == "" {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a summary annotation",
		})
	}

	return result
}

func isPascalCase(s string) bool {
	pascalCasePattern := `^[A-Z][a-zA-Z0-9]*(?:[A-Z][a-zA-Z0-9]*)*$`
	pascalCaseRegex := regexp.MustCompile(pascalCasePattern)
	return pascalCaseRegex.MatchString(s)
}

func isValidSeverity(severity string) bool {
	return severity == "critical" || severity == "warning" || severity == "info"
}
