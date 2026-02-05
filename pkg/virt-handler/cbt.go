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
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type CBTHandler struct {
	clientset       kubecli.KubevirtClient
	trackerInformer cache.SharedIndexInformer
}

func NewCBTHandler(
	clientset kubecli.KubevirtClient,
	trackerInformer cache.SharedIndexInformer,
) *CBTHandler {
	return &CBTHandler{
		clientset:       clientset,
		trackerInformer: trackerInformer,
	}
}

// HandleChangedBlockTracking updates CBT status based on domain state.
// If CBT is transitioning from Initializing to Enabled and there are trackers with checkpoints,
// they will be marked for redefinition before enabling CBT.
func (h *CBTHandler) HandleChangedBlockTracking(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil || !cbt.CBTStateInitializing(vmi.Status.ChangedBlockTracking) {
		return nil
	}

	if !h.allDisksHaveDataStore(vmi, domain) {
		return nil
	}

	// Before transitioning from Initializing to Enabled, mark trackers with checkpoints for redefinition
	if cbt.CompareCBTState(vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingInitializing) {
		if err := h.markTrackersForRedefinition(vmi); err != nil {
			return err
		}
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

func (h *CBTHandler) backupTrackersForVMI(vmi *v1.VirtualMachineInstance) []*backupv1.VirtualMachineBackupTracker {
	if h.trackerInformer == nil {
		return nil
	}

	key := fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)
	objs, err := h.trackerInformer.GetIndexer().ByIndex("vmi", key)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Warning("Failed to get backup trackers from informer index")
		return nil
	}

	var trackers []*backupv1.VirtualMachineBackupTracker
	for _, obj := range objs {
		if tracker, ok := obj.(*backupv1.VirtualMachineBackupTracker); ok {
			trackers = append(trackers, tracker)
		}
	}
	return trackers
}

func (h *CBTHandler) markTrackersForRedefinition(vmi *v1.VirtualMachineInstance) error {
	if h.trackerInformer == nil || h.clientset == nil {
		return fmt.Errorf("tracker informer or clientset is nil")
	}

	trackers := h.backupTrackersForVMI(vmi)

	for _, tracker := range trackers {
		if tracker.Status == nil || tracker.Status.LatestCheckpoint == nil {
			continue
		}
		if tracker.Status.CheckpointRedefinitionRequired != nil && *tracker.Status.CheckpointRedefinitionRequired {
			continue
		}

		patch := []byte(`{"status":{"checkpointRedefinitionRequired":true}}`)
		_, err := h.clientset.VirtualMachineBackupTracker(tracker.Namespace).Patch(
			context.Background(),
			tracker.Name,
			types.MergePatchType,
			patch,
			metav1.PatchOptions{},
			"status",
		)
		if err != nil {
			log.Log.Object(vmi).Reason(err).Warningf("Failed to mark tracker %s for checkpoint redefinition", tracker.Name)
			return fmt.Errorf("failed to mark tracker %s for checkpoint redefinition: %w", tracker.Name, err)
		}
		log.Log.Object(vmi).Infof("Marked tracker %s for checkpoint redefinition after VM restart", tracker.Name)
	}
	return nil
}
