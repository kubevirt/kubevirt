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

// Package hibernation implements the VirtualMachine side of hibernation: the
// memory and device state of a running VirtualMachineInstance is saved to a
// PVC, the VirtualMachine stops, and its next start resumes from that state.
//
// The lifecycle is tracked in vm.Status.SavedState:
//
//	Pending -> Saving -> Saved -> Restoring -> Restored
//	   \          \         \          \
//	    `----------`---------`----------`--> Failed
//
//	Restoring -> RestoreFailed -> (start) Saved
//
// virt-api records the request (Pending, with the run strategy and the time
// of the request) and stops the VM: Always and Once switch to Halted, Manual
// and RerunOnFailure get a stop request marked as the one of the hibernation.
// The VM controller provisions the PVC and arms the save by hot plugging it
// into the VMI as the SavedState utility volume VolumeName. virt-handler
// commits to the save by persisting the Saving status on the VMI, and only
// then asks virt-launcher to save the domain. A cancel disarms the save by
// unplugging the volume in a patch conditional on the resource version of
// the VMI that showed no committed save: the save and the cancel serialize on
// the VMI, and exactly one of them wins.
//
// The stop of the VMI is held while the hibernation is Pending or Saving. A
// hibernation that is cancelled, times out or fails while the VMI still runs
// leaves it running: the stop request of the hibernation is withdrawn, and a
// run strategy it replaced is restored before the hold on the stop is
// released. A save target is only unplugged and deleted once virt-launcher
// reported the end of the save, or the VMI, and so QEMU, is gone. A deleted VM
// holds nothing.
//
// On the next start a valid saved state is set as the VMI SavedStateSource,
// with its PVC as the utility volume SourceVolumeName (the only utility
// volume a VMI may be created with, and only by KubeVirt). The VM moves to
// Restoring once the VMI was created. virt-handler commits to the restore by
// persisting the Restoring status on the VMI before it starts the domain:
// until then, the restore can be retried or abandoned without losing the
// state; from then on, the guest may run. Once resumed, the volume is
// unplugged and the PVC is deleted: a saved state can be used only once, since
// the disks diverge from it as soon as the guest runs. A restore that fails
// never boots the domain: the VM moves to RestoreFailed and keeps its saved
// state. A start retries the restore, a discard lets the VM boot.
//
// A resumed VMI can be hibernated again. It keeps its SavedStateSource and
// the restore result in its status, so every save is identified by the file
// name derived from its own PVC (FileNameFor), never by the presence of a
// saved state status. The volume a VMI resumes from (SourceVolumeName) never
// collides with the volume a new state is saved to (VolumeName).
package hibernation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/storage/utilityvolume"
)

const (
	// VolumeName is the name of the hot plugged utility volume the state is saved to.
	VolumeName = "saved-state-target"
	// legacyVolumeName is the utility volume name used by the first prototype.
	legacyVolumeName = "saved-state"
	// SourceVolumeName is the name of the hot plugged utility volume a VMI resumes from.
	SourceVolumeName = "saved-state-source"

	// SourceNodeAnnotation on the saved state PVC records the node the state
	// was saved on. A resume prefers that node.
	SourceNodeAnnotation = "kubevirt.io/saved-state-source-node"

	// PendingTimeout bounds the time from the request until the save starts:
	// provisioning, hot plugging and mounting the volume.
	PendingTimeout = 10 * time.Minute
	// requestGrace is how long the VM controller waits for virt-api to
	// complete a request, which records the hibernation and then stops the VM
	// in its spec. An incomplete request is cancelled.
	requestGrace = 30 * time.Second
	// minSavingTimeout and savingTimeoutPerGi bound the time the save takes.
	minSavingTimeout   = 10 * time.Minute
	savingTimeoutPerGi = time.Minute
	// savingTimeoutGrace is how much longer than virt-launcher the controller
	// waits for a save before it reports it stuck. virt-launcher aborts a save
	// that takes too long, which resumes the guest.
	savingTimeoutGrace = time.Minute
	// RestoreStartTimeout bounds the time from the creation of a VMI resuming
	// from a saved state until virt-handler commits to the restore: scheduling
	// it and hot plugging the volume. A VMI that committed is never abandoned,
	// whatever the time it takes to load the state.
	RestoreStartTimeout = 10 * time.Minute

	// ErrorReason is the reason of the VM failure condition set on a sync error.
	ErrorReason = "HibernationError"

	// Event reasons.
	HibernatedReason           = "Hibernated"
	HibernationFailedReason    = "HibernationFailed"
	HibernationCancelledReason = "HibernationCancelled"
	HibernationStuckReason     = "HibernationStuck"
	ResumingReason             = "ResumingFromSavedState"
	ResumedReason              = "ResumedFromSavedState"
	ResumeFailedReason         = "ResumeFromSavedStateFailed"
	SavedStateDiscardedReason  = "SavedStateDiscarded"

	claimNameSuffix = "-saved-state-"
	maxClaimName    = 253
	uidPrefixLength = 8
	randomSuffixLen = 5
	fileNameHashLen = 12

	// The saved state holds at most the plugged guest memory (only the pages
	// in use with the sparse format) plus the device state and, with
	// mapped-ram, a bitmap of the saved pages and alignment padding per RAM
	// block. The file system keeps room for its metadata and reserved blocks,
	// like the default file system overhead of CDI.
	stateOverhead           = 512 * 1024 * 1024
	stateOverheadRatio      = 0.02
	filesystemOverheadRatio = 0.06
)

// ClaimName returns the name of the PVC holding the state saved from vmi.
// The random part makes every hibernation request use a new PVC, so that a
// retry never reuses a PVC of a previous attempt that is still terminating.
func ClaimName(vmName string, vmiUID types.UID, random string) string {
	suffix := claimNameSuffix + shortUID(vmiUID)
	if random != "" {
		suffix += "-" + random
	}
	return truncate(vmName, maxClaimName-len(suffix)) + suffix
}

// FileNameFor returns the name of the file a state is saved to at the root
// of the PVC claimName. It identifies the save: a VMI status or launcher
// metadata reporting another file belongs to another save or to a restore.
func FileNameFor(claimName string) string {
	sum := sha256.Sum256([]byte(claimName))
	return "domain-" + hex.EncodeToString(sum[:])[:fileNameHashLen] + ".save"
}

// IsSavePhase reports whether phase belongs to a save, rather than a restore.
func IsSavePhase(phase v1.VirtualMachineInstanceSavedStatePhase) bool {
	switch phase {
	case v1.SavedStateSaving, v1.SavedStateSaved, v1.SavedStateSaveFailed:
		return true
	}
	return false
}

// IsSavedStateVolume reports whether volume is a volume a state is saved to
// or resumed from.
func IsSavedStateVolume(volume v1.UtilityVolume) bool {
	return volume.Type != nil && *volume.Type == v1.SavedState
}

// SourceVolume returns the utility volume a VMI resumes from the state saved in claimName.
func SourceVolume(claimName string) v1.UtilityVolume {
	return v1.UtilityVolume{
		Name:                              SourceVolumeName,
		PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
		Type:                              new(v1.SavedState),
	}
}

// IsSourceVolumeOf reports whether volume is the utility volume source resumes from.
func IsSourceVolumeOf(volume v1.UtilityVolume, source *v1.SavedStateSource) bool {
	return source != nil && volume.Name == SourceVolumeName && volume.ClaimName == source.ClaimName &&
		volume.Type != nil && *volume.Type == v1.SavedState
}

// IsSaveTargetVolume reports whether volume is a volume a state is saved to.
func IsSaveTargetVolume(volume v1.UtilityVolume) bool {
	return IsSavedStateVolume(volume) && volume.Name != SourceVolumeName
}

func shortUID(uid types.UID) string {
	return truncate(string(uid), uidPrefixLength)
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

// IsInProgress reports whether vmi must not be stopped yet: its state is
// being saved, the run strategy a failed hibernation of vmi replaced is still
// to be restored, or vmi reported the result of its restore and the VM did
// not record it yet. A VM being deleted holds nothing.
func IsInProgress(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) bool {
	state := vm.Status.SavedState
	if state == nil || vmi == nil || vm.DeletionTimestamp != nil {
		return false
	}
	switch {
	case vmi.UID == state.SourceUID:
		switch state.Phase {
		case v1.HibernationPending, v1.HibernationSaving:
			return true
		case v1.HibernationFailed:
			_, pending := runStrategyToRestore(vm, vmi)
			return pending
		}
	case hasUnrecordedResult(state, vmi):
		// Losing the result with the VMI would discard a saved state that may
		// still be valid.
		return true
	}
	return false
}

// hasUnrecordedResult reports whether vmi resumes from the saved state of
// state, reported the result of its restore, and state does not record it
// yet: either the VM is Restoring, or the update recording the creation of
// vmi was lost and the VM is still Saved. A VMI a retry superseded is done.
func hasUnrecordedResult(state *v1.VirtualMachineSavedState, vmi *v1.VirtualMachineInstance) bool {
	status := vmi.Status.SavedState
	if status == nil || (status.Phase != v1.SavedStateRestored && status.Phase != v1.SavedStateRestoreFailed) ||
		!isRestoringVMI(state, vmi) {
		return false
	}
	switch state.Phase {
	case v1.HibernationRestoring:
		return true
	case v1.HibernationSaved:
		return vmi.UID != state.SupersededUID
	}
	return false
}

// IsHibernating reports whether the state of vmi, the instance of vm, is
// being saved before it stops.
func IsHibernating(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) bool {
	state := vm.Status.SavedState
	return state != nil && vmi != nil && vmi.UID == state.SourceUID &&
		(state.Phase == v1.HibernationPending || state.Phase == v1.HibernationSaving)
}

// IsRestoreFailure reports whether vmi failed because resuming it from its
// saved state failed. The saved state failed to start, not the VM: the
// hibernation holds the next start, and a discard boots the VM right away.
func IsRestoreFailure(vmi *v1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.Spec.SavedStateSource != nil && vmi.Status.SavedState != nil &&
		vmi.Status.SavedState.Phase == v1.SavedStateRestoreFailed
}

// IsSupersededRestore reports whether vmi, a final VMI of vm that failed to
// resume from the saved state, is to be replaced: a start request retried the
// resume and recorded vmi as superseded, the saved state was discarded, or
// the feature gate is turned off. Its guest never ran, so it is replaced like
// a VMI that never started, whatever the run strategy: with Once or Manual, a
// final VMI would otherwise stay and hold the next start.
//
// Any other failed VMI is kept: its failure waits for a retry or a discard,
// or is still to be recorded, e.g. once the update recording its creation
// was lost and the VM is still Saved. Replacing such a VMI would retry the
// resume without a request, again and again while the update fails.
func IsSupersededRestore(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, gateEnabled bool) bool {
	if vmi == nil || !vmi.IsFinal() || vmi.DeletionTimestamp != nil || !IsRestoreFailure(vmi) {
		return false
	}
	state := vm.Status.SavedState
	switch {
	case state == nil, state.Discard, !gateEnabled:
		return true
	case state.Phase == v1.HibernationFailed:
		// The saved state was discarded, the VM boots.
		return true
	case state.Phase == v1.HibernationSaved:
		return state.SupersededUID != "" && state.SupersededUID == vmi.UID
	}
	return false
}

// SupersedesRestore reports whether a start request retrying the failed
// resume of vm supersedes vmi: vmi failed to resume from the saved state of
// vm, and the VM controller is to replace it once it is final.
func SupersedesRestore(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) bool {
	state := vm.Status.SavedState
	return state != nil && vmi != nil && vmi.DeletionTimestamp == nil && IsRestoreFailure(vmi) &&
		vmi.Spec.SavedStateSource.ClaimName == state.ClaimName
}

// BlocksStart returns why vm must not be started on its own, if it must not:
// resuming from its saved state failed, and the saved state was neither
// discarded nor retried with a start request.
func BlocksStart(vm *v1.VirtualMachine) (string, bool) {
	state := vm.Status.SavedState
	if state == nil || state.Phase != v1.HibernationRestoreFailed || state.Discard {
		return "", false
	}
	return fmt.Sprintf("VirtualMachine %s could not resume from its saved state (%s): start it to retry, or discard the saved state to boot it", vm.Name, state.Message), true
}

// isRunningSource reports whether vmi is the running VMI the state of vm is saved from.
func isRunningSource(state *v1.VirtualMachineSavedState, vmi *v1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.UID == state.SourceUID && vmi.IsRunning() && vmi.DeletionTimestamp == nil
}

// replacesRunStrategy reports whether a hibernation stops the VM by
// switching runStrategy to Halted, rather than with a stop request.
func replacesRunStrategy(runStrategy v1.VirtualMachineRunStrategy) bool {
	return runStrategy == v1.RunStrategyAlways || runStrategy == v1.RunStrategyOnce
}

// runStrategyToRestore returns the run strategy a failed hibernation of the
// running VMI replaced in the spec of vm, while the replacement is in place.
func runStrategyToRestore(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) (v1.VirtualMachineRunStrategy, bool) {
	state := vm.Status.SavedState
	if state.Phase != v1.HibernationFailed || state.RunStrategy == nil || !isRunningSource(state, vmi) ||
		!replacesRunStrategy(*state.RunStrategy) {
		return "", false
	}
	if current, err := vm.RunStrategy(); err != nil || current != v1.RunStrategyHalted {
		return "", false
	}
	return *state.RunStrategy, true
}

// restoreRunStrategy restores the run strategy a failed hibernation replaced,
// so that the VMI keeps running.
func restoreRunStrategy(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	runStrategy, ok := runStrategyToRestore(vm, vmi)
	if !ok {
		return
	}
	log.Log.Object(vm).Infof("Restoring the run strategy %s replaced by the failed hibernation", runStrategy)
	if vm.Spec.Running != nil {
		if runStrategy == v1.RunStrategyAlways {
			vm.Spec.Running = new(true)
			return
		}
		vm.Spec.Running = nil
	}
	vm.Spec.RunStrategy = new(runStrategy)
}

// StopRequest returns the stop request a hibernation of vmi to claimName adds
// to the VM, marked so that a cancel withdraws only that request.
func StopRequest(vmi *v1.VirtualMachineInstance, claimName string) v1.VirtualMachineStateChangeRequest {
	return v1.VirtualMachineStateChangeRequest{
		Action: v1.StopRequest,
		UID:    new(vmi.UID),
		Data:   map[string]string{v1.HibernationStopRequestDataKey: claimName},
	}
}

func isStopRequestOf(request v1.VirtualMachineStateChangeRequest, claimName string) bool {
	return request.Action == v1.StopRequest && request.Data[v1.HibernationStopRequestDataKey] == claimName
}

// withdrawStopRequest removes the stop request of the hibernation of vm. A
// stop request of the user is kept: the VM stops once the hibernation ended.
func withdrawStopRequest(vm *v1.VirtualMachine) {
	claimName := vm.Status.SavedState.ClaimName
	vm.Status.StateChangeRequests = slices.DeleteFunc(vm.Status.StateChangeRequests, func(request v1.VirtualMachineStateChangeRequest) bool {
		return isStopRequestOf(request, claimName)
	})
	if len(vm.Status.StateChangeRequests) == 0 {
		vm.Status.StateChangeRequests = nil
	}
}

// requestComplete reports whether virt-api completed the hibernation request
// of vm. It records the hibernation in the status, then switches the run
// strategy to Halted in the spec, two writes that cannot be atomic; a stop
// request is part of the status write.
func requestComplete(vm *v1.VirtualMachine) bool {
	state := vm.Status.SavedState
	if state.RunStrategy == nil || !replacesRunStrategy(*state.RunStrategy) {
		return true
	}
	current, err := vm.RunStrategy()
	return err == nil && current == v1.RunStrategyHalted
}

// ValidateRequest checks that the state of vmi, the instance of vm, can be saved.
func ValidateRequest(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	if vmi == nil || !vmi.IsRunning() || vmi.DeletionTimestamp != nil {
		return fmt.Errorf("VirtualMachine %s is not running", vm.Name)
	}
	if vmi.Spec.SavedStateSource != nil && (vmi.Status.SavedState == nil || vmi.Status.SavedState.Phase == v1.SavedStateRestoring) {
		return fmt.Errorf("VirtualMachine %s is still resuming from a saved state", vm.Name)
	}
	if !savedStateVolumesDetached(vmi) {
		return fmt.Errorf("the saved state volume of a previous hibernation of VirtualMachine %s is still attached", vm.Name)
	}
	if state := vm.Status.SavedState; state != nil {
		switch state.Phase {
		case v1.HibernationPending, v1.HibernationSaving:
			return fmt.Errorf("VirtualMachine %s is already hibernating", vm.Name)
		case v1.HibernationRestoring:
			return fmt.Errorf("VirtualMachine %s is still resuming from a saved state", vm.Name)
		case v1.HibernationFailed:
			if _, pending := runStrategyToRestore(vm, vmi); pending {
				return fmt.Errorf("the run strategy of VirtualMachine %s is still being restored after a failed hibernation", vm.Name)
			}
		}
	}
	if vmi.Status.MigrationState != nil && !vmi.Status.MigrationState.Completed && !vmi.Status.MigrationState.Failed {
		return fmt.Errorf("VirtualMachine %s is migrating", vm.Name)
	}
	if hasCondition(vm.Status.Conditions, v1.VirtualMachineRestartRequired) {
		return fmt.Errorf("VirtualMachine %s has changes that require a restart, its next start cannot resume from a saved state", vm.Name)
	}
	if err := checkSupportedDevices(vmi); err != nil {
		return err
	}
	if err := checkMigratable(vmi); err != nil {
		return err
	}
	if err := checkPersistentDisks(vmi); err != nil {
		return err
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume != nil && !isUtilityVolume(vmi, volumeStatus.Name) {
			return fmt.Errorf("VirtualMachine %s has the hotplugged volume %s, hibernation does not support hotplugged volumes yet", vm.Name, volumeStatus.Name)
		}
	}
	return nil
}

// checkSupportedDevices rejects devices whose state a save does not cover, or
// covers in ways not validated yet, whatever the LiveMigratable condition
// says: host devices and SR-IOV are bound to a node, and the vTPM and the
// persistent EFI variables live outside the memory state.
func checkSupportedDevices(vmi *v1.VirtualMachineInstance) error {
	devices := vmi.Spec.Domain.Devices
	unsupported := func(what string) error {
		return fmt.Errorf("VirtualMachineInstance %s has %s, which hibernation does not support yet", vmi.Name, what)
	}
	switch {
	case len(devices.GPUs) > 0:
		return unsupported("GPUs")
	case len(devices.HostDevices) > 0:
		return unsupported("host devices")
	case devices.TPM != nil && (devices.TPM.Enabled == nil || *devices.TPM.Enabled):
		return unsupported("a vTPM")
	}
	if firmware := vmi.Spec.Domain.Firmware; firmware != nil && firmware.Bootloader != nil && firmware.Bootloader.EFI != nil &&
		firmware.Bootloader.EFI.Persistent != nil && *firmware.Bootloader.EFI.Persistent {
		return unsupported("persistent EFI variables")
	}
	for _, iface := range devices.Interfaces {
		if iface.SRIOV != nil {
			return unsupported(fmt.Sprintf("the SR-IOV interface %s", iface.Name))
		}
	}
	return nil
}

// checkMigratable rejects a VMI whose device state cannot be migrated, as saving
// it is a migration to a file. Disks that cannot be live migrated are fine, the
// domain is stopped before they are used elsewhere.
func checkMigratable(vmi *v1.VirtualMachineInstance) error {
	for _, condition := range vmi.Status.Conditions {
		if condition.Type != v1.VirtualMachineInstanceIsMigratable || condition.Status != k8sv1.ConditionFalse {
			continue
		}
		if condition.Reason == v1.VirtualMachineInstanceReasonDisksNotMigratable {
			return nil
		}
		return fmt.Errorf("the state of VirtualMachineInstance %s cannot be saved: %s", vmi.Name, condition.Message)
	}
	return nil
}

// checkPersistentDisks rejects a VMI with a writable disk that does not
// persist across a stop: the saved state would resume on a disk that lost
// what the guest wrote to it.
func checkPersistentDisks(vmi *v1.VirtualMachineInstance) error {
	cdroms := map[string]bool{}
	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		if disk.CDRom != nil {
			cdroms[disk.Name] = true
		}
	}
	for _, volume := range vmi.Spec.Volumes {
		ephemeral := volume.Ephemeral != nil || volume.EmptyDisk != nil || (volume.ContainerDisk != nil && !cdroms[volume.Name])
		if ephemeral {
			return fmt.Errorf("the disk of volume %s of VirtualMachineInstance %s does not persist across a stop, hibernation needs persistent disks", volume.Name, vmi.Name)
		}
	}
	return nil
}

func hasCondition(conditions []v1.VirtualMachineCondition, conditionType v1.VirtualMachineConditionType) bool {
	return slices.ContainsFunc(conditions, func(condition v1.VirtualMachineCondition) bool {
		return condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue
	})
}

func isUtilityVolume(vmi *v1.VirtualMachineInstance, name string) bool {
	return slices.ContainsFunc(vmi.Spec.UtilityVolumes, func(volume v1.UtilityVolume) bool {
		return volume.Name == name
	})
}

// NewSavedState returns the status of a hibernation requested for vmi.
func NewSavedState(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, options *v1.HibernateOptions) *v1.VirtualMachineSavedState {
	now := metav1.Now()
	state := &v1.VirtualMachineSavedState{
		Phase:            v1.HibernationPending,
		ClaimName:        ClaimName(vm.Name, vmi.UID, rand.String(randomSuffixLen)),
		SourceUID:        vmi.UID,
		RequestTimestamp: &now,
	}
	if runStrategy, err := vm.RunStrategy(); err == nil {
		state.RunStrategy = new(runStrategy)
	}
	if options != nil && options.StorageClassName != nil {
		state.StorageClassName = new(*options.StorageClassName)
	}
	return state
}

// SpecHash identifies the parts of the VirtualMachine spec that define the
// hardware of its instances. A state cannot be resumed with another hardware.
func SpecHash(vm *v1.VirtualMachine) (string, error) {
	identity := struct {
		Template     *v1.VirtualMachineInstanceSpec `json:"template,omitempty"`
		Instancetype *v1.InstancetypeMatcher        `json:"instancetype,omitempty"`
		Preference   *v1.PreferenceMatcher          `json:"preference,omitempty"`
		DataVolumes  []v1.DataVolumeTemplateSpec    `json:"dataVolumeTemplates,omitempty"`
	}{
		Instancetype: vm.Spec.Instancetype,
		Preference:   vm.Spec.Preference,
		DataVolumes:  vm.Spec.DataVolumeTemplates,
	}
	if vm.Spec.Template != nil {
		identity.Template = &vm.Spec.Template.Spec
	}
	data, err := json.Marshal(identity)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// VolumeClaimUIDs returns the UIDs of the PVCs backing the volumes, so that a
// PVC replaced while the VM is hibernated, e.g. by a snapshot restore, is detected.
func VolumeClaimUIDs(namespace string, volumes []v1.Volume, pvcStore cache.Store) (map[string]types.UID, error) {
	uids := map[string]types.UID{}
	for _, volume := range volumes {
		claimName := volumeClaimName(volume)
		if claimName == "" {
			continue
		}
		obj, exists, err := pvcStore.GetByKey(namespace + "/" + claimName)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		uids[claimName] = obj.(*k8sv1.PersistentVolumeClaim).UID
	}
	return uids, nil
}

func templateVolumes(vm *v1.VirtualMachine) []v1.Volume {
	if vm.Spec.Template == nil {
		return nil
	}
	return vm.Spec.Template.Spec.Volumes
}

func volumeClaimName(volume v1.Volume) string {
	switch {
	case volume.PersistentVolumeClaim != nil:
		return volume.PersistentVolumeClaim.ClaimName
	case volume.DataVolume != nil:
		return volume.DataVolume.Name
	}
	return ""
}

// HostModelNodeSelector returns the labels a node needs to run the host-model
// CPU of node, mirroring what a live migration target requires.
func HostModelNodeSelector(vmi *v1.VirtualMachineInstance, node *k8sv1.Node) map[string]string {
	if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.Model != "" && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostModel {
		return nil
	}
	selector := map[string]string{}
	for key, value := range node.Labels {
		if model, ok := strings.CutPrefix(key, v1.HostModelCPULabel); ok {
			selector[v1.SupportedHostModelMigrationCPU+model] = value
		}
		if feature, ok := strings.CutPrefix(key, v1.HostModelRequiredFeaturesLabel); ok {
			selector[v1.CPUFeatureLabel+feature] = value
		}
	}
	if len(selector) == 0 {
		return nil
	}
	return selector
}

// ClaimSize returns the size of a PVC that can hold the state of vmi.
func ClaimSize(vmi *v1.VirtualMachineInstance) resource.Quantity {
	memory := guestMemory(vmi)
	state := float64(memory.Value())*(1+stateOverheadRatio) + stateOverhead
	size := int64(state / (1 - filesystemOverheadRatio))
	const mi = 1024 * 1024
	size = (size + mi - 1) / mi * mi
	return *resource.NewQuantity(size, resource.BinarySI)
}

// guestMemory returns the memory plugged in the guest. With memory hotplug, the
// unplugged part of the hotpluggable region is never written: the save image
// is sparse and its logical size can exceed the PVC on file systems that
// support sparse files.
func guestMemory(vmi *v1.VirtualMachineInstance) resource.Quantity {
	if vmi.Status.Memory != nil && vmi.Status.Memory.GuestCurrent != nil {
		return *vmi.Status.Memory.GuestCurrent
	}
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return *vmi.Spec.Domain.Memory.Guest
	}
	return *vmi.Spec.Domain.Resources.Requests.Memory()
}

// SavingTimeout returns how long saving the state of vmi may take, after
// which virt-launcher aborts the save and the guest keeps running.
func SavingTimeout(vmi *v1.VirtualMachineInstance) time.Duration {
	if value, ok := vmi.Annotations[v1.SavedStateTimeoutAnnotation]; ok {
		timeout, err := time.ParseDuration(value)
		if err == nil && timeout > 0 {
			return timeout
		}
		log.Log.Object(vmi).Warningf("Ignoring invalid %s annotation %q", v1.SavedStateTimeoutAnnotation, value)
	}
	memory := guestMemory(vmi)
	gi := (memory.Value() + 1<<30 - 1) >> 30
	return max(minSavingTimeout, time.Duration(gi)*savingTimeoutPerGi)
}

// Controller performs the hibernation steps of the VirtualMachine controller.
type Controller struct {
	client   kubecli.KubevirtClient
	pvcStore cache.Store
	recorder record.EventRecorder
	now      func() time.Time
}

func NewController(client kubecli.KubevirtClient, pvcStore cache.Store, recorder record.EventRecorder) *Controller {
	return &Controller{client: client, pvcStore: pvcStore, recorder: recorder, now: time.Now}
}

// Sync provisions, arms, disarms, unplugs and deletes the saved state PVC of
// vm according to the hibernation phase, restores the run strategy a failed
// hibernation replaced in the spec of vm, and deletes a VMI that gave up
// resuming before it committed to the restore. It does not change the status
// of vm, except for its spec.
func (c *Controller) Sync(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	state := vm.Status.SavedState
	if state == nil {
		return nil
	}
	if err := c.deleteStaleClaims(vm, state.ClaimName); err != nil {
		return err
	}
	switch state.Phase {
	case v1.HibernationPending:
		if !isRunningSource(state, vmi) || saveStatus(state, vmi) != nil {
			return nil
		}
		if _, cancelled := c.cancelReason(vm); cancelled {
			return c.disarm(vmi)
		}
		if !requestComplete(vm) {
			return nil
		}
		claim, err := c.getClaim(vm.Namespace, state.ClaimName)
		if err != nil {
			return err
		}
		if claim == nil {
			return c.createClaim(vm, vmi, state)
		}
		if !metav1.IsControlledBy(claim, vm) {
			return fmt.Errorf("PVC %s/%s exists and is not owned by VirtualMachine %s", vm.Namespace, state.ClaimName, vm.Name)
		}
		if state.SpecHash == "" {
			// UpdateStatus records what the state is valid with before it is armed.
			return nil
		}
		return c.attach(vmi, VolumeName, state.ClaimName)
	case v1.HibernationRestoring:
		if !isRestoringVMI(state, vmi) || vmi.DeletionTimestamp != nil || vmi.IsFinal() {
			return nil
		}
		if _, abandoned := c.restoreAbandoned(state, vmi); abandoned {
			return c.deleteUncommittedVMI(vmi)
		}
		return c.attach(vmi, SourceVolumeName, state.ClaimName)
	case v1.HibernationFailed, v1.HibernationRestored:
		// Keep the VMI of a failed hibernation running.
		restoreRunStrategy(vm, vmi)
		if vmi != nil && (vmi.UID == state.SourceUID || vmi.UID == state.RestoreUID) && !vmi.IsFinal() &&
			vmi.DeletionTimestamp == nil && !savedStateVolumesDetached(vmi) {
			// Deleting the PVC before it is detached leaves it hanging on its protection finalizer.
			return c.detach(vmi)
		}
		return c.deleteClaim(vm, state.ClaimName)
	case v1.HibernationSaved, v1.HibernationRestoreFailed:
		if state.Discard {
			return c.deleteClaim(vm, state.ClaimName)
		}
	}
	// Saving: the volume stays attached until virt-launcher reports the end of
	// the save, or the VMI is gone.
	return nil
}

// cancelReason returns why the Pending hibernation of vm is cancelled, if it
// is: a discard request, a request virt-api did not complete, or no save
// within PendingTimeout of the request.
func (c *Controller) cancelReason(vm *v1.VirtualMachine) (string, bool) {
	state := vm.Status.SavedState
	if state.Discard {
		return "The hibernation was cancelled before the state was saved", true
	}
	requested := c.requestTime(vm)
	if requested == nil {
		return "", false
	}
	waited := c.now().Sub(requested.Time)
	if !requestComplete(vm) && waited > requestGrace {
		return "The hibernation request was not completed, the VirtualMachine was not stopped", true
	}
	if waited > PendingTimeout {
		return fmt.Sprintf("The state saving did not start within %s of the hibernation request", PendingTimeout), true
	}
	return "", false
}

// requestTime returns when the hibernation of vm was requested, or for a
// request that predates the request timestamp, when its PVC was created.
func (c *Controller) requestTime(vm *v1.VirtualMachine) *metav1.Time {
	state := vm.Status.SavedState
	if state.RequestTimestamp != nil {
		return state.RequestTimestamp
	}
	if claim, err := c.getClaim(vm.Namespace, state.ClaimName); err == nil && claim != nil && !claim.CreationTimestamp.IsZero() {
		return &claim.CreationTimestamp
	}
	return nil
}

// armed reports whether the save of the hibernation of state is armed on
// vmi: virt-handler may commit to it.
func armed(state *v1.VirtualMachineSavedState, vmi *v1.VirtualMachineInstance) bool {
	return slices.ContainsFunc(vmi.Spec.UtilityVolumes, func(volume v1.UtilityVolume) bool {
		return IsSaveTargetVolume(volume) && volume.ClaimName == state.ClaimName
	})
}

// disarm unplugs the save target of a cancelled hibernation from vmi, unless
// virt-handler committed to the save first. virt-handler persists the Saving
// status on the VMI before it asks virt-launcher to save, and the patch is
// conditional on the resource version of the VMI that showed no such status:
// whichever writes first wins, and the other one fails.
func (c *Controller) disarm(vmi *v1.VirtualMachineInstance) error {
	if !slices.ContainsFunc(vmi.Spec.UtilityVolumes, IsSaveTargetVolume) {
		return nil
	}
	volumes := slices.DeleteFunc(slices.Clone(vmi.Spec.UtilityVolumes), IsSaveTargetVolume)
	patchSet := patch.New(patch.WithTest("/metadata/uid", vmi.UID))
	if vmi.ResourceVersion != "" {
		patchSet.AddOption(patch.WithTest("/metadata/resourceVersion", vmi.ResourceVersion))
	}
	if len(volumes) == 0 {
		patchSet.AddOption(patch.WithRemove("/spec/utilityVolumes"))
	} else {
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", volumes))
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = c.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to disarm the save of the cancelled hibernation: %w", err)
	}
	return nil
}

// isRestoringVMI reports whether vmi resumes from the saved state of state.
func isRestoringVMI(state *v1.VirtualMachineSavedState, vmi *v1.VirtualMachineInstance) bool {
	return vmi != nil && vmi.Spec.SavedStateSource != nil && vmi.Spec.SavedStateSource.ClaimName == state.ClaimName &&
		(state.RestoreUID == "" || state.RestoreUID == vmi.UID)
}

// restoreCommitted reports whether virt-handler committed vmi to the restore:
// from then on, its guest may run.
func restoreCommitted(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Status.SavedState != nil
}

// restoreAbandoned returns why vmi gives up resuming from the saved state, if
// it does. Only a VMI that did not commit to the restore gives up: the saved
// state was discarded, or it did not commit within RestoreStartTimeout of its
// creation, e.g. because no node can run it.
func (c *Controller) restoreAbandoned(state *v1.VirtualMachineSavedState, vmi *v1.VirtualMachineInstance) (string, bool) {
	if restoreCommitted(vmi) {
		return "", false
	}
	if state.Discard {
		return "The saved state was discarded before the VirtualMachine resumed from it", true
	}
	if vmi.CreationTimestamp.IsZero() || c.now().Sub(vmi.CreationTimestamp.Time) <= RestoreStartTimeout {
		return "", false
	}
	message := fmt.Sprintf("The VirtualMachineInstance did not start resuming from the saved state within %s", RestoreStartTimeout)
	if reason := unschedulableReason(vmi); reason != "" {
		message += ": " + reason
	}
	return message, true
}

// unschedulableReason returns why no node can run vmi, if none can.
func unschedulableReason(vmi *v1.VirtualMachineInstance) string {
	for _, condition := range vmi.Status.Conditions {
		if condition.Type == v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled) &&
			condition.Status == k8sv1.ConditionFalse && condition.Reason == k8sv1.PodReasonUnschedulable {
			return fmt.Sprintf("no node can resume the saved state: %s", condition.Message)
		}
	}
	return ""
}

// deleteUncommittedVMI stops vmi, which gave up resuming before it committed
// to the restore. The deletion is conditional on the resource version that
// showed no commit, so that a concurrent commit of virt-handler wins.
func (c *Controller) deleteUncommittedVMI(vmi *v1.VirtualMachineInstance) error {
	log.Log.Object(vmi).Info("Stopping the VirtualMachineInstance that gave up resuming from the saved state")
	preconditions := &metav1.Preconditions{UID: &vmi.UID}
	if vmi.ResourceVersion != "" {
		preconditions.ResourceVersion = &vmi.ResourceVersion
	}
	err := c.client.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{
		Preconditions: preconditions,
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *Controller) getClaim(namespace, name string) (*k8sv1.PersistentVolumeClaim, error) {
	obj, exists, err := c.pvcStore.GetByKey(namespace + "/" + name)
	if err != nil || !exists {
		return nil, err
	}
	return obj.(*k8sv1.PersistentVolumeClaim), nil
}

// deleteStaleClaims deletes the saved state PVCs of vm other than the one of
// its current hibernation, e.g. the PVC a resumed VMI was restored from when
// it is hibernated again before that PVC was deleted.
func (c *Controller) deleteStaleClaims(vm *v1.VirtualMachine, current string) error {
	var objs []interface{}
	if indexer, ok := c.pvcStore.(cache.Indexer); ok {
		var err error
		if objs, err = indexer.ByIndex(cache.NamespaceIndex, vm.Namespace); err != nil {
			objs = indexer.List()
		}
	} else {
		objs = c.pvcStore.List()
	}
	for _, obj := range objs {
		claim := obj.(*k8sv1.PersistentVolumeClaim)
		if claim.Namespace != vm.Namespace || claim.Name == current || claim.DeletionTimestamp != nil ||
			claim.Labels[v1.AppLabel] != appLabelValue || claim.Labels[v1.VirtualMachineLabel] != vm.Name ||
			!metav1.IsControlledBy(claim, vm) {
			continue
		}
		log.Log.Object(vm).Infof("Deleting the stale saved state PVC %s", claim.Name)
		if err := c.deleteClaim(vm, claim.Name); err != nil {
			return err
		}
	}
	return nil
}

const appLabelValue = "hibernation"

func (c *Controller) createClaim(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, state *v1.VirtualMachineSavedState) error {
	claim := &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      state.ClaimName,
			Namespace: vm.Namespace,
			Labels: map[string]string{
				v1.AppLabel:            appLabelValue,
				v1.VirtualMachineLabel: vm.Name,
			},
			Annotations: map[string]string{
				SourceNodeAnnotation: vmi.Status.NodeName,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vm, v1.VirtualMachineGroupVersionKind),
			},
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes:      []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			VolumeMode:       new(k8sv1.PersistentVolumeFilesystem),
			StorageClassName: state.StorageClassName,
			Resources: k8sv1.VolumeResourceRequirements{
				Requests: k8sv1.ResourceList{k8sv1.ResourceStorage: ClaimSize(vmi)},
			},
		},
	}
	_, err := c.client.CoreV1().PersistentVolumeClaims(vm.Namespace).Create(context.Background(), claim, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (c *Controller) attach(vmi *v1.VirtualMachineInstance, volumeName, claimName string) error {
	found, err := utilityvolume.Find(vmi, volumeName, claimName, v1.SavedState)
	if err != nil || found {
		return err
	}
	patchBytes, err := utilityvolume.AddPatch(vmi, volumeName, claimName, v1.SavedState)
	if err != nil {
		return err
	}
	_, err = c.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to attach the saved state volume: %w", err)
	}
	return nil
}

// savedStateVolumesDetached reports whether no volume a state is saved to or
// resumed from is attached to vmi, including the legacy volume name of the
// first prototype.
func savedStateVolumesDetached(vmi *v1.VirtualMachineInstance) bool {
	if slices.ContainsFunc(vmi.Spec.UtilityVolumes, IsSavedStateVolume) {
		return false
	}
	return utilityvolume.Detached(vmi, VolumeName) && utilityvolume.Detached(vmi, SourceVolumeName) &&
		utilityvolume.Detached(vmi, legacyVolumeName)
}

func (c *Controller) detach(vmi *v1.VirtualMachineInstance) error {
	if !slices.ContainsFunc(vmi.Spec.UtilityVolumes, IsSavedStateVolume) {
		return nil
	}
	volumes := slices.DeleteFunc(slices.Clone(vmi.Spec.UtilityVolumes), IsSavedStateVolume)
	patchSet := patch.New(
		patch.WithTest("/metadata/uid", vmi.UID),
		patch.WithTest("/spec/utilityVolumes", vmi.Spec.UtilityVolumes),
	)
	if len(volumes) == 0 {
		patchSet.AddOption(patch.WithRemove("/spec/utilityVolumes"))
	} else {
		patchSet.AddOption(patch.WithReplace("/spec/utilityVolumes", volumes))
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}
	_, err = c.client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("unable to detach the saved state volume: %w", err)
	}
	return nil
}

func (c *Controller) deleteClaim(vm *v1.VirtualMachine, claimName string) error {
	claim, err := c.getClaim(vm.Namespace, claimName)
	if err != nil || claim == nil || claim.DeletionTimestamp != nil {
		return err
	}
	if !metav1.IsControlledBy(claim, vm) {
		// Never delete a PVC this VirtualMachine does not own.
		return nil
	}
	err = c.client.CoreV1().PersistentVolumeClaims(vm.Namespace).Delete(context.Background(), claimName, metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &claim.UID},
	})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

// UpdateStatus moves the hibernation of vm to its next phase according to
// vmi, the current instance of vm, if any.
func (c *Controller) UpdateStatus(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if state == nil {
		return
	}
	switch state.Phase {
	case v1.HibernationPending, v1.HibernationSaving:
		c.updateSaving(vm, vmi)
	case v1.HibernationSaved:
		c.updateSaved(vm, vmi)
	case v1.HibernationRestoring:
		c.updateRestoring(vm, vmi)
	case v1.HibernationRestoreFailed:
		c.updateRestoreFailed(vm, vmi)
	case v1.HibernationFailed:
		if _, pending := runStrategyToRestore(vm, vmi); pending {
			// Sync restores it first, the record must outlive the PVC.
			return
		}
		// Restored, or nothing to restore: a later stop of the VM is not undone.
		state.RunStrategy = nil
		c.forgetDiscarded(vm)
	case v1.HibernationRestored:
		c.forgetDiscarded(vm)
	}
}

// forgetDiscarded removes a discarded saved state from the status of vm once
// its PVC is gone.
func (c *Controller) forgetDiscarded(vm *v1.VirtualMachine) {
	state := vm.Status.SavedState
	if state.Discard && c.claimGone(vm.Namespace, state.ClaimName) {
		vm.Status.SavedState = nil
	}
}

// saveStatus returns the save to the PVC of state that vmi reports, if any.
// A resumed VMI reports its restore, and a VMI whose earlier hibernation
// failed reports that save: only a save to this PVC counts.
func saveStatus(state *v1.VirtualMachineSavedState, vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstanceSavedStateStatus {
	if vmi == nil {
		return nil
	}
	status := vmi.Status.SavedState
	if status == nil || !IsSavePhase(status.Phase) || status.FileName != FileNameFor(state.ClaimName) {
		return nil
	}
	return status
}

func (c *Controller) updateSaving(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if vmi == nil || vmi.UID != state.SourceUID {
		c.fail(vm, "The VirtualMachineInstance stopped before its state was saved")
		return
	}
	status := saveStatus(state, vmi)
	if status == nil {
		c.updateUncommitted(vm, vmi)
		return
	}
	// virt-handler committed to the save, which a cancel cannot undo anymore.
	if state.Discard {
		state.Discard = false
		state.Message = "The hibernation could not be cancelled, the state was already being saved"
	}
	state.FileName = status.FileName
	state.SaveStartTimestamp = status.StartTimestamp
	switch status.Phase {
	case v1.SavedStateSaving:
		state.Phase = v1.HibernationSaving
		if vmi.IsFinal() || vmi.DeletionTimestamp != nil {
			// QEMU is gone with the VMI, nothing writes to the PVC anymore.
			c.fail(vm, "The VirtualMachineInstance stopped while its state was being saved")
			return
		}
		c.checkSaveDuration(vm, vmi)
	case v1.SavedStateSaved:
		state.Phase = v1.HibernationSaved
		state.SaveEndTimestamp = status.EndTimestamp
		state.Message = ""
		c.recorder.Eventf(vm, k8sv1.EventTypeNormal, HibernatedReason, "Saved the state of VirtualMachineInstance %s to PVC %s%s%s",
			vmi.Name, state.ClaimName, durationSuffix(status.StartTimestamp, status.EndTimestamp), c.attachSuffix(vm.Namespace, state))
	case v1.SavedStateSaveFailed:
		state.SaveEndTimestamp = status.EndTimestamp
		c.failSave(vm, vmi, fmt.Sprintf("Saving the state failed: %s", status.Message))
	}
}

// updateUncommitted moves a hibernation virt-handler did not commit to yet.
// It is cancelled once its save is disarmed.
func (c *Controller) updateUncommitted(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if vmi.IsFinal() || vmi.DeletionTimestamp != nil {
		c.fail(vm, "The VirtualMachineInstance stopped before its state was saved")
		return
	}
	if reason, cancelled := c.cancelReason(vm); cancelled {
		if armed(state, vmi) {
			// Sync disarms the save, unless virt-handler commits to it first.
			state.Message = reason + ", waiting for the save to be disarmed"
			return
		}
		if state.Discard {
			c.cancel(vm, vmi, reason)
		} else {
			c.failSave(vm, vmi, reason)
		}
		return
	}
	if state.SpecHash == "" && requestComplete(vm) {
		if err := c.recordSource(vm, vmi); err != nil {
			// The save is armed only once the source is recorded.
			log.Log.Object(vm).Reason(err).Error("Failed to record the hibernation source")
			c.failSave(vm, vmi, fmt.Sprintf("Failed to record the hibernation source: %v", err))
		}
	}
}

// checkSaveDuration reports a save that takes longer than virt-launcher allows
// it. The hibernation keeps holding the stop of the VMI and the PVC stays
// attached: QEMU may still write to it, or the guest may be paused by the
// save. virt-launcher aborts the save, and an unresponsive virt-launcher fails
// the VMI, either of which ends the hibernation.
func (c *Controller) checkSaveDuration(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if state.SaveStartTimestamp == nil {
		return
	}
	timeout := SavingTimeout(vmi) + savingTimeoutGrace
	if c.now().Sub(state.SaveStartTimestamp.Time) <= timeout {
		return
	}
	message := fmt.Sprintf("Saving the state did not complete within %s, waiting for virt-launcher to report the end of the save", timeout)
	if state.Message != message {
		c.recorder.Event(vm, k8sv1.EventTypeWarning, HibernationStuckReason, message)
		state.Message = message
	}
}

// failSave fails a hibernation before its state was saved. A VMI that still
// runs keeps running: the stop request of the hibernation is withdrawn here,
// and a run strategy it replaced is restored by Sync before the stop of the
// VMI is released.
func (c *Controller) failSave(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, message string) {
	if isRunningSource(vm.Status.SavedState, vmi) {
		withdrawStopRequest(vm)
		message += ", the VirtualMachine keeps running"
	}
	c.fail(vm, message)
}

// cancel ends a hibernation that was cancelled before its state was saved.
// The VMI keeps running, as after a failed save, which Sync completes.
func (c *Controller) cancel(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, message string) {
	state := vm.Status.SavedState
	if isRunningSource(state, vmi) {
		withdrawStopRequest(vm)
		message += ", the VirtualMachine keeps running"
	}
	if state.Phase != v1.HibernationFailed {
		c.recorder.Event(vm, k8sv1.EventTypeNormal, HibernationCancelledReason, message)
	}
	state.Phase = v1.HibernationFailed
	state.Message = message
}

// attachSuffix reports how long it took from the creation of the PVC to the
// start of the save: provisioning, hot plugging and mounting the volume.
func (c *Controller) attachSuffix(namespace string, state *v1.VirtualMachineSavedState) string {
	claim, err := c.getClaim(namespace, state.ClaimName)
	if err != nil || claim == nil || claim.CreationTimestamp.IsZero() || state.SaveStartTimestamp == nil {
		return ""
	}
	return fmt.Sprintf(", the volume was ready %s after the PVC was created", state.SaveStartTimestamp.Sub(claim.CreationTimestamp.Time))
}

// recordSource records what the saved state is only valid with: the hardware
// spec, the PVCs of the disks and the CPU of the node.
func (c *Controller) recordSource(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	state := vm.Status.SavedState
	if hasCondition(vm.Status.Conditions, v1.VirtualMachineRestartRequired) {
		return fmt.Errorf("the VirtualMachine has changes that require a restart")
	}
	hash, err := SpecHash(vm)
	if err != nil {
		return err
	}
	uids, err := VolumeClaimUIDs(vm.Namespace, templateVolumes(vm), c.pvcStore)
	if err != nil {
		return err
	}
	if vmi.Status.NodeName != "" {
		node, err := c.client.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
		if err != nil {
			return err
		}
		state.SourceNodeSelector = HostModelNodeSelector(vmi, node)
	}
	state.SpecHash = hash
	state.VolumeClaimUIDs = uids
	return nil
}

func (c *Controller) updateSaved(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if state.Discard {
		c.forgetDiscarded(vm)
		return
	}
	if vmi == nil || vmi.UID == state.SourceUID {
		return
	}
	if isRestoringVMI(state, vmi) {
		if vmi.UID == state.SupersededUID {
			// A retry superseded vmi, the VM controller replaces it.
			return
		}
		if restoreCommitted(vmi) || (!vmi.IsFinal() && vmi.DeletionTimestamp == nil) {
			// The update recording the creation of vmi was lost: record it,
			// and the result vmi may have reported. A VMI that stopped before
			// it committed changed nothing, the state stays Saved.
			c.RestoreCreated(vm, vmi)
			c.updateRestoring(vm, vmi)
		}
		return
	}
	c.bootedWithout(vm, vmi)
}

// bootedWithout discards the saved state of vm, since vmi was started without
// resuming from it, e.g. with the feature gate turned off: the guest changes
// the disks behind it.
func (c *Controller) bootedWithout(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	c.fail(vm, fmt.Sprintf("VirtualMachineInstance %s was started without resuming from the saved state, which was discarded", vmi.Name))
}

func (c *Controller) updateRestoring(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if vmi == nil {
		// The VMI may not be in the cache yet. A VMI that is gone is handled by
		// the next start, which knows whether it committed to the restore.
		return
	}
	if state.RestoreUID == "" && isRestoringVMI(state, vmi) {
		state.RestoreUID = vmi.UID
	}
	if vmi.UID != state.RestoreUID {
		if vmi.Spec.SavedStateSource == nil {
			c.bootedWithout(vm, vmi)
		}
		return
	}
	status := vmi.Status.SavedState
	if status != nil {
		if state.RestoreStartTimestamp == nil {
			state.RestoreStartTimestamp = status.StartTimestamp
			if state.RestoreStartTimestamp == nil {
				state.RestoreStartTimestamp = new(metav1.NewTime(c.now()))
			}
		}
		switch status.Phase {
		case v1.SavedStateRestored:
			state.Phase = v1.HibernationRestored
			state.RestoreEndTimestamp = status.EndTimestamp
			state.Message = ""
			c.recorder.Eventf(vm, k8sv1.EventTypeNormal, ResumedReason, "VirtualMachineInstance %s resumed from the saved state%s",
				vmi.Name, durationSuffix(status.StartTimestamp, status.EndTimestamp))
			return
		case v1.SavedStateRestoreFailed:
			state.RestoreEndTimestamp = status.EndTimestamp
			c.restoreFailed(vm, fmt.Sprintf("Resuming from the saved state failed: %s", status.Message))
			return
		}
	}
	if vmi.IsFinal() || vmi.DeletionTimestamp != nil {
		if restoreCommitted(vmi) {
			// The guest may have run, the disks may not match the saved state anymore.
			c.fail(vm, "The VirtualMachineInstance stopped while resuming from the saved state, which was discarded")
			return
		}
		c.restoreNotStarted(vm, "The VirtualMachineInstance stopped before it started resuming from the saved state, which is kept")
		return
	}
	if message, abandoned := c.restoreAbandoned(state, vmi); abandoned {
		if state.Discard {
			c.fail(vm, message)
		} else {
			c.restoreFailed(vm, message+". The saved state is kept")
		}
		return
	}
	switch reason := unschedulableReason(vmi); {
	case reason != "":
		state.Message = "Waiting for a node that can resume the saved state, " + reason
	case restoreCommitted(vmi) && state.RestoreStartTimestamp != nil && c.now().Sub(state.RestoreStartTimestamp.Time) > SavingTimeout(vmi):
		state.Message = fmt.Sprintf("Resuming from the saved state takes longer than %s", SavingTimeout(vmi))
	default:
		state.Message = ""
	}
}

// restoreNotStarted moves vm back to Saved: its restore VMI stopped before it
// committed to the restore, the saved state is still valid.
func (c *Controller) restoreNotStarted(vm *v1.VirtualMachine, message string) {
	state := vm.Status.SavedState
	state.Phase = v1.HibernationSaved
	state.RestoreUID = ""
	state.RestoreStartTimestamp = nil
	state.Message = message
}

func (c *Controller) updateRestoreFailed(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if state.Discard {
		c.forgetDiscarded(vm)
		return
	}
	if vmi != nil && vmi.UID != state.RestoreUID && vmi.UID != state.SourceUID && !isRestoringVMI(state, vmi) {
		c.bootedWithout(vm, vmi)
	}
}

// restoreFailed keeps the saved state of a VM whose restore failed before the
// guest ran, and holds the next start until the saved state is discarded or
// a start retries the restore.
func (c *Controller) restoreFailed(vm *v1.VirtualMachine, message string) {
	state := vm.Status.SavedState
	if state.Phase != v1.HibernationRestoreFailed {
		c.recorder.Event(vm, k8sv1.EventTypeWarning, ResumeFailedReason,
			message+". Start the VirtualMachine to retry, or discard the saved state to boot it")
	}
	state.Phase = v1.HibernationRestoreFailed
	state.Message = message
}

func (c *Controller) fail(vm *v1.VirtualMachine, message string) {
	state := vm.Status.SavedState
	if state.Phase != v1.HibernationFailed {
		c.recorder.Event(vm, k8sv1.EventTypeWarning, HibernationFailedReason, message)
	}
	state.Phase = v1.HibernationFailed
	state.Message = message
}

// PrepareRestore makes vmi, about to be created for vm, resume from the saved
// state of vm if there is a valid one. An invalid saved state is discarded and
// vmi boots. The phase moves to Restoring only once vmi was created, see
// RestoreCreated: a failed creation keeps the saved state.
func (c *Controller) PrepareRestore(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	state := vm.Status.SavedState
	if state == nil || state.Discard {
		return nil
	}
	switch state.Phase {
	case v1.HibernationSaved:
	case v1.HibernationRestoring:
		if state.RestoreStartTimestamp != nil {
			// The previous VMI committed to the restore and is gone without a
			// result: its guest may have run.
			c.fail(vm, "A previous VirtualMachineInstance stopped while resuming from the saved state, which was discarded")
			return nil
		}
		// The previous VMI never started resuming, the saved state is intact.
	default:
		return nil
	}
	if reason, err := c.invalidReason(vm, state); err != nil {
		return err
	} else if reason != "" {
		message := fmt.Sprintf("The saved state was discarded and the VirtualMachine boots: %s", reason)
		c.recorder.Event(vm, k8sv1.EventTypeWarning, SavedStateDiscardedReason, message)
		c.fail(vm, message)
		return nil
	}

	vmi.Spec.SavedStateSource = &v1.SavedStateSource{ClaimName: state.ClaimName, FileName: state.FileName}
	// Hot plugged from the start, and unplugged once resumed.
	vmi.Spec.UtilityVolumes = append(vmi.Spec.UtilityVolumes, SourceVolume(state.ClaimName))
	if len(state.SourceNodeSelector) > 0 {
		if vmi.Spec.NodeSelector == nil {
			vmi.Spec.NodeSelector = map[string]string{}
		}
		maps.Copy(vmi.Spec.NodeSelector, state.SourceNodeSelector)
	}
	if claim, err := c.getClaim(vm.Namespace, state.ClaimName); err == nil && claim != nil {
		preferNode(vmi, claim.Annotations[SourceNodeAnnotation])
	}
	return nil
}

// RestoreCreated records that vmi, created for vm, resumes from its saved state.
func (c *Controller) RestoreCreated(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	state := vm.Status.SavedState
	if state == nil || vmi.Spec.SavedStateSource == nil || vmi.Spec.SavedStateSource.ClaimName != state.ClaimName {
		return
	}
	state.Phase = v1.HibernationRestoring
	state.RestoreUID = vmi.UID
	state.SupersededUID = ""
	state.RestoreStartTimestamp = nil
	state.RestoreEndTimestamp = nil
	state.Message = ""
	c.recorder.Eventf(vm, k8sv1.EventTypeNormal, ResumingReason, "Resuming the VirtualMachine from the state saved in PVC %s", state.ClaimName)
}

func (c *Controller) claimGone(namespace, name string) bool {
	claim, err := c.getClaim(namespace, name)
	return err == nil && claim == nil
}

func durationSuffix(start, end *metav1.Time) string {
	if start == nil || end == nil {
		return ""
	}
	return fmt.Sprintf(" in %s", end.Sub(start.Time))
}

// preferNode makes vmi prefer to run on node, the node its state was saved
// on. Storage is often faster to read from there, e.g. node local volumes or
// an NFS server on that node, and the CPU matches for sure.
func preferNode(vmi *v1.VirtualMachineInstance, node string) {
	if node == "" {
		return
	}
	if vmi.Spec.Affinity == nil {
		vmi.Spec.Affinity = &k8sv1.Affinity{}
	}
	if vmi.Spec.Affinity.NodeAffinity == nil {
		vmi.Spec.Affinity.NodeAffinity = &k8sv1.NodeAffinity{}
	}
	nodeAffinity := vmi.Spec.Affinity.NodeAffinity
	nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		k8sv1.PreferredSchedulingTerm{
			Weight: sourceNodeWeight,
			Preference: k8sv1.NodeSelectorTerm{
				MatchFields: []k8sv1.NodeSelectorRequirement{{
					Key:      "metadata.name",
					Operator: k8sv1.NodeSelectorOpIn,
					Values:   []string{node},
				}},
			},
		})
}

const sourceNodeWeight = 100

// invalidReason returns why the saved state cannot be resumed, if it cannot.
func (c *Controller) invalidReason(vm *v1.VirtualMachine, state *v1.VirtualMachineSavedState) (string, error) {
	if state.FileName == "" || state.SpecHash == "" {
		return "the saved state is incomplete", nil
	}
	claim, err := c.getClaim(vm.Namespace, state.ClaimName)
	if err != nil {
		return "", err
	}
	if claim == nil || claim.DeletionTimestamp != nil {
		return fmt.Sprintf("PVC %s holding the saved state is gone", state.ClaimName), nil
	}
	if !metav1.IsControlledBy(claim, vm) || claim.Labels[v1.AppLabel] != appLabelValue {
		// Only a PVC the VM controller provisioned for this VM holds its state.
		return fmt.Sprintf("PVC %s is not a saved state PVC of the VirtualMachine", state.ClaimName), nil
	}
	hash, err := SpecHash(vm)
	if err != nil {
		return "", err
	}
	if hash != state.SpecHash {
		return "the VirtualMachine spec changed since its state was saved", nil
	}
	uids, err := VolumeClaimUIDs(vm.Namespace, templateVolumes(vm), c.pvcStore)
	if err != nil {
		return "", err
	}
	if !maps.Equal(uids, state.VolumeClaimUIDs) {
		return "the PVCs of the VirtualMachine disks changed since its state was saved", nil
	}
	return "", nil
}
