/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package snapshotoverlay

import (
	v1 "kubevirt.io/api/core/v1"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/storage/utilityvolume"
)

// AwaitingRecoveryScratch reports a VMI still waiting for the scratch volume it recovers
func AwaitingRecoveryScratch(vmi *v1.VirtualMachineInstance) bool {
	expected := vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation]
	if expected == "" {
		return false
	}
	found, err := utilityvolume.Find(vmi, expected, expected, v1.SnapshotOverlay)
	return err != nil || !found
}
