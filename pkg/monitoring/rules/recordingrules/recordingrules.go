/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package recordingrules

import "github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"

func Register(registry *operatorrules.Registry, namespace string) error {
	return registry.RegisterRecordingRules(
		apiRecordingRules,
		nodesRecordingRules,
		operatorRecordingRules,
		virtRecordingRules(namespace),
		vmRecordingRules,
		vmiRecordingRules,
		vmsnapshotRecordingRules,
		deprecatedRecordingRules,
	)
}
