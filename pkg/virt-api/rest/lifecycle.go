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
	"io"
	"net/http"
	"strings"

	"github.com/emicklei/go-restful/v3"

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

func (app *SubresourceAPIApp) StartVM(ctx context.Context, namespace, name string, startOptions *v1.StartOptions) *errors.StatusError {
	vm, err := app.virtCli.VirtualMachine(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
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
	// RunStrategyHalted         -> spec.running = true / send start request for paused start
	// RunStrategyManual         -> send start request
	// RunStrategyAlways         -> doesn't make sense
	// RunStrategyRerunOnFailure -> doesn't make sense
	// RunStrategyOnce           -> doesn't make sense
	switch runStrategy {
	case v1.RunStrategyHalted:
		pausedStartStrategy := v1.StartStrategyPaused
		if startPaused && (vm.Spec.Template == nil || vm.Spec.Template.Spec.StartStrategy != &pausedStartStrategy) {
			patchBytes, err := getChangeRequestJson(vm, v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
				Data:   startChangeRequestData,
			})
			if err != nil {
				return errors.NewInternalError(err)
			}
			log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
			_, patchErr = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: startOptions.DryRun})
		} else {
			patchBytes, err := getRunningPatch(vm, true)
			if err != nil {
				return errors.NewInternalError(err)
			}
			log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
			_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(ctx, vm.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: startOptions.DryRun})
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
			patchBytes, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		} else {
			patchBytes, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		}
		if err != nil {
			return errors.NewInternalError(err)
		}
		log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
		_, patchErr = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: startOptions.DryRun})
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

func (app *SubresourceAPIApp) StartVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	startOptions := &v1.StartOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, startOptions); err != nil {
			writeError(err, response)
			return
		}
	}

	if statusErr := app.StartVM(request.Request.Context(), namespace, name, startOptions); statusErr != nil {
		writeError(statusErr, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) StopVM(ctx context.Context, namespace, name string, stopOptions *v1.StopOptions) *errors.StatusError {
	// RunStrategyHalted         -> force stop if grace period in request is shorter than before, otherwise doesn't make sense
	// RunStrategyManual         -> send stop request
	// RunStrategyAlways         -> spec.running = false
	// RunStrategyRerunOnFailure -> send stop request
	// RunStrategyOnce           -> spec.running = false

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		return statusErr
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		return errors.NewInternalError(err)
	}

	hasVMI := true
	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		hasVMI = false
	} else if err != nil {
		return errors.NewInternalError(err)
	}

	var oldGracePeriodSeconds int64
	var patchErr error
	if hasVMI && !vmi.IsFinal() && stopOptions.GracePeriod != nil {
		var err error
		oldGracePeriodSeconds, err = app.patchVMITerminationGracePeriod(ctx, vmi, namespace, *stopOptions.GracePeriod, stopOptions.DryRun)
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
		// same behavior as RunStrategyManual
		patchErr = app.patchVMStatusStopped(ctx, vmi, vm, stopOptions)
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		if !hasVMI || vmi.IsFinal() {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning))
		}
		// pass the buck and ask virt-controller to stop the VM. this way the
		// VM will retain RunStrategy = manual
		patchErr = app.patchVMStatusStopped(ctx, vmi, vm, stopOptions)
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		patchBytes, err := getRunningPatch(vm, false)
		if err != nil {
			return errors.NewInternalError(err)
		}
		log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
		_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(ctx, vm.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: stopOptions.DryRun})
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			return errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr)
		}
		return errors.NewInternalError(patchErr)
	}

	return nil
}

func (app *SubresourceAPIApp) StopVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.StopOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}

	if statusErr := app.StopVM(request.Request.Context(), namespace, name, bodyStruct); statusErr != nil {
		writeError(statusErr, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

// PauseVMI pauses a running VMI. The optional PauseOptions body is decoded to
// detect a dry-run request before the body is proxied to virt-handler
func (app *SubresourceAPIApp) PauseVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		if vmi.Spec.LivenessProbe != nil && vmi.Spec.LivenessProbe.GuestAgentPing == nil {
			return errors.NewForbidden(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Pausing VMIs with a non-GuestAgentPing LivenessProbe is not supported"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is already paused"))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.PauseURI(vmi)
	}

	bodyStruct := &v1.PauseOptions{}
	if body != nil {
		if err := decodeBodyReader(body, bodyStruct); err != nil {
			return err
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == metav1.DryRunAll {
		dryRun = true
	}
	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, dryRun)
}

func (app *SubresourceAPIApp) PauseVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.PauseVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// UnpauseVMI resumes a paused VMI. It first ensures the owning VM (if any) has
// no snapshot in progress then proxies the request to virt-handler
func (app *SubresourceAPIApp) UnpauseVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	// Check VM status - only continue if VM doesn't exist or if it exists without snapshot in progress
	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		if !errors.IsNotFound(statusErr) {
			return statusErr
		}
	} else if vm.Status.SnapshotInProgress != nil {
		// VM exists - check if snapshot is in progress
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmSnapshotInprogress))
	}

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotPaused))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnpauseURI(vmi)
	}

	bodyStruct := &v1.UnpauseOptions{}
	if body != nil {
		if err := decodeBodyReader(body, bodyStruct); err != nil {
			return err
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == metav1.DryRunAll {
		dryRun = true
	}
	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, dryRun)
}

func (app *SubresourceAPIApp) UnpauseVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.UnpauseVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// FreezeVMI freezes the filesystems of a running VMI
// The request body is proxied unchanged to virt-handler
func (app *SubresourceAPIApp) FreezeVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FreezeURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) FreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.FreezeVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// UnfreezeVMI thaws the filesystems of a running VMI.
func (app *SubresourceAPIApp) UnfreezeVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnfreezeURI(vmi)
	}
	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) UnfreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.UnfreezeVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

// ResetVMI triggers a reset of a running VMI.
func (app *SubresourceAPIApp) ResetVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	// Post process any error responses in order to append human
	// readable explanation for why the reset may have failed.
	errorPostProcessing := func(vmi *v1.VirtualMachineInstance, err error) error {
		// VMI reset request could have been sent while VMI was in the process of transitioning
		// from scheduled to running.
		if vmi != nil && !vmi.IsRunning() {
			return fmt.Errorf("Failed to reset non-running VMI with phase %s: %v", vmi.Status.Phase, err)
		}
		return err
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.ResetURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, nil, errorPostProcessing, getURL, false)
}

func (app *SubresourceAPIApp) ResetVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.ResetVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

func (app *SubresourceAPIApp) RestartVM(ctx context.Context, namespace, name string, restartOptions *v1.RestartOptions) *errors.StatusError {
	// RunStrategyHalted         -> doesn't make sense
	// RunStrategyManual         -> send restart request
	// RunStrategyAlways         -> send restart request
	// RunStrategyRerunOnFailure -> send restart request
	// RunStrategyOnce           -> doesn't make sense

	if restartOptions.GracePeriodSeconds != nil {
		if *restartOptions.GracePeriodSeconds > 0 {
			return errors.NewBadRequest(fmt.Sprintf("For force restart, only gracePeriod=0 is supported for now"))
		} else if *restartOptions.GracePeriodSeconds < 0 {
			return errors.NewBadRequest(fmt.Sprintf("gracePeriod has to be greater or equal to 0"))
		}
	}

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
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

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return errors.NewInternalError(err)
		}
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running: %v", v1.RunStrategyHalted))
	}

	// Set terminationGracePeriodSeconds to 1 (the shortest safe restart period) before
	// sending stateChangeRequests, so virt-handler shuts down the guest promptly regardless
	// of which controller deletes the pod first.
	forceRestart := restartOptions.GracePeriodSeconds != nil && *restartOptions.GracePeriodSeconds == 0
	var oldGracePeriodSeconds int64
	if forceRestart {
		var err error
		oldGracePeriodSeconds, err = app.patchVMITerminationGracePeriod(ctx, vmi, namespace, int64(1), restartOptions.DryRun)
		if err != nil {
			return errors.NewInternalError(err)
		}
		// Reflect the patched value locally so the rollback patch uses the correct
		// optimistic concurrency test if PatchStatus fails below.
		newGracePeriod := int64(1)
		vmi.Spec.TerminationGracePeriodSeconds = &newGracePeriod
	}

	patchBytes, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
		v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
	if err != nil {
		return errors.NewInternalError(err)
	}

	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	_, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: restartOptions.DryRun})
	if err != nil {
		if forceRestart {
			if _, rollbackErr := app.patchVMITerminationGracePeriod(ctx, vmi, namespace, oldGracePeriodSeconds, restartOptions.DryRun); rollbackErr != nil {
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

func (app *SubresourceAPIApp) RestartVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.RestartOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}

	if statusErr := app.RestartVM(request.Request.Context(), namespace, name, bodyStruct); statusErr != nil {
		writeError(statusErr, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

// SoftRebootVMI issues an ACPI soft reboot of a running VMI
func (app *SubresourceAPIApp) SoftRebootVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if condManager.HasConditionWithStatus(vmi, v1.VirtualMachineInstancePaused, k8sv1.ConditionTrue) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is paused"))
		}
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			if features := vmi.Spec.Domain.Features; features != nil && features.ACPI.Enabled != nil && !(*features.ACPI.Enabled) {
				return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI neither have the agent connected nor the ACPI feature enabled"))
			}
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SoftRebootURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) SoftRebootVMIRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	if statusErr := app.SoftRebootVMI(request.Request.Context(), namespace, name, request.Request.Body); statusErr != nil {
		writeError(statusErr, response)
	}
}

func (app *SubresourceAPIApp) MigrateVM(ctx context.Context, namespace, name string, migrateOptions *v1.MigrateOptions) *errors.StatusError {
	if _, statusErr := app.fetchVirtualMachine(name, namespace); statusErr != nil {
		return statusErr
	}

	vmi, statusErr := app.FetchVirtualMachineInstance(namespace, name)
	if statusErr != nil {
		return statusErr
	}

	if vmi.Status.Phase != v1.Running {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning))
	}

	_, err := app.virtCli.VirtualMachineInstanceMigration(namespace).Create(ctx, &v1.VirtualMachineInstanceMigration{
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

func (app *SubresourceAPIApp) MigrateVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.MigrateOptions{}
	if request.Request.Body != nil {
		if err := decodeBody(request, bodyStruct); err != nil {
			writeError(err, response)
			return
		}
	}

	if statusErr := app.MigrateVM(request.Request.Context(), namespace, name, bodyStruct); statusErr != nil {
		writeError(statusErr, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) patchVMITerminationGracePeriod(ctx context.Context, vmi *v1.VirtualMachineInstance, namespace string, gracePeriod int64, dryRun []string) (int64, error) {
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
	_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(ctx, vmi.GetName(), types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: dryRun})
	return oldGracePeriod, err
}

func (app *SubresourceAPIApp) patchVMStatusStopped(ctx context.Context, vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine, stopOptions *v1.StopOptions) error {
	patchBytes, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
	if err != nil {
		return err
	}
	log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
	_, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(ctx, vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{DryRun: stopOptions.DryRun})
	return err
}

func getChangeRequestJson(vm *v1.VirtualMachine, changes ...v1.VirtualMachineStateChangeRequest) ([]byte, error) {
	patchSet := patch.New()
	// Special case: if there's no status field at all, add one.
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
			// If this is a stopRequest, replace all existing StateChangeRequests.
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

func (app *SubresourceAPIApp) BackupVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.BackupURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) RedefineCheckpointVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.ChangedBlockTracking == nil ||
			vmi.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name,
				fmt.Errorf("ChangedBlockTracking is not enabled"))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.RedefineCheckpointURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}
