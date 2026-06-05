package testutil

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

type RecordRuleValidation = func(rr *operatorrules.RecordingRule) []Problem

var defaultRecordRuleValidations = []RecordRuleValidation{
	validateRecordingRuleName,
	validateRecordingRuleExpression,
}

func validateRecordingRuleName(recordingRule *operatorrules.RecordingRule) []Problem {
	var result []Problem

	if recordingRule.MetricsOpts.Name == "" {
		result = append(result, Problem{
			ResourceName: recordingRule.MetricsOpts.Name,
			Description:  "recording rule must have a name",
		})
	}

	return result
}

func validateRecordingRuleExpression(recordingRule *operatorrules.RecordingRule) []Problem {
	var result []Problem

	if recordingRule.Expr.StrVal == "" {
		result = append(result, Problem{
			ResourceName: recordingRule.MetricsOpts.Name,
			Description:  "recording rule must have an expression",
		})
	}

	return result
}
