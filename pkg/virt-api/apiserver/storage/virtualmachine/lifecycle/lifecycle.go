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

package lifecycle

import (
	"context"
	"fmt"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	vmNotRunning                             = "VM is not running"
	patchingVMFmt                            = "Patching VM: %s"
	patchingVMStatusFmt                      = "Patching VM status: %s"
	jsonpatchTestErr                         = "jsonpatch test operation does not apply"
	volumeMigrationManualRecoveryRequiredErr = "VM recovery required: Volume migration failed, leaving some volumes pointing to non-consistent targets; manual intervention is needed to reassign them to their original volumes."
)

// Handler contains the VirtualMachine lifecycle operations served by the
// GenericAPIServer storage Connecters.
type Handler struct {
	virtCli kubecli.KubevirtClient
}

func NewHandler(virtCli kubecli.KubevirtClient) *Handler {
	return &Handler{virtCli: virtCli}
}

func (h *Handler) StartVM(ctx context.Context, namespace, name string, startOptions *v1.StartOptions) *errors.StatusError {
	vm, err := h.virtCli.VirtualMachine(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return errors.NewInternalError(err)
	}

	if vmi != nil && !vmi.IsFinal() && vmi.Status.Phase != v1.Unknown && vmi.Status.Phase != v1.VmPhaseUnset {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is already running"))
	}
	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm, v1.VirtualMachineManualRecoveryRequired, k8sv1.ConditionTrue) {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(volumeMigrationManualRecoveryRequiredErr))
	}

	startPaused := startOptions.Paused
	startChangeRequestData := make(map[string]string)
	if startPaused {
		startChangeRequestData[v1.StartRequestDataPausedKey] = v1.StartRequestDataPausedTrue
	}

	var patchErr error
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return errors.NewInternalError(err)
	}

	switch runStrategy {
	case v1.RunStrategyHalted:
		pausedStartStrategy := v1.StartStrategyPaused
		if startPaused && (vm.Spec.Template == nil || vm.Spec.Template.Spec.StartStrategy != &pausedStartStrategy) {
			patchBytes, err := getChangeRequestJSON(vm, v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
				Data:   startChangeRequestData,
			})
			if err != nil {
				return errors.NewInternalError(err)
			}
			log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
			_, patchErr = h.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: startOptions.DryRun})
		} else {
			patchBytes, err := getRunningPatch(vm, true)
			if err != nil {
				return errors.NewInternalError(err)
			}
			log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
			_, patchErr = h.virtCli.VirtualMachine(namespace).Patch(ctx, vm.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: startOptions.DryRun})
		}
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		needsRestart := false
		if (runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Succeeded) ||
			(runStrategy == v1.RunStrategyManual && vmi != nil && vmi.IsFinal()) {
			needsRestart = true
		} else if runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Failed {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support starting VM from failed state", v1.RunStrategyRerunOnFailure))
		}

		var patchBytes []byte
		if needsRestart {
			patchBytes, err = getChangeRequestJSON(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		} else {
			patchBytes, err = getChangeRequestJSON(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		}
		if err != nil {
			return errors.NewInternalError(err)
		}
		log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
		_, patchErr = h.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: startOptions.DryRun})
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support manual start requests", runStrategy))
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr)
		}
		return errors.NewInternalError(patchErr)
	}
	return nil
}

func (h *Handler) StopVM(ctx context.Context, namespace, name string, stopOptions *v1.StopOptions) *errors.StatusError {
	vm, statusErr := h.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		return statusErr
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return errors.NewInternalError(err)
	}

	hasVMI := true
	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		hasVMI = false
	} else if err != nil {
		return errors.NewInternalError(err)
	}

	var oldGracePeriodSeconds int64
	var patchErr error
	if hasVMI && !vmi.IsFinal() && stopOptions.GracePeriod != nil {
		oldGracePeriodSeconds, err = h.patchVMITerminationGracePeriod(ctx, vmi, namespace, *stopOptions.GracePeriod, stopOptions.DryRun)
		if err != nil {
			return errors.NewInternalError(err)
		}
	}

	switch runStrategy {
	case v1.RunStrategyHalted:
		if !hasVMI || vmi.IsFinal() {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning))
		}
		if stopOptions.GracePeriod == nil || (vmi.Spec.TerminationGracePeriodSeconds != nil && *stopOptions.GracePeriod >= oldGracePeriodSeconds) {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v only supports manual stop requests with a shorter graceperiod", v1.RunStrategyHalted))
		}
		patchErr = h.patchVMStatusStopped(ctx, vmi, vm, stopOptions)
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		if !hasVMI || vmi.IsFinal() {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning))
		}
		patchErr = h.patchVMStatusStopped(ctx, vmi, vm, stopOptions)
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		patchBytes, err := getRunningPatch(vm, false)
		if err != nil {
			return errors.NewInternalError(err)
		}
		log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
		_, patchErr = h.virtCli.VirtualMachine(namespace).Patch(ctx, vm.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: stopOptions.DryRun})
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr)
		}
		return errors.NewInternalError(patchErr)
	}
	return nil
}

func (h *Handler) RestartVM(ctx context.Context, namespace, name string, restartOptions *v1.RestartOptions) *errors.StatusError {
	if restartOptions.GracePeriodSeconds != nil {
		if *restartOptions.GracePeriodSeconds > 0 {
			return errors.NewBadRequest(fmt.Sprintf("For force restart, only gracePeriod=0 is supported for now"))
		} else if *restartOptions.GracePeriodSeconds < 0 {
			return errors.NewBadRequest(fmt.Sprintf("gracePeriod has to be greater or equal to 0"))
		}
	}

	vm, statusErr := h.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		return statusErr
	}
	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm,
		v1.VirtualMachineConditionType(v1.VirtualMachineInstanceVolumesChange), k8sv1.ConditionTrue) {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(volumeMigrationManualRecoveryRequiredErr))
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return errors.NewInternalError(err)
	}
	if runStrategy == v1.RunStrategyHalted || runStrategy == v1.RunStrategyOnce {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("RunStategy %v does not support manual restart requests", runStrategy))
	}

	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return errors.NewInternalError(err)
		}
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running: %v", v1.RunStrategyHalted))
	}

	forceRestart := restartOptions.GracePeriodSeconds != nil && *restartOptions.GracePeriodSeconds == 0
	var oldGracePeriodSeconds int64
	if forceRestart {
		oldGracePeriodSeconds, err = h.patchVMITerminationGracePeriod(ctx, vmi, namespace, int64(1), restartOptions.DryRun)
		if err != nil {
			return errors.NewInternalError(err)
		}
		newGracePeriod := int64(1)
		vmi.Spec.TerminationGracePeriodSeconds = &newGracePeriod
	}

	patchBytes, err := getChangeRequestJSON(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
		v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
	if err != nil {
		return errors.NewInternalError(err)
	}

	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	_, err = h.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: restartOptions.DryRun})
	if err != nil {
		if forceRestart {
			if _, rollbackErr := h.patchVMITerminationGracePeriod(ctx, vmi, namespace, oldGracePeriodSeconds, restartOptions.DryRun); rollbackErr != nil {
				log.Log.Object(vmi).Errorf("Failed to rollback VMI terminationGracePeriodSeconds: %v", rollbackErr)
			}
		}
		if strings.Contains(err.Error(), jsonpatchTestErr) {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
		}
		return errors.NewInternalError(err)
	}
	return nil
}

func (h *Handler) MigrateVM(ctx context.Context, namespace, name string, migrateOptions *v1.MigrateOptions) *errors.StatusError {
	if _, statusErr := h.fetchVirtualMachine(name, namespace); statusErr != nil {
		return statusErr
	}

	vmi, statusErr := h.fetchVirtualMachineInstance(namespace, name)
	if statusErr != nil {
		return statusErr
	}
	if vmi.Status.Phase != v1.Running {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning))
	}

	_, err := h.virtCli.VirtualMachineInstanceMigration(namespace).Create(ctx, &v1.VirtualMachineInstanceMigration{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "kubevirt-migrate-vm-",
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName:           name,
			AddedNodeSelector: migrateOptions.AddedNodeSelector,
		},
	}, metav1.CreateOptions{DryRun: migrateOptions.DryRun})
	if err != nil {
		return errors.NewInternalError(err)
	}
	return nil
}

func (h *Handler) fetchVirtualMachine(name, namespace string) (*v1.VirtualMachine, *errors.StatusError) {
	vm, err := h.virtCli.VirtualMachine(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}
	return vm, nil
}

func (h *Handler) fetchVirtualMachineInstance(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}
	return vmi, nil
}

func (h *Handler) patchVMITerminationGracePeriod(ctx context.Context, vmi *v1.VirtualMachineInstance, namespace string, gracePeriod int64, dryRun []string) (int64, error) {
	var oldGracePeriod int64
	patchSet := patch.New()
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		oldGracePeriod = *vmi.Spec.TerminationGracePeriodSeconds
		patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", *vmi.Spec.TerminationGracePeriodSeconds))
	} else {
		patchSet.AddOption(patch.WithTest("/spec/terminationGracePeriodSeconds", nil))
	}
	patchSet.AddOption(patch.WithReplace("/spec/terminationGracePeriodSeconds", gracePeriod))
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return 0, err
	}
	log.Log.Object(vmi).V(2).Infof("Patching VMI terminationGracePeriodSeconds: %s", string(patchBytes))
	_, err = h.virtCli.VirtualMachineInstance(namespace).Patch(ctx, vmi.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRun})
	return oldGracePeriod, err
}

func (h *Handler) patchVMStatusStopped(ctx context.Context, vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine, stopOptions *v1.StopOptions) error {
	patchBytes, err := getChangeRequestJSON(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
	if err != nil {
		return err
	}
	log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
	_, err = h.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: stopOptions.DryRun})
	return err
}

func getChangeRequestJSON(vm *v1.VirtualMachine, changes ...v1.VirtualMachineStateChangeRequest) ([]byte, error) {
	patchSet := patch.New()
	newStatus := v1.VirtualMachineStatus{}
	if equality.Semantic.DeepEqual(vm.Status, newStatus) {
		newStatus.StateChangeRequests = append(newStatus.StateChangeRequests, changes...)
		patchSet.AddOption(patch.WithAdd("/status", newStatus))
	} else {
		patchSet.AddOption(patch.WithTest("/status/stateChangeRequests", vm.Status.StateChangeRequests))
		switch {
		case len(vm.Status.StateChangeRequests) == 0:
			patchSet.AddOption(patch.WithAdd("/status/stateChangeRequests", changes))
		case len(changes) == 1 && changes[0].Action == v1.StopRequest:
			patchSet.AddOption(patch.WithReplace("/status/stateChangeRequests", changes))
		default:
			return nil, fmt.Errorf("unable to complete request: stop/start already underway")
		}
	}

	if vm.Status.StartFailure != nil {
		patchSet.AddOption(patch.WithRemove("/status/startFailure"))
	}
	return patchSet.GeneratePayload()
}

func getRunningPatch(vm *v1.VirtualMachine, running bool) ([]byte, error) {
	runStrategy := v1.RunStrategyHalted
	if running {
		runStrategy = v1.RunStrategyAlways
	}

	if vm.Spec.RunStrategy != nil {
		return patch.New(
			patch.WithTest("/spec/runStrategy", vm.Spec.RunStrategy),
			patch.WithReplace("/spec/runStrategy", runStrategy),
		).GeneratePayload()
	}

	return patch.New(
		patch.WithTest("/spec/running", vm.Spec.Running),
		patch.WithReplace("/spec/running", running),
	).GeneratePayload()
}
