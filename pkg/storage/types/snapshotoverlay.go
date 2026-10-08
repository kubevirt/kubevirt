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

package types

import (
	"encoding/json"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
)

// OverlayScratchPVCPrefix names the volume holding the qcow2 overlays of one
// External mode snapshot.
const OverlayScratchPVCPrefix = "snap-scratch-"

// OverlayOwnerVMIUIDAnnotation records which VMI the overlays belong to.
const OverlayOwnerVMIUIDAnnotation = "snapshot.kubevirt.io/owner-vmi-uid"

// OverlayProtectionFinalizer keeps the scratch PVC alive until its overlays are proven safe.
const OverlayProtectionFinalizer = "snapshot.kubevirt.io/overlay-protection"

// OverlaySnapshotReferenceAnnotation carries the expected transaction to virt-launcher
// through the VMI JSON already included in snapshot and commit requests.
const OverlaySnapshotReferenceAnnotation = "snapshot.kubevirt.io/overlay-reference"

// UncommittedOverlaysReason is the ManualRecoveryRequired reason for overlays a
// stopped launcher left uncommitted.
const UncommittedOverlaysReason = "UncommittedOverlays"

// OverlaysCommittedAnnotation records on the VM which scratch PVC had its overlays committed manually.
const OverlaysCommittedAnnotation = "snapshot.kubevirt.io/overlays-committed"

// SnapshotOverlayReference identifies the transaction carried in launcher
// requests. It is not part of the public API.
type SnapshotOverlayReference struct {
	ContentUID    types.UID `json:"contentUID"`
	ScratchPVCUID types.UID `json:"scratchPVCUID"`
}

// HasSnapshotOverlayState recognizes prepared or active work independently of
// the reference annotation, which can also be present on an ordinary VMI.
func HasSnapshotOverlayState(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Type != nil && *volume.Type == virtv1.SnapshotOverlay {
			return true
		}
	}
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, virtv1.VirtualMachineInstanceOverlaySnapshotActive, k8sv1.ConditionTrue)
}

// GetOverlaySnapshotReference returns nil when the VMI carries no reference.
func GetOverlaySnapshotReference(vmi *virtv1.VirtualMachineInstance) (*SnapshotOverlayReference, error) {
	if vmi == nil || vmi.Annotations[OverlaySnapshotReferenceAnnotation] == "" {
		return nil, nil
	}
	var reference SnapshotOverlayReference
	if err := json.Unmarshal([]byte(vmi.Annotations[OverlaySnapshotReferenceAnnotation]), &reference); err != nil {
		return nil, fmt.Errorf("invalid snapshot overlay reference: %w", err)
	}
	if reference.ContentUID == "" || reference.ScratchPVCUID == "" {
		return nil, fmt.Errorf("snapshot overlay reference requires content and scratch PVC UIDs")
	}
	return &reference, nil
}

// ScratchPVCName is derived from the content UID, so a snapshot recreated
// under the same name gets a new volume.
func ScratchPVCName(contentUID types.UID) string {
	return OverlayScratchPVCPrefix + string(contentUID)
}
