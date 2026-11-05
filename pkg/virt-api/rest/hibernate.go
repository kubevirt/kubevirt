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

package rest

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/hibernation"
)

const vmHibernationGateDisabled = "VMHibernation feature gate is not enabled"

// HibernateVMRequestHandler saves the memory and device state of a running
// VirtualMachine and stops it. The next start resumes from the saved state.
//
// RunStrategyAlways, RunStrategyOnce -> runStrategy = Halted
// RunStrategyManual, RunStrategyRerunOnFailure -> stop request
// The VM controller holds the stop until the state is saved.
func (app *SubresourceAPIApp) HibernateVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.VMHibernationEnabled() {
		writeError(errors.NewBadRequest(vmHibernationGateDisabled), response)
		return
	}
	// The state is saved to a hot plugged utility volume.
	if !app.clusterConfig.UtilityVolumesEnabled() {
		writeError(errors.NewBadRequest("VMHibernation requires the UtilityVolumes feature gate"), response)
		return
	}

	options := &v1.HibernateOptions{}
	if request.Request.Body != nil {
		defer request.Request.Body.Close()
		if err := decodeBody(request, options); err != nil {
			writeError(err, response)
			return
		}
	}

	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		writeError(statErr, response)
		return
	}
	vmi, statErr := app.FetchVirtualMachineInstanceForVM(namespace, name)
	if statErr != nil {
		writeError(statErr, response)
		return
	}
	if err := hibernation.ValidateRequest(vm, vmi); err != nil {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, err), response)
		return
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	state := hibernation.NewSavedState(vm, vmi, options)
	statusPatch := savedStatePatch(vm, state)

	var specPatch []byte
	switch runStrategy {
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		if specPatch, err = getRunningPatch(vm, false); err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
	case v1.RunStrategyManual, v1.RunStrategyRerunOnFailure:
		// The stop request is part of the status patch, so that the hibernation
		// and the stop are recorded together.
		addStopRequest(statusPatch, vm, hibernation.StopRequest(vmi, state.ClaimName))
	default:
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name,
			fmt.Errorf("hibernation is not supported with runStrategy %s", runStrategy)), response)
		return
	}

	patchBytes, err := statusPatch.GeneratePayload()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
	if _, err := app.virtClient.VirtualMachine(namespace).PatchStatus(context.Background(), name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: options.DryRun}); err != nil {
		writePatchError(name, err, response)
		return
	}

	if specPatch != nil {
		log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(specPatch))
		_, err := app.virtClient.VirtualMachine(namespace).Patch(context.Background(), name, types.JSONPatchType, specPatch, metav1.PatchOptions{DryRun: options.DryRun})
		if err != nil {
			// Without the stop the saved state would be resumed right away,
			// withdraw the request. The VM controller also cancels a request
			// whose stop does not follow.
			app.withdrawHibernation(vm, state, options.DryRun)
			writePatchError(name, err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

// DiscardSavedStateVMRequestHandler deletes the saved state of a hibernated
// VirtualMachine. The next start boots the VirtualMachine.
//
// It also cancels a hibernation that is still Pending, i.e. whose save did
// not start yet: the VirtualMachine keeps running with its previous run
// strategy. After a failed restore, it lets the VirtualMachine boot. While
// Restoring, it stops a VirtualMachineInstance that did not start resuming
// yet, e.g. because no node can resume the state, and the VirtualMachine
// boots instead; once the state is being loaded, the restore completes first.
func (app *SubresourceAPIApp) DiscardSavedStateVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		writeError(statErr, response)
		return
	}
	state := vm.Status.SavedState
	if state == nil {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VirtualMachine %s has no saved state", name)), response)
		return
	}
	if err := app.discardSavedState(vm, nil); err != nil {
		writeError(err, response)
		return
	}
	response.WriteHeader(http.StatusAccepted)
}

// discardSavedState requests the saved state of vm to be discarded, or its
// Pending hibernation to be cancelled.
func (app *SubresourceAPIApp) discardSavedState(vm *v1.VirtualMachine, dryRun []string) *errors.StatusError {
	state := vm.Status.SavedState
	switch state.Phase {
	case v1.HibernationPending, v1.HibernationSaved, v1.HibernationRestoring, v1.HibernationRestoreFailed,
		v1.HibernationFailed, v1.HibernationRestored:
	case v1.HibernationSaving:
		return errors.NewConflict(v1.Resource("virtualmachine"), vm.Name,
			fmt.Errorf("the state of VirtualMachine %s is being saved, the hibernation cannot be cancelled anymore", vm.Name))
	default:
		return errors.NewConflict(v1.Resource("virtualmachine"), vm.Name,
			fmt.Errorf("the saved state of VirtualMachine %s cannot be discarded in phase %s", vm.Name, state.Phase))
	}
	if state.Discard {
		return nil
	}

	patchBytes, err := patch.New(
		patch.WithTest("/status/savedState/phase", state.Phase),
		patch.WithTest("/status/savedState/claimName", state.ClaimName),
		patch.WithAdd("/status/savedState/discard", true),
	).GeneratePayload()
	if err != nil {
		return errors.NewInternalError(err)
	}
	if _, err := app.virtClient.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRun}); err != nil {
		return patchError(vm.Name, err)
	}
	return nil
}

// startDuringHibernation handles a start request for vm while it hibernates
// or after its resume failed. It reports whether it handled the request.
//
// Starting a VirtualMachine whose hibernation is Pending cancels it, so that
// the VirtualMachine keeps running with its previous run strategy. Starting
// one whose resume failed retries the resume once: the saved state goes back
// to Saved, the VM controller replaces the VMI that failed to resume, which
// the retry records as superseded, and a
// VirtualMachine whose run strategy asks for it to run (Always,
// RerunOnFailure, and Once, whose single run never happened) starts on its
// own; a Halted or Manual one starts as usual.
func (app *SubresourceAPIApp) startDuringHibernation(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, dryRun []string) (bool, *errors.StatusError) {
	state := vm.Status.SavedState
	if state == nil || state.Discard {
		return false, nil
	}
	if state.Phase == v1.HibernationRestoreFailed {
		if err := app.retryRestore(vm, vmi, dryRun); err != nil {
			return true, err
		}
		runStrategy, err := vm.RunStrategy()
		if err != nil {
			return true, errors.NewInternalError(err)
		}
		switch runStrategy {
		case v1.RunStrategyAlways, v1.RunStrategyRerunOnFailure, v1.RunStrategyOnce:
			return true, nil
		}
		return false, nil
	}
	if vmi == nil || vmi.UID != state.SourceUID || vmi.IsFinal() {
		return false, nil
	}
	switch state.Phase {
	case v1.HibernationPending:
		return true, app.discardSavedState(vm, dryRun)
	case v1.HibernationSaving:
		return true, errors.NewConflict(v1.Resource("virtualmachine"), vm.Name,
			fmt.Errorf("the state of VirtualMachine %s is being saved, start it once it is hibernated to resume it", vm.Name))
	}
	return false, nil
}

// retryRestore moves a saved state whose resume failed back to Saved, so that
// the next start resumes from it again. The guest never ran, the saved state
// still matches the disks. vmi, the current VMI of vm if any, is recorded as
// superseded if it failed to resume: the VM controller replaces only a VMI a
// retry superseded, and records the failure of any other one.
func (app *SubresourceAPIApp) retryRestore(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, dryRun []string) *errors.StatusError {
	state := vm.Status.SavedState
	patchSet := patch.New(
		patch.WithTest("/status/savedState/phase", v1.HibernationRestoreFailed),
		patch.WithTest("/status/savedState/claimName", state.ClaimName),
		patch.WithReplace("/status/savedState/phase", v1.HibernationSaved),
		patch.WithAdd("/status/savedState/message", "Retrying to resume from the saved state after: "+state.Message),
	)
	if state.RestoreUID != "" {
		// The next restore VMI is recorded by its own UID.
		patchSet.AddOption(
			patch.WithTest("/status/savedState/restoreUID", state.RestoreUID),
			patch.WithRemove("/status/savedState/restoreUID"),
		)
	}
	if hibernation.SupersedesRestore(vm, vmi) {
		patchSet.AddOption(patch.WithAdd("/status/savedState/supersededUID", vmi.UID))
	}
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return errors.NewInternalError(err)
	}
	if _, err := app.virtClient.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRun}); err != nil {
		return patchError(vm.Name, err)
	}
	return nil
}

// stopDuringHibernation refuses to stop vm while its state is being saved:
// it stops once its state is saved, and a discard cancels a hibernation that
// did not start saving.
func stopDuringHibernation(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if state := vm.Status.SavedState; state == nil || state.Discard || !hibernation.IsHibernating(vm, vmi) {
		return nil
	}
	return errors.NewConflict(v1.Resource("virtualmachine"), vm.Name,
		fmt.Errorf("VirtualMachine %s is hibernating: it stops once its state is saved, discard the saved state to cancel the hibernation", vm.Name))
}

// startRunStrategy returns the run strategy a start sets on a Halted vm: the
// one the VM had when it was hibernated, if it was hibernated, e.g. Once
// rather than Always.
func startRunStrategy(vm *v1.VirtualMachine) v1.VirtualMachineRunStrategy {
	if state := vm.Status.SavedState; state != nil && !state.Discard && state.RunStrategy != nil && *state.RunStrategy == v1.RunStrategyOnce {
		switch state.Phase {
		case v1.HibernationSaved, v1.HibernationRestoreFailed:
			return v1.RunStrategyOnce
		}
	}
	return v1.RunStrategyAlways
}

func savedStatePatch(vm *v1.VirtualMachine, state *v1.VirtualMachineSavedState) *patch.PatchSet {
	patchSet := patch.New(patch.WithTest("/status/savedState", vm.Status.SavedState))
	if vm.Status.SavedState == nil {
		patchSet.AddOption(patch.WithAdd("/status/savedState", state))
	} else {
		patchSet.AddOption(patch.WithReplace("/status/savedState", state))
	}
	return patchSet
}

func addStopRequest(patchSet *patch.PatchSet, vm *v1.VirtualMachine, request v1.VirtualMachineStateChangeRequest) {
	stop := []v1.VirtualMachineStateChangeRequest{request}
	patchSet.AddOption(patch.WithTest("/status/stateChangeRequests", vm.Status.StateChangeRequests))
	if len(vm.Status.StateChangeRequests) == 0 {
		patchSet.AddOption(patch.WithAdd("/status/stateChangeRequests", stop))
	} else {
		// Like a stop, replace any pending state change request.
		patchSet.AddOption(patch.WithReplace("/status/stateChangeRequests", stop))
	}
}

// withdrawHibernation cancels the hibernation request of state, which the VM
// controller may have extended already, as long as it is Pending: the VM
// controller then cancels it like a discard.
func (app *SubresourceAPIApp) withdrawHibernation(vm *v1.VirtualMachine, state *v1.VirtualMachineSavedState, dryRun []string) {
	patchBytes, err := patch.New(
		patch.WithTest("/status/savedState/phase", v1.HibernationPending),
		patch.WithTest("/status/savedState/claimName", state.ClaimName),
		patch.WithTest("/status/savedState/sourceUID", state.SourceUID),
		patch.WithAdd("/status/savedState/discard", true),
	).GeneratePayload()
	if err == nil {
		_, err = app.virtClient.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRun})
	}
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Failed to withdraw the hibernation request")
	}
}

func writePatchError(name string, err error, response *restful.Response) {
	writeError(patchError(name, err), response)
}

func patchError(name string, err error) *errors.StatusError {
	if strings.Contains(err.Error(), jsonpatchTestErr) {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}
	if statusErr, ok := err.(*errors.StatusError); ok && errors.IsInvalid(err) {
		return statusErr
	}
	return errors.NewInternalError(err)
}
