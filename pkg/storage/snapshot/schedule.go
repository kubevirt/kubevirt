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

package snapshot

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"

	"github.com/robfig/cron/v3"
)

const (
	vmSnapshotScheduleFinalizer = "snapshot.kubevirt.io/vmsnapshotschedule-protection"
)

// VMSnapshotScheduleController is responsible for managing VirtualMachineSnapshotSchedule resources
type VMSnapshotScheduleController struct {
	VMSnapshotScheduleInformer cache.SharedIndexInformer
	VMSnapshotInformer         cache.SharedIndexInformer
	VMInformer                 cache.SharedIndexInformer

	Client kubecli.KubevirtClient

	ResyncPeriod time.Duration
}

// Run runs the controller
func (ctrl *VMSnapshotScheduleController) Run(threadiness int, stopCh <-chan struct{}) {
	defer controller.HandlePanic()
	defer log.Log.Info("Shutting down snapshot schedule controller")

	log.Log.Info("Starting snapshot schedule controller")

	if !cache.WaitForCacheSync(stopCh, ctrl.VMSnapshotScheduleInformer.HasSynced, ctrl.VMSnapshotInformer.HasSynced, ctrl.VMInformer.HasSynced) {
		log.Log.Error("Timed out waiting for caches to sync")
		return
	}

	// Set up event handlers
	_, err := ctrl.VMSnapshotScheduleInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshotSchedule,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshotSchedule(newObj) },
			DeleteFunc: ctrl.handleVMSnapshotSchedule,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		log.Log.Errorf("Failed to add event handler for VMSnapshotSchedule: %v", err)
		return
	}

	// Process existing schedules
	schedules := ctrl.VMSnapshotScheduleInformer.GetStore().List()
	log.Log.Infof("Initial processing of %d snapshot schedules", len(schedules))
	for _, obj := range schedules {
		ctrl.handleVMSnapshotSchedule(obj)
	}

	// Use a ticker for periodic processing - use 1 minute intervals for snapshot schedules
	// since cron schedules can be as frequent as every minute
	tickerPeriod := ctrl.ResyncPeriod
	if tickerPeriod > time.Minute {
		tickerPeriod = time.Minute
	}
	log.Log.Infof("Snapshot schedule controller will check schedules every %v", tickerPeriod)
	ticker := time.NewTicker(tickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			// Process all schedules periodically
			schedules := ctrl.VMSnapshotScheduleInformer.GetStore().List()
			log.Log.Infof("Processing %d snapshot schedules", len(schedules))
			for _, obj := range schedules {
				ctrl.handleVMSnapshotSchedule(obj)
			}
		}
	}
}

// handleVMSnapshotSchedule handles changes to VirtualMachineSnapshotSchedule resources
func (ctrl *VMSnapshotScheduleController) handleVMSnapshotSchedule(obj interface{}) {
	// Handle deleted objects - nothing to do, object is already gone
	if _, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		// Object was deleted and we missed the delete event
		// The object is already gone from the API server, nothing to process
		log.Log.V(3).Infof("Received DeletedFinalStateUnknown for VirtualMachineSnapshotSchedule, skipping")
		return
	}

	schedule, ok := obj.(*snapshotv1.VirtualMachineSnapshotSchedule)
	if !ok {
		log.Log.Errorf("Expected VirtualMachineSnapshotSchedule, got %+v", obj)
		return
	}

	if err := ctrl.updateVMSnapshotSchedule(schedule); err != nil {
		// Don't log "not found" errors as they're expected when object is deleted
		if !errors.IsNotFound(err) {
			log.Log.Errorf("Failed to update VirtualMachineSnapshotSchedule %s/%s: %v", schedule.Namespace, schedule.Name, err)
		}
	}
}

// updateVMSnapshotSchedule processes a VirtualMachineSnapshotSchedule
func (ctrl *VMSnapshotScheduleController) updateVMSnapshotSchedule(schedule *snapshotv1.VirtualMachineSnapshotSchedule) error {
	log.Log.Infof("Processing VirtualMachineSnapshotSchedule %s/%s", schedule.Namespace, schedule.Name)

	// First, fetch the latest version from the API server to ensure object still exists
	// This handles the case where the informer cache has stale data
	current, err := ctrl.Client.GeneratedKubeVirtClient().SnapshotV1beta1().VirtualMachineSnapshotSchedules(schedule.Namespace).Get(context.TODO(), schedule.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Object no longer exists, nothing to do
			log.Log.V(3).Infof("VirtualMachineSnapshotSchedule %s/%s no longer exists, skipping", schedule.Namespace, schedule.Name)
			return nil
		}
		return err
	}

	// Use the fresh copy from API server
	schedule = current

	// Handle deletion
	if schedule.DeletionTimestamp != nil {
		if controller.HasFinalizer(schedule, vmSnapshotScheduleFinalizer) {
			// Perform cleanup (e.g., delete associated snapshots if needed)
			scheduleCopy := schedule.DeepCopy()
			controller.RemoveFinalizer(scheduleCopy, vmSnapshotScheduleFinalizer)
			_, err := ctrl.Client.GeneratedKubeVirtClient().SnapshotV1beta1().VirtualMachineSnapshotSchedules(scheduleCopy.Namespace).Update(context.TODO(), scheduleCopy, metav1.UpdateOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}

	// Add finalizer if not present
	if !controller.HasFinalizer(schedule, vmSnapshotScheduleFinalizer) {
		scheduleCopy := schedule.DeepCopy()
		controller.AddFinalizer(scheduleCopy, vmSnapshotScheduleFinalizer)
		updatedSchedule, err := ctrl.Client.GeneratedKubeVirtClient().SnapshotV1beta1().VirtualMachineSnapshotSchedules(scheduleCopy.Namespace).Update(context.TODO(), scheduleCopy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		// Use the updated schedule for the rest of the processing
		schedule = updatedSchedule
	}

	// Check if schedule is disabled
	if schedule.Spec.Disabled != nil && *schedule.Spec.Disabled {
		log.Log.V(3).Infof("VirtualMachineSnapshotSchedule %s/%s is disabled", schedule.Namespace, schedule.Name)
		return nil
	}

	// Get VMs that match the selector
	vms, err := ctrl.getMatchingVMs(schedule)
	if err != nil {
		return err
	}

	log.Log.Infof("Found %d VMs matching schedule %s/%s selector", len(vms), schedule.Namespace, schedule.Name)

	// Process each matching VM
	for _, vm := range vms {
		if err := ctrl.processVMSnapshotSchedule(schedule, &vm); err != nil {
			log.Log.Errorf("Failed to process snapshot schedule for VM %s/%s: %v", vm.Namespace, vm.Name, err)
			// Continue with other VMs
		}
	}

	// Update status after processing
	if err := ctrl.updateScheduleStatus(context.TODO(), schedule); err != nil {
		log.Log.Errorf("Failed to update status for schedule %s/%s: %v", schedule.Namespace, schedule.Name, err)
	}

	return nil
}

// updateScheduleStatus updates the schedule status with next run time
func (ctrl *VMSnapshotScheduleController) updateScheduleStatus(ctx context.Context, schedule *snapshotv1.VirtualMachineSnapshotSchedule) error {
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := ctrl.tryUpdateScheduleStatus(ctx, schedule)
		if err == nil {
			return nil
		}

		// If it's a conflict error, retry
		if errors.IsConflict(err) {
			log.Log.V(3).Infof("Conflict updating schedule %s/%s status, retrying (attempt %d/%d)", schedule.Namespace, schedule.Name, attempt+1, maxRetries)
			continue
		}

		// For non-conflict errors, return immediately
		return err
	}

	return fmt.Errorf("failed to update schedule status after %d retries due to conflicts", maxRetries)
}

// tryUpdateScheduleStatus attempts a single update of the schedule status
func (ctrl *VMSnapshotScheduleController) tryUpdateScheduleStatus(ctx context.Context, schedule *snapshotv1.VirtualMachineSnapshotSchedule) error {
	// Fetch the latest version to avoid update conflicts
	current, err := ctrl.Client.GeneratedKubeVirtClient().SnapshotV1beta1().VirtualMachineSnapshotSchedules(schedule.Namespace).Get(ctx, schedule.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	// Use last snapshot time as base, or now if none
	var baseTime time.Time
	if current.Status != nil && current.Status.LastSnapshotTime != nil {
		baseTime = current.Status.LastSnapshotTime.Time
	} else {
		baseTime = now
	}

	// Calculate next run from base time
	nextRun, err := ctrl.calculateNextRun(current.Spec.Schedule, baseTime)
	if err != nil {
		return fmt.Errorf("failed to calculate next run: %v", err)
	}

	// Initialize status if nil
	if current.Status == nil {
		current.Status = &snapshotv1.VirtualMachineSnapshotScheduleStatus{}
	}

	// Update the status
	current.Status.NextSnapshotTime = &metav1.Time{Time: nextRun}

	// Update last snapshot time if we have snapshots
	vms, err := ctrl.getMatchingVMs(current)
	if err != nil {
		return err
	}

	var latestSnapshotTime *metav1.Time
	for _, vm := range vms {
		snapshots, err := ctrl.getSnapshotsForSchedule(current, &vm)
		if err != nil {
			continue
		}
		if len(snapshots) > 0 && (latestSnapshotTime == nil || snapshots[0].CreationTimestamp.After(latestSnapshotTime.Time)) {
			latestSnapshotTime = &snapshots[0].CreationTimestamp
		}
	}

	if latestSnapshotTime != nil {
		current.Status.LastSnapshotTime = latestSnapshotTime
	}

	// Persist the status update to the cluster
	_, err = ctrl.Client.GeneratedKubeVirtClient().SnapshotV1beta1().VirtualMachineSnapshotSchedules(schedule.Namespace).UpdateStatus(ctx, current, metav1.UpdateOptions{})
	if err == nil {
		log.Log.V(3).Infof("Updated schedule %s/%s status: next=%v, last=%v", current.Namespace, current.Name, current.Status.NextSnapshotTime, latestSnapshotTime)
	}
	return err
}

// getMatchingVMs returns VMs that match the schedule's selector
func (ctrl *VMSnapshotScheduleController) getMatchingVMs(schedule *snapshotv1.VirtualMachineSnapshotSchedule) ([]kubevirtv1.VirtualMachine, error) {
	var vms []kubevirtv1.VirtualMachine

	if schedule.Spec.ClaimSelector == nil {
		// If no selector, get all VMs in the same namespace
		objs := ctrl.VMInformer.GetStore().List()
		for _, obj := range objs {
			vm, ok := obj.(*kubevirtv1.VirtualMachine)
			if !ok {
				continue
			}
			if vm.Namespace == schedule.Namespace {
				vms = append(vms, *vm)
			}
		}
	} else {
		// Use selector to filter VMs
		selector, err := metav1.LabelSelectorAsSelector(schedule.Spec.ClaimSelector)
		if err != nil {
			return nil, err
		}

		objs := ctrl.VMInformer.GetStore().List()
		for _, obj := range objs {
			vm, ok := obj.(*kubevirtv1.VirtualMachine)
			if !ok {
				continue
			}
			if vm.Namespace != schedule.Namespace {
				continue
			}
			if selector.Matches(labels.Set(vm.Labels)) {
				vms = append(vms, *vm)
			}
		}
	}

	return vms, nil
}

// processVMSnapshotSchedule processes snapshot creation and cleanup for a specific VM
func (ctrl *VMSnapshotScheduleController) processVMSnapshotSchedule(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vm *kubevirtv1.VirtualMachine) error {
	log.Log.Infof("Processing snapshot schedule for VM %s/%s", vm.Namespace, vm.Name)

	// Get existing snapshots for this VM and schedule
	snapshots, err := ctrl.getSnapshotsForSchedule(schedule, vm)
	if err != nil {
		return err
	}

	log.Log.Infof("Found %d existing snapshots for VM %s/%s from schedule %s", len(snapshots), vm.Namespace, vm.Name, schedule.Name)

	// Check if we need to create a new snapshot
	if ctrl.shouldCreateSnapshot(schedule, snapshots) {
		log.Log.Infof("Creating new snapshot for VM %s/%s", vm.Namespace, vm.Name)
		if err := ctrl.createSnapshot(schedule, vm); err != nil {
			return err
		}
	} else {
		log.Log.Infof("No new snapshot needed for VM %s/%s (not time yet)", vm.Namespace, vm.Name)
	}

	// Clean up old snapshots based on retention policy
	if err := ctrl.cleanupSnapshots(schedule, snapshots); err != nil {
		return err
	}

	return nil
}

// getSnapshotsForSchedule returns snapshots created by this schedule for the given VM
func (ctrl *VMSnapshotScheduleController) getSnapshotsForSchedule(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vm *kubevirtv1.VirtualMachine) ([]*snapshotv1.VirtualMachineSnapshot, error) {
	var snapshots []*snapshotv1.VirtualMachineSnapshot

	objs := ctrl.VMSnapshotInformer.GetStore().List()
	for _, obj := range objs {
		snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
		if !ok {
			continue
		}

		// Check if snapshot belongs to this schedule and VM
		if snapshot.Namespace != schedule.Namespace {
			continue
		}

		if snapshot.Labels == nil {
			continue
		}

		scheduleName, exists := snapshot.Labels["snapshot.kubevirt.io/schedule-name"]
		if !exists || scheduleName != schedule.Name {
			continue
		}

		vmName, exists := snapshot.Labels["snapshot.kubevirt.io/source-vm-name"]
		if !exists || vmName != vm.Name {
			continue
		}

		snapshots = append(snapshots, snapshot)
	}

	// Sort snapshots by creation time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].CreationTimestamp.IsZero() {
			return false
		}
		if snapshots[j].CreationTimestamp.IsZero() {
			return true
		}
		return snapshots[i].CreationTimestamp.After(snapshots[j].CreationTimestamp.Time)
	})

	return snapshots, nil
}

// shouldCreateSnapshot determines if a new snapshot should be created
func (ctrl *VMSnapshotScheduleController) shouldCreateSnapshot(schedule *snapshotv1.VirtualMachineSnapshotSchedule, snapshots []*snapshotv1.VirtualMachineSnapshot) bool {
	now := time.Now().UTC()

	// Get the last snapshot time
	var lastSnapshotTime time.Time
	if len(snapshots) > 0 {
		lastSnapshotTime = snapshots[0].CreationTimestamp.Time // Already sorted, newest first
	} else {
		// If no snapshots exist, create the first one immediately
		return true
	}

	// Calculate next run based on schedule
	nextRun, err := ctrl.calculateNextRun(schedule.Spec.Schedule, lastSnapshotTime)
	if err != nil {
		log.Log.Errorf("Failed to calculate next run for schedule %s: %v", schedule.Spec.Schedule, err)
		return false
	}

	return nextRun.Before(now) || nextRun.Equal(now)
}

// calculateNextRun calculates the next run time based on the cron schedule
func (ctrl *VMSnapshotScheduleController) calculateNextRun(schedule string, from time.Time) (time.Time, error) {
	// Handle predefined schedules
	switch schedule {
	case "@hourly":
		return from.Add(time.Hour), nil
	case "@daily":
		return from.AddDate(0, 0, 1), nil
	case "@weekly":
		return from.AddDate(0, 0, 7), nil
	case "@monthly":
		return from.AddDate(0, 1, 0), nil
	case "@yearly":
		return from.AddDate(1, 0, 0), nil
	}

	// For full cron expressions, try to use the cron library
	cronSchedule, err := cron.ParseStandard(schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse cron schedule %s: %v", schedule, err)
	}

	return cronSchedule.Next(from), nil
}

// createSnapshot creates a new snapshot for the given VM
func (ctrl *VMSnapshotScheduleController) createSnapshot(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vm *kubevirtv1.VirtualMachine) error {
	snapshotName := fmt.Sprintf("%s-%s-%d", schedule.Name, vm.Name, time.Now().Unix())

	apiGroup := "kubevirt.io"
	snapshot := &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: schedule.Namespace,
			Labels: map[string]string{
				"snapshot.kubevirt.io/schedule-name":       schedule.Name,
				"snapshot.kubevirt.io/source-vm-name":      vm.Name,
				"snapshot.kubevirt.io/source-vm-namespace": vm.Namespace,
			},
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
		},
	}

	// Apply template labels and annotations
	if schedule.Spec.SnapshotTemplate.Labels != nil {
		for k, v := range schedule.Spec.SnapshotTemplate.Labels {
			snapshot.Labels[k] = v
		}
	}

	if schedule.Spec.SnapshotTemplate.Annotations != nil {
		snapshot.Annotations = make(map[string]string)
		for k, v := range schedule.Spec.SnapshotTemplate.Annotations {
			snapshot.Annotations[k] = v
		}
	}

	// Apply template spec
	if schedule.Spec.SnapshotTemplate.Spec.DeletionPolicy != nil {
		snapshot.Spec.DeletionPolicy = schedule.Spec.SnapshotTemplate.Spec.DeletionPolicy
	}
	if schedule.Spec.SnapshotTemplate.Spec.FailureDeadline != nil {
		snapshot.Spec.FailureDeadline = schedule.Spec.SnapshotTemplate.Spec.FailureDeadline
	}

	_, err := ctrl.Client.VirtualMachineSnapshot(schedule.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	log.Log.Infof("Created snapshot %s/%s for schedule %s", schedule.Namespace, snapshotName, schedule.Name)
	return nil
}

// cleanupSnapshots removes snapshots that exceed the retention policy
func (ctrl *VMSnapshotScheduleController) cleanupSnapshots(schedule *snapshotv1.VirtualMachineSnapshotSchedule, snapshots []*snapshotv1.VirtualMachineSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	var toDelete []*snapshotv1.VirtualMachineSnapshot

	// Check max count
	// Snapshots are sorted newest-first, so we keep the first MaxCount and delete the rest (oldest)
	if schedule.Spec.Retention.MaxCount != nil && len(snapshots) > int(*schedule.Spec.Retention.MaxCount) {
		// Delete oldest snapshots beyond the limit (those after index MaxCount-1)
		toDelete = append(toDelete, snapshots[int(*schedule.Spec.Retention.MaxCount):]...)
	}

	// Check expiration
	if schedule.Spec.Retention.Expires != nil {
		cutoff := time.Now().Add(-schedule.Spec.Retention.Expires.Duration)
		for _, snapshot := range snapshots {
			if !snapshot.CreationTimestamp.IsZero() && snapshot.CreationTimestamp.Time.Before(cutoff) {
				toDelete = append(toDelete, snapshot)
			}
		}
	}

	// Remove duplicates
	toDeleteSet := sets.NewString()
	var uniqueToDelete []*snapshotv1.VirtualMachineSnapshot
	for _, snapshot := range toDelete {
		key := fmt.Sprintf("%s/%s", snapshot.Namespace, snapshot.Name)
		if !toDeleteSet.Has(key) {
			toDeleteSet.Insert(key)
			uniqueToDelete = append(uniqueToDelete, snapshot)
		}
	}

	// Delete snapshots
	for _, snapshot := range uniqueToDelete {
		err := ctrl.Client.VirtualMachineSnapshot(snapshot.Namespace).Delete(context.Background(), snapshot.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			log.Log.Errorf("Failed to delete snapshot %s/%s: %v", snapshot.Namespace, snapshot.Name, err)
			continue
		}
		log.Log.Infof("Deleted snapshot %s/%s due to retention policy", snapshot.Namespace, snapshot.Name)
	}

	return nil
}
