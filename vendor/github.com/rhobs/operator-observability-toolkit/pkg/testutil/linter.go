package testutil

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

type Linter struct {
	customAlertValidations      []AlertValidation
	customRecordRuleValidations []RecordRuleValidation
}

func New() *Linter {
	return &Linter{
		customAlertValidations:      []AlertValidation{},
		customRecordRuleValidations: []RecordRuleValidation{},
	}
}

func (linter *Linter) AddCustomAlertValidations(validations ...AlertValidation) {
	linter.customAlertValidations = append(linter.customAlertValidations, validations...)
}

func (linter *Linter) AddCustomRecordRuleValidations(validations ...RecordRuleValidation) {
	linter.customRecordRuleValidations = append(linter.customRecordRuleValidations, validations...)
}

func (linter *Linter) LintAlerts(alerts []promv1.Rule) []Problem {
	var result []Problem

	for _, alert := range alerts {
		result = append(result, linter.LintAlert(&alert)...)
	}

	return result
}

func (linter *Linter) LintAlert(alert *promv1.Rule) []Problem {
	var result []Problem

	for _, alertValidation := range defaultAlertValidations {
		result = append(result, alertValidation(alert)...)
	}

	for _, alertValidation := range linter.customAlertValidations {
		result = append(result, alertValidation(alert)...)
	}

	return result
}

func (linter *Linter) LintRecordingRules(recordingRules []operatorrules.RecordingRule) []Problem {
	var result []Problem

	for _, recordingRule := range recordingRules {
		result = append(result, linter.LintRecordingRule(&recordingRule)...)
	}

	return result
}

func (linter *Linter) LintRecordingRule(recordingRule *operatorrules.RecordingRule) []Problem {
	var result []Problem

	for _, recordingRuleValidation := range defaultRecordRuleValidations {
		result = append(result, recordingRuleValidation(recordingRule)...)
	}

	for _, recordingRuleValidation := range linter.customRecordRuleValidations {
		result = append(result, recordingRuleValidation(recordingRule)...)
	}

	return result
}
