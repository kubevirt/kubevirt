package recordingrules

import "github.com/machadovilaca/operator-observability/pkg/operatorrules"

func Register() error {
	return operatorrules.RegisterRecordingRules(
		operatorRecordingRules,
	)
}
