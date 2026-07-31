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

package virthandler

import (
	"fmt"
	"slices"

	"k8s.io/client-go/tools/cache"
	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type CBTHandler struct {
	trackerInformer cache.SharedIndexInformer
}

func NewCBTHandler(
	trackerInformer cache.SharedIndexInformer,
) *CBTHandler {
	return &CBTHandler{
		trackerInformer: trackerInformer,
	}
}

// HandleChangedBlockTracking updates CBT status based on domain state.
// If CBT is Initializing and all disks have DataStore, it transitions to
// Enabled only after all trackers with checkpoints have been synced for
// the current pod lifecycle.
func (h *CBTHandler) HandleChangedBlockTracking(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil || !cbt.CBTStateInitializing(vmi.Status.ChangedBlockTracking) {
		return nil
	}

	if !h.allDisksHaveDataStore(vmi, domain) {
		return nil
	}

	needsRedefinition, err := h.anyTrackerNeedsRedefinition(vmi)
	if err != nil {
		return err
	}
	if needsRedefinition {
		return nil
	}

	cbt.SetCBTState(&vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
	return nil
}

func (h *CBTHandler) allDisksHaveDataStore(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	for _, volume := range vmi.Spec.Volumes {
		if !cbt.IsCBTEligibleVolume(&volume) {
			continue
		}
		found := false
		for _, disk := range domain.Spec.Devices.Disks {
			if disk.Alias.GetName() == volume.Name {
				found = true
				if disk.Source.DataStore == nil {
					return false
				}
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (h *CBTHandler) backupTrackersForVMI(vmi *v1.VirtualMachineInstance) ([]*backupv1.VirtualMachineBackupTracker, error) {
	if h.trackerInformer == nil {
		return nil, nil
	}

	key := fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)
	objs, err := h.trackerInformer.GetIndexer().ByIndex("vmi", key)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup trackers from informer index: %w", err)
	}

	var trackers []*backupv1.VirtualMachineBackupTracker
	for _, obj := range objs {
		if tracker, ok := obj.(*backupv1.VirtualMachineBackupTracker); ok {
			trackers = append(trackers, tracker)
		}
	}
	return trackers, nil
}

func (h *CBTHandler) anyTrackerNeedsRedefinition(vmi *v1.VirtualMachineInstance) (bool, error) {
	podUID := cbt.ActivePodUID(vmi)
	if podUID == "" {
		return false, nil
	}
	trackers, err := h.backupTrackersForVMI(vmi)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(trackers, func(t *backupv1.VirtualMachineBackupTracker) bool {
		return t.Status != nil &&
			t.Status.LatestCheckpoint != nil &&
			t.Status.LatestCheckpoint.Name != "" &&
			(t.Status.LastTrackedPodUID == nil || *t.Status.LastTrackedPodUID != podUID)
	}), nil
}
