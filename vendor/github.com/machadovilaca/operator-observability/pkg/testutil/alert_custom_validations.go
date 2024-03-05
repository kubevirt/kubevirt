package testutil

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

func ValidateAlertNameLength(alert *promv1.Rule) []Problem {
	var result []Problem

	if len(alert.Alert) > 50 {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert name exceeds 50 characters",
		})
	}

	return result
}

func ValidateAlertHasDescriptionAnnotation(alert *promv1.Rule) []Problem {
	var result []Problem

	description := alert.Annotations["description"]
	if description == "" {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a description annotation",
		})
	}

	return result
}

func ValidateAlertRunbookURLAnnotation(alert *promv1.Rule) []Problem {
	var result []Problem

	runbookURL := alert.Annotations["runbook_url"]
	if runbookURL == "" {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a runbook_url annotation",
		})
	}

	return result
}

func ValidateAlertHealthImpactLabel(alert *promv1.Rule) []Problem {
	var result []Problem

	healthImpact := alert.Labels["operator_health_impact"]
	if !isValidHealthImpact(healthImpact) {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a operator_health_impact label with value critical, warning, or none",
		})
	}

	return result
}

func ValidateAlertPartOfAndComponentLabels(alert *promv1.Rule) []Problem {
	var result []Problem

	partOf := alert.Labels["kubernetes_operator_part_of"]
	if partOf == "" {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a kubernetes_operator_part_of label",
		})
	}

	component := alert.Labels["kubernetes_operator_component"]
	if component == "" {
		result = append(result, Problem{
			ResourceName: alert.Alert,
			Description:  "alert must have a kubernetes_operator_component label",
		})
	}

	return result
}

func isValidHealthImpact(healthImpact string) bool {
	return healthImpact == "critical" || healthImpact == "warning" || healthImpact == "none"
}
