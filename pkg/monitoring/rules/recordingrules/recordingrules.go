package recordingrules

import "github.com/machadovilaca/operator-observability/pkg/operatorrules"

func Register(operatorRegistry *operatorrules.Registry) error {
	return operatorRegistry.RegisterRecordingRules(
		operatorRecordingRules,
	)
}
