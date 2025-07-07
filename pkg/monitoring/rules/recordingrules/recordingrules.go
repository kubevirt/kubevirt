package recordingrules

import "github.com/machadovilaca/operator-observability/pkg/operatorrules"

func Register(namespace string) error {
	return operatorrules.RegisterRecordingRules(
		apiRecordingRules,
		nodesRecordingRules,
		operatorRecordingRules,
		virtRecordingRules(namespace),
		vmRecordingRules,
		vmiRecordingRules,
		vmsnapshotRecordingRules,
	)
}
